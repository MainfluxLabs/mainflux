// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfevents "github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	"github.com/MainfluxLabs/mainflux/shadows"
	"github.com/MainfluxLabs/mainflux/shadows/api"
	httpapi "github.com/MainfluxLabs/mainflux/shadows/api/http"
	"github.com/MainfluxLabs/mainflux/shadows/events"
	"github.com/MainfluxLabs/mainflux/shadows/postgres"
	"github.com/MainfluxLabs/mainflux/shadows/tracing"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "shadows"
	stopWaitTime = 5 * time.Second

	defLogLevel          = "error"
	defDBHost            = "localhost"
	defDBPort            = "5432"
	defDBUser            = "mainflux"
	defDBPass            = "mainflux"
	defDB                = svcName
	defDBSSLMode         = "disable"
	defDBSSLCert         = ""
	defDBSSLKey          = ""
	defDBSSLRootCert     = ""
	defClientTLS         = "false"
	defCACerts           = ""
	defHTTPPort          = "9031"
	defJaegerURL         = ""
	defServerCert        = ""
	defServerKey         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defAuthGRPCURL       = "localhost:8181"
	defAuthGRPCTimeout   = "1s"
	defBrokerURL         = "nats://localhost:4222"
	defESURL             = "redis://localhost:6379/0"

	envLogLevel          = "MF_SHADOWS_LOG_LEVEL"
	envDBHost            = "MF_SHADOWS_DB_HOST"
	envDBPort            = "MF_SHADOWS_DB_PORT"
	envDBUser            = "MF_SHADOWS_DB_USER"
	envDBPass            = "MF_SHADOWS_DB_PASS"
	envDB                = "MF_SHADOWS_DB"
	envDBSSLMode         = "MF_SHADOWS_DB_SSL_MODE"
	envDBSSLCert         = "MF_SHADOWS_DB_SSL_CERT"
	envDBSSLKey          = "MF_SHADOWS_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_SHADOWS_DB_SSL_ROOT_CERT"
	envClientTLS         = "MF_SHADOWS_CLIENT_TLS"
	envCACerts           = "MF_SHADOWS_CA_CERTS"
	envHTTPPort          = "MF_SHADOWS_HTTP_PORT"
	envServerCert        = "MF_SHADOWS_SERVER_CERT"
	envServerKey         = "MF_SHADOWS_SERVER_KEY"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envAuthGRPCURL       = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout   = "MF_AUTH_GRPC_TIMEOUT"
	envBrokerURL         = "MF_BROKER_URL"
	envESURL             = "MF_SHADOWS_ES_URL"
)

type config struct {
	logLevel          string
	dbConfig          postgres.Config
	httpConfig        servers.Config
	thingsConfig      clients.Config
	authConfig        clients.Config
	jaegerURL         string
	thingsGRPCTimeout time.Duration
	authGRPCTimeout   time.Duration
	brokerURL         string
	esURL             string
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err.Error())
	}

	shadowsTracer, shadowsCloser := jaeger.Init(svcName, cfg.jaegerURL, logger)
	defer shadowsCloser.Close()

	thingsTracer, thingsCloser := jaeger.Init("shadows_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	thingsConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer thingsConn.Close()

	things := thingsapi.NewClient(thingsConn, thingsTracer, cfg.thingsGRPCTimeout)

	authTracer, authCloser := jaeger.Init("shadows_auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	authConn := clientsgrpc.Connect(cfg.authConfig, logger)
	defer authConn.Close()

	auth := authapi.NewClient(authConn, authTracer, cfg.authGRPCTimeout)

	dbTracer, dbCloser := jaeger.Init("shadows_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	pubSub, err := nats.NewPubSub(cfg.brokerURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	svc := newService(things, pubSub, dbTracer, db, logger)

	subjects := []string{nats.SubjectMessages, nats.SubjectMessagesWithSubtopic}
	if err := consumers.Start(svcName, consumers.Messages(pubSub, svc, subjects...)); err != nil {
		logger.Error(fmt.Sprintf("Failed to subscribe to message broker: %s", err))
		os.Exit(1)
	}

	g.Go(func() error {
		return subscribeToThingsES(ctx, svc, cfg, logger)
	})

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(shadowsTracer, svc, auth, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Shadows service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Shadows service terminated: %s", err))
	}
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	thingsGRPCTimeout, err := time.ParseDuration(mainflux.Env(envThingsGRPCTimeout, defThingsGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsGRPCTimeout, err.Error())
	}

	authGRPCTimeout, err := time.ParseDuration(mainflux.Env(envAuthGRPCTimeout, defAuthGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthGRPCTimeout, err.Error())
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

	authConfig := clients.Config{
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envCACerts, defCACerts),
		URL:        mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		ClientName: clients.Auth,
	}

	return config{
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:          dbConfig,
		httpConfig:        httpConfig,
		thingsConfig:      thingsConfig,
		authConfig:        authConfig,
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
		authGRPCTimeout:   authGRPCTimeout,
		esURL:             mainflux.Env(envESURL, defESURL),
	}
}

func subscribeToThingsES(ctx context.Context, svc shadows.Service, cfg config, logger logger.Logger) error {
	subscriber, err := mfevents.NewSubscriber(mfevents.SubscriberConfig{
		URL:    cfg.esURL,
		Stream: mfevents.ThingsStream,
		Name:   svcName,
	}, logger)
	if err != nil {
		return err
	}

	defer func() {
		if err := subscriber.Close(); err != nil {
			logger.Error(fmt.Sprintf("Failed to close subscriber: %s", err))
		}
	}()

	return subscriber.Subscribe(ctx, events.NewEventHandler(svc))
}

func newService(things domain.ThingsClient, pub messaging.CommandPublisher, dbTracer opentracing.Tracer, db *sqlx.DB, logger logger.Logger) shadows.Service {
	repo := postgres.NewShadowRepository(dbutil.NewDatabase(db))
	repo = tracing.ShadowRepositoryMiddleware(dbTracer, repo)

	svc := shadows.New(things, repo, pub, logger)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "shadows",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "shadows",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}
