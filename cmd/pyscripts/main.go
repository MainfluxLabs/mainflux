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
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfevents "github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/pyscripts"
	"github.com/MainfluxLabs/mainflux/pyscripts/api"
	httpapi "github.com/MainfluxLabs/mainflux/pyscripts/api/http"
	"github.com/MainfluxLabs/mainflux/pyscripts/events"
	"github.com/MainfluxLabs/mainflux/pyscripts/postgres"
	"github.com/MainfluxLabs/mainflux/pyscripts/runner"
	"github.com/MainfluxLabs/mainflux/pyscripts/tracing"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "pyscripts"
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
	defHTTPPort          = "9032"
	defJaegerURL         = ""
	defServerCert        = ""
	defServerKey         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defAuthGRPCURL       = "localhost:8181"
	defAuthGRPCTimeout   = "1s"
	defESURL             = "redis://localhost:6379/0"

	defPython        = "python3"
	defWorkers       = "2"
	defRunTimeout    = "10s"
	defAcquireWait   = "5s"
	defStartWait     = "60s"
	defMaxScriptSize = "65535"
	defMaxPayload    = "1048576"
	defMaxBody       = "4194304"
	defMaxOutput     = "1048576"
	defMaxLogLines   = "256"
	defMaxCPUSeconds = "5"
	defMaxMemory     = "268435456"
	defMaxFileSize   = "0"
	defBLASThreads   = "1"

	envLogLevel          = "MF_PYSCRIPTS_LOG_LEVEL"
	envDBHost            = "MF_PYSCRIPTS_DB_HOST"
	envDBPort            = "MF_PYSCRIPTS_DB_PORT"
	envDBUser            = "MF_PYSCRIPTS_DB_USER"
	envDBPass            = "MF_PYSCRIPTS_DB_PASS"
	envDB                = "MF_PYSCRIPTS_DB"
	envDBSSLMode         = "MF_PYSCRIPTS_DB_SSL_MODE"
	envDBSSLCert         = "MF_PYSCRIPTS_DB_SSL_CERT"
	envDBSSLKey          = "MF_PYSCRIPTS_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_PYSCRIPTS_DB_SSL_ROOT_CERT"
	envClientTLS         = "MF_PYSCRIPTS_CLIENT_TLS"
	envCACerts           = "MF_PYSCRIPTS_CA_CERTS"
	envHTTPPort          = "MF_PYSCRIPTS_HTTP_PORT"
	envServerCert        = "MF_PYSCRIPTS_SERVER_CERT"
	envServerKey         = "MF_PYSCRIPTS_SERVER_KEY"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envAuthGRPCURL       = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout   = "MF_AUTH_GRPC_TIMEOUT"
	envESURL             = "MF_PYSCRIPTS_ES_URL"

	envPython        = "MF_PYSCRIPTS_PYTHON"
	envWorkers       = "MF_PYSCRIPTS_WORKERS"
	envRunTimeout    = "MF_PYSCRIPTS_RUN_TIMEOUT"
	envAcquireWait   = "MF_PYSCRIPTS_ACQUIRE_TIMEOUT"
	envStartWait     = "MF_PYSCRIPTS_START_TIMEOUT"
	envMaxScriptSize = "MF_PYSCRIPTS_MAX_SCRIPT_SIZE"
	envMaxPayload    = "MF_PYSCRIPTS_MAX_PAYLOAD_SIZE"
	envMaxBody       = "MF_PYSCRIPTS_MAX_BODY_SIZE"
	envMaxOutput     = "MF_PYSCRIPTS_MAX_OUTPUT_SIZE"
	envMaxLogLines   = "MF_PYSCRIPTS_MAX_LOG_LINES"
	envMaxCPUSeconds = "MF_PYSCRIPTS_MAX_CPU_SECONDS"
	envMaxMemory     = "MF_PYSCRIPTS_MAX_MEMORY"
	envMaxFileSize   = "MF_PYSCRIPTS_MAX_FILE_SIZE"
	envBLASThreads   = "MF_PYSCRIPTS_BLAS_THREADS"
)

