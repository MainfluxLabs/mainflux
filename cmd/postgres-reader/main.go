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
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/readers/api"
	httpapi "github.com/MainfluxLabs/mainflux/readers/api/http"
	"github.com/MainfluxLabs/mainflux/readers/postgres"
	"github.com/MainfluxLabs/mainflux/readers/tracing"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "postgres-reader"
	stopWaitTime = 5 * time.Second

	defLogLevel          = "error"
	defPort              = "8180"
	defClientTLS         = "false"
	defCACerts           = ""
	defDBHost            = "localhost"
	defDBPort            = "5432"
	defDBUser            = "mainflux"
	defDBPass            = "mainflux"
	defDB                = "mainflux"
	defDBSSLMode         = "disable"
	defDBSSLCert         = ""
	defDBSSLKey          = ""
	defDBSSLRootCert     = ""
	defJaegerURL         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defAuthGRPCURL       = "localhost:8181"
	defAuthGRPCTimeout   = "1s"

	envLogLevel          = "MF_POSTGRES_READER_LOG_LEVEL"
	envPort              = "MF_POSTGRES_READER_PORT"
	envClientTLS         = "MF_POSTGRES_READER_CLIENT_TLS"
	envCACerts           = "MF_POSTGRES_READER_CA_CERTS"
	envDBHost            = "MF_POSTGRES_READER_DB_HOST"
	envDBPort            = "MF_POSTGRES_READER_DB_PORT"
	envDBUser            = "MF_POSTGRES_READER_DB_USER"
	envDBPass            = "MF_POSTGRES_READER_DB_PASS"
	envDB                = "MF_POSTGRES_READER_DB"
	envDBSSLMode         = "MF_POSTGRES_READER_DB_SSL_MODE"
	envDBSSLCert         = "MF_POSTGRES_READER_DB_SSL_CERT"
	envDBSSLKey          = "MF_POSTGRES_READER_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_POSTGRES_READER_DB_SSL_ROOT_CERT"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envAuthGRPCURL       = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout   = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	logLevel          string
	dbConfig          postgres.Config
	httpConfig        servers.Config
	authConfig        clients.Config
	thingsConfig      clients.Config
	jaegerURL         string
	thingsGRPCTimeout time.Duration
	authGRPCTimeout   time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err)
	}

	postgresHttpTracer, postgresHttpCloser := jaeger.Init("postgres_http", cfg.jaegerURL, logger)
	defer postgresHttpCloser.Close()

	conn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer conn.Close()

	thingsTracer, thingsCloser := jaeger.Init("postgres_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsGRPCTimeout)

	authTracer, authCloser := jaeger.Init("postgres_auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	authConn := clientsgrpc.Connect(cfg.authConfig, logger)
	defer authConn.Close()

	auth := authapi.NewClient(authConn, authTracer, cfg.authGRPCTimeout)

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	dbTracer, dbCloser := jaeger.Init("postgres", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(db, dbTracer, auth, tc, logger)

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(svc, postgresHttpTracer, svcName, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Postgres reader service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Postgres reader service terminated: %s", err))
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
		Port:         mainflux.Env(envPort, defPort),
		StopWaitTime: stopWaitTime,
	}

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
		httpConfig:        httpConfig,
		thingsConfig:      thingsConfig,
		authConfig:        authConfig,
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:          dbConfig,
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
		authGRPCTimeout:   authGRPCTimeout,
	}
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func newService(db *sqlx.DB, dbTracer opentracing.Tracer, ac protomfx.AuthServiceClient, tc protomfx.ThingsServiceClient, logger logger.Logger) readers.Service {
	database := dbutil.NewDatabase(db)

	jsonRepo := postgres.NewJSONRepository(database)
	jsonRepo = tracing.JSONRepositoryMiddleware(dbTracer, jsonRepo)

	senmlRepo := postgres.NewSenMLRepository(database)
	senmlRepo = tracing.SenMLRepositoryMiddleware(dbTracer, senmlRepo)

	svc := readers.New(ac, tc, jsonRepo, senmlRepo)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "postgres",
			Subsystem: "message_reader",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "postgres",
			Subsystem: "message_reader",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}
