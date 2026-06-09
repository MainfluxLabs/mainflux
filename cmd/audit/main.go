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
	"github.com/MainfluxLabs/mainflux/audit"
	api "github.com/MainfluxLabs/mainflux/audit/api"
	httpapi "github.com/MainfluxLabs/mainflux/audit/api/http"
	auditevents "github.com/MainfluxLabs/mainflux/audit/events"
	"github.com/MainfluxLabs/mainflux/audit/postgres"
	"github.com/MainfluxLabs/mainflux/audit/tracing"
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfevents "github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
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
	stopWaitTime = 5 * time.Second
	svcName      = "audit"

	defLogLevel        = "error"
	defDBHost          = "localhost"
	defDBPort          = "5432"
	defDBUser          = "mainflux"
	defDBPass          = "mainflux"
	defDB              = svcName
	defDBSSLMode       = "disable"
	defDBSSLCert       = ""
	defDBSSLKey        = ""
	defDBSSLRootCert   = ""
	defHTTPPort        = "9029"
	defServerCert      = ""
	defServerKey       = ""
	defJaegerURL       = ""
	defESURL           = "redis://localhost:6379/0"
	defAuthGRPCURL     = "localhost:8181"
	defAuthGRPCTimeout = "1s"
	defAuthClientTLS   = "false"
	defAuthCACerts     = ""
	defThingsGRPCURL   = "localhost:8183"
	defThingsTimeout   = "1s"
	defThingsClientTLS = "false"
	defThingsCACerts   = ""

	envLogLevel        = "MF_AUDIT_LOG_LEVEL"
	envDBHost          = "MF_AUDIT_DB_HOST"
	envDBPort          = "MF_AUDIT_DB_PORT"
	envDBUser          = "MF_AUDIT_DB_USER"
	envDBPass          = "MF_AUDIT_DB_PASS"
	envDB              = "MF_AUDIT_DB"
	envDBSSLMode       = "MF_AUDIT_DB_SSL_MODE"
	envDBSSLCert       = "MF_AUDIT_DB_SSL_CERT"
	envDBSSLKey        = "MF_AUDIT_DB_SSL_KEY"
	envDBSSLRootCert   = "MF_AUDIT_DB_SSL_ROOT_CERT"
	envHTTPPort        = "MF_AUDIT_HTTP_PORT"
	envServerCert      = "MF_AUDIT_SERVER_CERT"
	envServerKey       = "MF_AUDIT_SERVER_KEY"
	envJaegerURL       = "MF_JAEGER_URL"
	envESURL           = "MF_AUDIT_ES_URL"
	envAuthGRPCURL     = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout = "MF_AUTH_GRPC_TIMEOUT"
	envAuthClientTLS   = "MF_AUTH_CLIENT_TLS"
	envAuthCACerts     = "MF_AUTH_CA_CERTS"
	envThingsGRPCURL   = "MF_THINGS_AUTH_GRPC_URL"
	envThingsTimeout   = "MF_THINGS_GRPC_TIMEOUT"
	envThingsClientTLS = "MF_THINGS_CLIENT_TLS"
	envThingsCACerts   = "MF_THINGS_CA_CERTS"
)

type config struct {
	logLevel          string
	dbConfig          postgres.Config
	httpConfig        servers.Config
	authConfig        clients.Config
	thingsConfig      clients.Config
	jaegerURL         string
	esURL             string
	authGRPCTimeout   time.Duration
	thingsGRPCTimeout time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err)
	}

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	httpTracer, httpCloser := jaeger.Init("audit_http", cfg.jaegerURL, logger)
	defer httpCloser.Close()

	dbTracer, dbCloser := jaeger.Init("audit_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	authConn := clientsgrpc.Connect(cfg.authConfig, logger)
	defer authConn.Close()

	authTracer, authCloser := jaeger.Init("audit_auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	ac := authapi.NewClient(authConn, authTracer, cfg.authGRPCTimeout)

	thingsConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer thingsConn.Close()

	thingsTracer, thingsCloser := jaeger.Init("audit_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	tc := thingsapi.NewClient(thingsConn, thingsTracer, cfg.thingsGRPCTimeout)

	svc := newService(db, ac, tc, dbTracer, logger)

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(svc, ac, httpTracer, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		return subscribeToES(ctx, svc, cfg, mfevents.AuthStream, logger)
	})

	g.Go(func() error {
		return subscribeToES(ctx, svc, cfg, mfevents.ThingsStream, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Audit service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Audit service terminated: %s", err))
	}
}

func loadConfig() config {
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

	authClientTLS, err := strconv.ParseBool(mainflux.Env(envAuthClientTLS, defAuthClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envAuthClientTLS)
	}

	thingsClientTLS, err := strconv.ParseBool(mainflux.Env(envThingsClientTLS, defThingsClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envThingsClientTLS)
	}

	authConfig := clients.Config{
		ClientTLS:  authClientTLS,
		CaCerts:    mainflux.Env(envAuthCACerts, defAuthCACerts),
		URL:        mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		ClientName: clients.Auth,
	}

	thingsConfig := clients.Config{
		ClientTLS:  thingsClientTLS,
		CaCerts:    mainflux.Env(envThingsCACerts, defThingsCACerts),
		URL:        mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		ClientName: clients.Things,
	}

	authGRPCTimeout, err := time.ParseDuration(mainflux.Env(envAuthGRPCTimeout, defAuthGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthGRPCTimeout, err.Error())
	}

	thingsGRPCTimeout, err := time.ParseDuration(mainflux.Env(envThingsTimeout, defThingsTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsTimeout, err.Error())
	}

	return config{
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:          dbConfig,
		httpConfig:        httpConfig,
		authConfig:        authConfig,
		thingsConfig:      thingsConfig,
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		esURL:             mainflux.Env(envESURL, defESURL),
		authGRPCTimeout:   authGRPCTimeout,
		thingsGRPCTimeout: thingsGRPCTimeout,
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

func newService(db *sqlx.DB, ac domain.AuthClient, tc domain.ThingsClient, dbTracer opentracing.Tracer, logger logger.Logger) audit.Service {
	database := dbutil.NewDatabase(db)

	repo := postgres.NewEventRepository(database)
	repo = tracing.NewEventRepositoryMiddleware(dbTracer, repo)

	idp := uuid.New()

	svc := audit.New(repo, ac, tc, idp)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "audit",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "audit",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}

func subscribeToES(ctx context.Context, svc audit.Service, cfg config, stream string, logger logger.Logger) error {
	subscriber, err := mfevents.NewSubscriber(mfevents.SubscriberConfig{
		URL:    cfg.esURL,
		Stream: stream,
		Name:   svcName,
	}, logger)
	if err != nil {
		return err
	}

	defer func() {
		if err := subscriber.Close(); err != nil {
			logger.Error(fmt.Sprintf("Failed to close %s subscriber: %s", stream, err))
		}
	}()

	handler := auditevents.NewEventHandler(svc)

	return subscriber.Subscribe(ctx, handler)
}