type config struct {
	logLevel          string
	dbConfig          postgres.Config
	httpConfig        servers.Config
	thingsConfig      clients.Config
	authConfig        clients.Config
	runnerConfig      runner.Config
	jaegerURL         string
	thingsGRPCTimeout time.Duration
	authGRPCTimeout   time.Duration
	esURL             string
	maxScriptSize     int
	maxPayloadSize    int64
	maxBodySize       int64
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err)
	}

	pyscripts.MaxScriptSize = cfg.maxScriptSize
	httpapi.MaxPayloadSize = cfg.maxPayloadSize
	httpapi.MaxBodySize = cfg.maxBodySize

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	httpTracer, httpCloser := jaeger.Init("pyscripts_http", cfg.jaegerURL, logger)
	defer httpCloser.Close()

	dbTracer, dbCloser := jaeger.Init("pyscripts_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	thConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer thConn.Close()

	thingsTracer, thingsCloser := jaeger.Init("pyscripts_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	tc := thingsapi.NewClient(thConn, thingsTracer, cfg.thingsGRPCTimeout)

	authConn := clientsgrpc.Connect(cfg.authConfig, logger)
	defer authConn.Close()

	authTracer, authCloser := jaeger.Init("pyscripts_auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	auth := authapi.NewClient(authConn, authTracer, cfg.authGRPCTimeout)

	run, err := runner.New(cfg.runnerConfig, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to start python worker pool: %s", err))
		os.Exit(1)
	}
	defer run.Close()

	svc := newService(dbTracer, db, run, tc, logger)

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(httpTracer, svc, auth, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		return subscribeToThingsES(ctx, svc, cfg, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Pyscripts service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Pyscripts service terminated: %s", err))
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

	runTimeout, err := time.ParseDuration(mainflux.Env(envRunTimeout, defRunTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envRunTimeout, err.Error())
	}

	acquireWait, err := time.ParseDuration(mainflux.Env(envAcquireWait, defAcquireWait))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAcquireWait, err.Error())
	}

	startWait, err := time.ParseDuration(mainflux.Env(envStartWait, defStartWait))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envStartWait, err.Error())
	}

	return config{
		logLevel: mainflux.Env(envLogLevel, defLogLevel),
		dbConfig: postgres.Config{
			Host:        mainflux.Env(envDBHost, defDBHost),
			Port:        mainflux.Env(envDBPort, defDBPort),
			User:        mainflux.Env(envDBUser, defDBUser),
			Pass:        mainflux.Env(envDBPass, defDBPass),
			Name:        mainflux.Env(envDB, defDB),
			SSLMode:     mainflux.Env(envDBSSLMode, defDBSSLMode),
			SSLCert:     mainflux.Env(envDBSSLCert, defDBSSLCert),
			SSLKey:      mainflux.Env(envDBSSLKey, defDBSSLKey),
			SSLRootCert: mainflux.Env(envDBSSLRootCert, defDBSSLRootCert),
		},
		httpConfig: servers.Config{
			ServerName:   svcName,
			ServerCert:   mainflux.Env(envServerCert, defServerCert),
			ServerKey:    mainflux.Env(envServerKey, defServerKey),
			Port:         mainflux.Env(envHTTPPort, defHTTPPort),
			StopWaitTime: stopWaitTime,
		},
		thingsConfig: clients.Config{
			ClientName: clients.Things,
			ClientTLS:  tls,
			CaCerts:    mainflux.Env(envCACerts, defCACerts),
			URL:        mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		},
		authConfig: clients.Config{
			ClientName: clients.Auth,
			ClientTLS:  tls,
			CaCerts:    mainflux.Env(envCACerts, defCACerts),
			URL:        mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		},
		runnerConfig: runner.Config{
			Python:         mainflux.Env(envPython, defPython),
			Workers:        envInt(envWorkers, defWorkers),
			Timeout:        runTimeout,
			AcquireTimeout: acquireWait,
			StartTimeout:   startWait,
			MaxOutput:      int64(envInt(envMaxOutput, defMaxOutput)),
			MaxLogLines:    envInt(envMaxLogLines, defMaxLogLines),
			BLASThreads:    envInt(envBLASThreads, defBLASThreads),
			Limits: runner.Limits{
				CPUSeconds:  envInt(envMaxCPUSeconds, defMaxCPUSeconds),
				MaxMemory:   int64(envInt(envMaxMemory, defMaxMemory)),
				MaxFileSize: int64(envInt(envMaxFileSize, defMaxFileSize)),
			},
		},
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
		authGRPCTimeout:   authGRPCTimeout,
		esURL:             mainflux.Env(envESURL, defESURL),
		maxScriptSize:     envInt(envMaxScriptSize, defMaxScriptSize),
		maxPayloadSize:    int64(envInt(envMaxPayload, defMaxPayload)),
		maxBodySize:       int64(envInt(envMaxBody, defMaxBody)),
	}
}

func envInt(key, fallback string) int {
	v, err := strconv.Atoi(mainflux.Env(key, fallback))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", key, err.Error())
	}

	return v
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}

	return db
}

func subscribeToThingsES(ctx context.Context, svc pyscripts.Service, cfg config, logger logger.Logger) error {
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

func newService(dbTracer opentracing.Tracer, db *sqlx.DB, run runner.Runner, tc domain.ThingsClient, logger logger.Logger) pyscripts.Service {
	database := dbutil.NewDatabase(db)

	repo := postgres.NewScriptRepository(database)
	repo = tracing.ScriptRepositoryMiddleware(dbTracer, repo)

	idProvider := uuid.New()
	svc := pyscripts.New(repo, run, tc, idProvider, logger)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "pyscripts",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "pyscripts",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}
