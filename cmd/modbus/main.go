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

	_ "time/tzdata"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/modbus/api"
	"github.com/MainfluxLabs/mainflux/modbus/events"
	"github.com/MainfluxLabs/mainflux/modbus/postgres"
	"github.com/MainfluxLabs/mainflux/modbus/tracing"

	httpapi "github.com/MainfluxLabs/mainflux/modbus/api/http"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfevents "github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "modbus"
	stopWaitTime = 5 * time.Second
	esGroupName  = svcName

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
	defHTTPPort          = "9028"
	defJaegerURL         = ""
	defBrokerURL         = "nats://localhost:4222"
	defServerCert        = ""
	defServerKey         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defESURL             = "redis://localhost:6379/0"
	defESConsumerName    = svcName

	envLogLevel          = "MF_MODBUS_LOG_LEVEL"
	envDBHost            = "MF_MODBUS_DB_HOST"
	envDBPort            = "MF_MODBUS_DB_PORT"
	envDBUser            = "MF_MODBUS_DB_USER"
	envDBPass            = "MF_MODBUS_DB_PASS"
	envDB                = "MF_MODBUS_DB"
	envDBSSLMode         = "MF_MODBUS_DB_SSL_MODE"
	envDBSSLCert         = "MF_MODBUS_DB_SSL_CERT"
	envDBSSLKey          = "MF_MODBUS_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_MODBUS_DB_SSL_ROOT_CERT"
	envClientTLS         = "MF_MODBUS_CLIENT_TLS"
	envCACerts           = "MF_MODBUS_CA_CERTS"
	envHTTPPort          = "MF_MODBUS_HTTP_PORT"
	envServerCert        = "MF_MODBUS_SERVER_CERT"
	envServerKey         = "MF_MODBUS_SERVER_KEY"
	envJaegerURL         = "MF_JAEGER_URL"
	envBrokerURL         = "MF_BROKER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envESURL             = "MF_MODBUS_ES_URL"
	envESConsumerName    = "MF_MODBUS_EVENT_CONSUMER"
)

type config struct {
	logLevel          string
	dbConfig          postgres.Config
	httpConfig        servers.Config
	thingsConfig      clients.Config
	jaegerURL         string
	brokerURL         string
	thingsGRPCTimeout time.Duration
	esURL             string
	esConsumerName    string
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err.Error())
	}

	modbusTracer, modbusCloser := jaeger.Init(svcName, cfg.jaegerURL, logger)
	defer modbusCloser.Close()

	thingsTracer, thingsCloser := jaeger.Init("modbus_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	thingsConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer thingsConn.Close()

	things := thingsapi.NewClient(thingsConn, thingsTracer, cfg.thingsGRPCTimeout)

	dbTracer, dbCloser := jaeger.Init("modbus_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	pub, err := brokers.NewPublisher(cfg.brokerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pub.Close()

	svc := newService(things, pub, dbTracer, db, logger)

	g.Go(func() error {
		return subscribeToThingsES(ctx, svc, cfg, logger)
	})

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(modbusTracer, svc, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		return svc.LoadAndScheduleTasks(ctx)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Modbus service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Modbus service terminated: %s", err))
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
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:          dbConfig,
		httpConfig:        httpConfig,
		thingsConfig:      thingsConfig,
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		thingsGRPCTimeout: thingsAuthGRPCTimeout,
		esURL:             mainflux.Env(envESURL, defESURL),
		esConsumerName:    mainflux.Env(envESConsumerName, defESConsumerName),
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

func subscribeToThingsES(ctx context.Context, svc modbus.Service, cfg config, logger logger.Logger) error {
	subscriber, err := mfevents.NewSubscriber(cfg.esURL, mfevents.ThingsStream, esGroupName, cfg.esConsumerName, logger)
	if err != nil {
		return err
	}

	defer func() {
		if err := subscriber.Close(); err != nil {
			logger.Error(fmt.Sprintf("Failed to close subscriber: %s", err))
		}
	}()

	handler := events.NewEventHandler(svc)

	return subscriber.Subscribe(ctx, handler)
}

func newService(ts protomfx.ThingsServiceClient, pub messaging.Publisher, dbTracer opentracing.Tracer, db *sqlx.DB, logger logger.Logger) modbus.Service {
	database := dbutil.NewDatabase(db)
	clientsRepo := postgres.NewClientRepository(database)
	clientsRepo = tracing.ClientRepositoryMiddleware(dbTracer, clientsRepo)
	idProvider := uuid.New()
	svc := modbus.New(ts, pub, clientsRepo, idProvider, logger)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "modbus",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "modbus",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}
