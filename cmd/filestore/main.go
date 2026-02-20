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
	"github.com/MainfluxLabs/mainflux/filestore"
	"github.com/MainfluxLabs/mainflux/filestore/api"
	httpapi "github.com/MainfluxLabs/mainflux/filestore/api/http"
	"github.com/MainfluxLabs/mainflux/filestore/events"
	"github.com/MainfluxLabs/mainflux/filestore/postgres"
	"github.com/MainfluxLabs/mainflux/filestore/tracing"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfevents "github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "filestore"
	stopWaitTime = 5 * time.Second
	esGroupName  = svcName

	defLogLevel          = "error"
	defHTTPPort          = "9024"
	defJaegerURL         = ""
	defServerCert        = ""
	defServerKey         = ""
	defCACerts           = ""
	defThingsAuthURL     = "localhost:8183"
	defThingsAuthTimeout = "1s"
	defClientTLS         = "false"
	defDBHost            = "localhost"
	defDBPort            = "5432"
	defDBUser            = "mainflux"
	defDBPass            = "mainflux"
	defDB                = svcName
	defDBSSLMode         = "disable"
	defDBSSLCert         = ""
	defDBSSLKey          = ""
	defDBSSLRootCert     = ""
	defESURL             = "redis://localhost:6379/0"
	defESConsumerName    = svcName

	envDBHost            = "MF_FILESTORE_DB_HOST"
	envDBPort            = "MF_FILESTORE_DB_PORT"
	envDBUser            = "MF_FILESTORE_DB_USER"
	envDBPass            = "MF_FILESTORE_DB_PASS"
	envDB                = "MF_FILESTORE_DB"
	envDBSSLMode         = "MF_FILESTORE_DB_SSL_MODE"
	envDBSSLCert         = "MF_FILESTORE_DB_SSL_CERT"
	envDBSSLKey          = "MF_FILESTORE_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_FILESTORE_DB_SSL_ROOT_CERT"
	envLogLevel          = "MF_FILESTORE_LOG_LEVEL"
	envHTTPPort          = "MF_FILESTORE_HTTP_PORT"
	envServerCert        = "MF_FILESTORE_SERVER_CERT"
	envServerKey         = "MF_FILESTORE_SERVER_KEY"
	envJaegerURL         = "MF_JAEGER_URL"
	envFileStoreCert     = "MF_FILESTORE_SERVER_CERT"
	envClientTLS         = "MF_FILESTORE_TLS"
	envThingsAuthURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsAuthTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envESURL             = "MF_FILESTORE_ES_URL"
	envESConsumerName    = "MF_FILESTORE_EVENT_CONSUMER"
)

type config struct {
	logLevel          string
	jaegerURL         string
	thingsAuthTimeout time.Duration
	dbConfig          postgres.Config
	httpConfig        servers.Config
	thingsConfig      clients.Config
	esURL             string
	esConsumerName    string
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	// Create temp dir to support uploading large files
	tempDir := os.TempDir()
	if err := os.MkdirAll(tempDir, 1777); err != nil {
		log.Fatalf("Failed to create temporary directory %s: %s", tempDir, err)
	}

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	fileStoreTracer, fileStoreCloser := jaeger.Init(svcName, cfg.jaegerURL, logger)
	defer fileStoreCloser.Close()

	thingsTracer, thingsCloser := jaeger.Init("filestore_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	thingsAuthConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer thingsAuthConn.Close()

	thingsAuth := thingsapi.NewClient(thingsAuthConn, thingsTracer, cfg.thingsAuthTimeout)

	dbTracer, dbCloser := jaeger.Init("filestore_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(thingsAuth, dbTracer, db, logger)

	g.Go(func() error {
		return subscribeToThingsES(ctx, svc, cfg, logger)
	})

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(fileStoreTracer, svc, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Filestore service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Filestore service terminated: %s", err))
	}
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	thingsAuthTimeout, err := time.ParseDuration(mainflux.Env(envThingsAuthTimeout, defThingsAuthTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsAuthTimeout, err.Error())
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
		CaCerts:    mainflux.Env(envFileStoreCert, defCACerts),
		URL:        mainflux.Env(envThingsAuthURL, defThingsAuthURL),
		ClientName: clients.Things,
	}

	return config{
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsAuthTimeout: thingsAuthTimeout,
		dbConfig:          dbConfig,
		httpConfig:        httpConfig,
		thingsConfig:      thingsConfig,
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

func subscribeToThingsES(ctx context.Context, svc filestore.Service, cfg config, logger logger.Logger) error {
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

func newService(thingsAuth protomfx.ThingsServiceClient, dbTracer opentracing.Tracer, db *sqlx.DB, logger logger.Logger) filestore.Service {
	thRepo := postgres.NewThingsRepository(db)
	thRepo = tracing.ThingsRepositoryMiddleware(dbTracer, thRepo)
	grRepo := postgres.NewGroupsRepository(db)
	grRepo = tracing.GroupsRepositoryMiddleware(dbTracer, grRepo)
	svc := filestore.New(thingsAuth, thRepo, grRepo)

	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "filestore",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "filestore",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}
