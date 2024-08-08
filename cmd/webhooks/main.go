//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	"github.com/MainfluxLabs/mainflux/webhooks"
	"github.com/MainfluxLabs/mainflux/webhooks/api"
	httpapi "github.com/MainfluxLabs/mainflux/webhooks/api/http"
	"github.com/MainfluxLabs/mainflux/webhooks/postgres"
	"github.com/MainfluxLabs/mainflux/webhooks/tracing"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "webhooks"
	stopWaitTime = 5 * time.Second

	defBrokerURL         = "nats://localhost:4222"
	defLogLevel          = "error"
	defDBHost            = "localhost"
	defDBPort            = "5432"
	defDBUser            = "mainflux"
	defDBPass            = "mainflux"
	defDB                = "webhooks"
	defDBSSLMode         = "disable"
	defDBSSLCert         = ""
	defDBSSLKey          = ""
	defDBSSLRootCert     = ""
	defClientTLS         = "false"
	defCACerts           = ""
	defHTTPPort          = "9021"
	defJaegerURL         = ""
	defServerCert        = ""
	defServerKey         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"

	envBrokerURL         = "MF_BROKER_URL"
	envLogLevel          = "MF_WEBHOOKS_LOG_LEVEL"
	envDBHost            = "MF_WEBHOOKS_DB_HOST"
	envDBPort            = "MF_WEBHOOKS_DB_PORT"
	envDBUser            = "MF_WEBHOOKS_DB_USER"
	envDBPass            = "MF_WEBHOOKS_DB_PASS"
	envDB                = "MF_WEBHOOKS_DB"
	envDBSSLMode         = "MF_WEBHOOKS_DB_SSL_MODE"
	envDBSSLCert         = "MF_WEBHOOKS_DB_SSL_CERT"
	envDBSSLKey          = "MF_WEBHOOKS_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_WEBHOOKS_DB_SSL_ROOT_CERT"
	envClientTLS         = "MF_WEBHOOKS_CLIENT_TLS"
	envCACerts           = "MF_WEBHOOKS_CA_CERTS"
	envHTTPPort          = "MF_WEBHOOKS_HTTP_PORT"
	envServerCert        = "MF_WEBHOOKS_SERVER_CERT"
	envServerKey         = "MF_WEBHOOKS_SERVER_KEY"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
)

type config struct {
	brokerURL         string
	logLevel          string
	dbConfig          postgres.Config
	httpConfig        servers.Config
	thingsConfig      clients.Config
	jaegerURL         string
	thingsGRPCTimeout time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	pubSub, err := brokers.NewPubSub(cfg.brokerURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	webhooksTracer, webhooksCloser := jaeger.Init(svcName, cfg.jaegerURL, logger)
	defer webhooksCloser.Close()

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	thingsTracer, thingsCloser := jaeger.Init("webhooks_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	thingsConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer thingsConn.Close()

	things := thingsapi.NewClient(thingsConn, thingsTracer, cfg.thingsGRPCTimeout)

	dbTracer, dbCloser := jaeger.Init("webhooks_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(things, dbTracer, db, logger)

	if err = consumers.Start(svcName, pubSub, svc, brokers.SubjectWebhook); err != nil {
		logger.Error(fmt.Sprintf("Failed to create Webhook: %s", err))
	}

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(webhooksTracer, svc, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Webhooks service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Webhooks service terminated: %s", err))
	}
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	thingsAuthGRPCTimeout, err := time.ParseDuration(mainflux.Env(envThingsGRPCTimeout, defThingsGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsGRPCTimeout, err.Error())
	}

	dbConfig := postgres.Config{
		Host:        mainflux.Env(envDBHost, defDBHost),
		Port:        mainflux.Env(envDBPort, defDBPort),
		User:        mainflux.Env(envDBUser, defDBUser),
		Pass:        mainflux.Env(envDBPass, defDBPass),
		Name:        mainflux.Env(envDB, defDB),
		SSLMode:     mainflux.Env(envDBSSLMode, defDBSSLMode),
		SSLCert:     mainflux.Env(envDBSSLCert, defDBSSLCert),
		SSLKey:      mainflux.Env(envDBSSLKey, defDBSSLKey),
		SSLRootCert: mainflux.Env(envDBSSLRootCert, defDBSSLRootCert),
	}

	httpConfig := servers.Config{
		ServerName:   svcName,
		ServerCert:   mainflux.Env(envServerCert, defServerCert),
		ServerKey:    mainflux.Env(envServerKey, defServerKey),
		Port:         mainflux.Env(envHTTPPort, defHTTPPort),
		StopWaitTime: stopWaitTime,
	}

	thingsConfig := clients.Config{
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envCACerts, defCACerts),
		URL:        mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		ClientName: clients.Things,
	}

	return config{
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:          dbConfig,
		httpConfig:        httpConfig,
		thingsConfig:      thingsConfig,
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCTimeout: thingsAuthGRPCTimeout,
	}
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func newService(ts protomfx.ThingsServiceClient, dbTracer opentracing.Tracer, db *sqlx.DB, logger logger.Logger) webhooks.Service {
	database := postgres.NewDatabase(db)
	webhooksRepo := postgres.NewWebhookRepository(database)
	webhooksRepo = tracing.WebhookRepositoryMiddleware(dbTracer, webhooksRepo)
	forwarder := webhooks.NewForwarder()
	idProvider := uuid.New()

	svc := webhooks.New(ts, webhooksRepo, forwarder, idProvider)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "webhooks",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "webhooks",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}
