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
	"github.com/MainfluxLabs/mainflux/ota"
	"github.com/MainfluxLabs/mainflux/ota/api"
	httpapi "github.com/MainfluxLabs/mainflux/ota/api/http"
	"github.com/MainfluxLabs/mainflux/ota/events"
	"github.com/MainfluxLabs/mainflux/ota/postgres"
	"github.com/MainfluxLabs/mainflux/ota/tracing"
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
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "ota"
	stopWaitTime = 5 * time.Second

	defBrokerURL         = "nats://localhost:4222"
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
	defHTTPPort          = "9032"
	defJaegerURL         = ""
	defServerCert        = ""
	defServerKey         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defAuthGRPCURL       = "localhost:8181"
	defAuthGRPCTimeout   = "1s"
	defESURL             = "redis://localhost:6379/0"
	defFirmwareURL       = "https://localhost/fs/groupfiles"
	defTimeoutInterval   = "1m"

	envBrokerURL         = "MF_BROKER_URL"
	envLogLevel          = "MF_OTA_LOG_LEVEL"
	envDBHost            = "MF_OTA_DB_HOST"
	envDBPort            = "MF_OTA_DB_PORT"
	envDBUser            = "MF_OTA_DB_USER"
	envDBPass            = "MF_OTA_DB_PASS"
	envDB                = "MF_OTA_DB"
	envDBSSLMode         = "MF_OTA_DB_SSL_MODE"
	envDBSSLCert         = "MF_OTA_DB_SSL_CERT"
	envDBSSLKey          = "MF_OTA_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_OTA_DB_SSL_ROOT_CERT"
	envClientTLS         = "MF_OTA_CLIENT_TLS"
	envCACerts           = "MF_OTA_CA_CERTS"
	envHTTPPort          = "MF_OTA_HTTP_PORT"
	envServerCert        = "MF_OTA_SERVER_CERT"
	envServerKey         = "MF_OTA_SERVER_KEY"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envAuthGRPCURL       = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout   = "MF_AUTH_GRPC_TIMEOUT"
	envESURL             = "MF_OTA_ES_URL"
	envFirmwareURL       = "MF_OTA_FIRMWARE_URL"
	envTimeoutInterval   = "MF_OTA_TIMEOUT_CHECK_INTERVAL"
)

type config struct {
	brokerURL         string
	logLevel          string
	dbConfig          postgres.Config
	httpConfig        servers.Config
	thingsConfig      clients.Config
	authConfig        clients.Config
	jaegerURL         string
	thingsGRPCTimeout time.Duration
	authGRPCTimeout   time.Duration
	esURL             string
	firmwareURL       string
	timeoutInterval   time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err.Error())
	}

	pubSub, err := nats.NewPubSub(cfg.brokerURL, svcName, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	otaTracer, otaCloser := jaeger.Init(svcName, cfg.jaegerURL, logger)
	defer otaCloser.Close()

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	thingsTracer, thingsCloser := jaeger.Init("ota_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	thingsConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer thingsConn.Close()

	things := thingsapi.NewClient(thingsConn, thingsTracer, cfg.thingsGRPCTimeout)

	authTracer, authCloser := jaeger.Init("ota_auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	authConn := clientsgrpc.Connect(cfg.authConfig, logger)
	defer authConn.Close()

	auth := authapi.NewClient(authConn, authTracer, cfg.authGRPCTimeout)

	dbTracer, dbCloser := jaeger.Init("ota_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(things, pubSub, dbTracer, db, cfg.firmwareURL, logger)

	g.Go(func() error {
		return subscribeToThingsES(ctx, svc, cfg, logger)
	})

	if err := consumers.Messages(svcName, pubSub, svc, nats.SubjectOTA); err != nil {
		logger.Error(fmt.Sprintf("Failed to subscribe to message broker: %s", err))
		os.Exit(1)
	}

	g.Go(func() error {
		return failTimedOut(ctx, svc, cfg.timeoutInterval)
	})

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(otaTracer, svc, auth, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("OTA service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("OTA service terminated: %s", err))
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

	authGRPCTimeout, err := time.ParseDuration(mainflux.Env(envAuthGRPCTimeout, defAuthGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthGRPCTimeout, err.Error())
	}

	timeoutInterval, err := time.ParseDuration(mainflux.Env(envTimeoutInterval, defTimeoutInterval))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envTimeoutInterval, err.Error())
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
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:          dbConfig,
		httpConfig:        httpConfig,
		thingsConfig:      thingsConfig,
		authConfig:        authConfig,
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCTimeout: thingsAuthGRPCTimeout,
		authGRPCTimeout:   authGRPCTimeout,
		esURL:             mainflux.Env(envESURL, defESURL),
		firmwareURL:       mainflux.Env(envFirmwareURL, defFirmwareURL),
		timeoutInterval:   timeoutInterval,
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

func subscribeToThingsES(ctx context.Context, svc ota.Service, cfg config, logger logger.Logger) error {
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

	handler := events.NewEventHandler(svc)

	return subscriber.Subscribe(ctx, handler)
}

// failTimedOut periodically fails thing updates without progress; errors are logged by the service middleware.
func failTimedOut(ctx context.Context, svc ota.Service, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			svc.FailTimedOut(ctx)
		}
	}
}

func newService(ts domain.ThingsClient, pub messaging.CommandPublisher, dbTracer opentracing.Tracer, db *sqlx.DB, firmwareURL string, logger logger.Logger) ota.Service {
	database := dbutil.NewDatabase(db)
	updatesRepo := postgres.NewUpdateRepository(database)
	updatesRepo = tracing.UpdateRepositoryMiddleware(dbTracer, updatesRepo)
	idProvider := uuid.New()

	svc := ota.New(ts, updatesRepo, pub, idProvider, firmwareURL, logger)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "ota",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "ota",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}
