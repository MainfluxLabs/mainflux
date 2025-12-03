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
	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/consumers/notifiers/api"
	httpapi "github.com/MainfluxLabs/mainflux/consumers/notifiers/api/http"
	"github.com/MainfluxLabs/mainflux/consumers/notifiers/events"
	"github.com/MainfluxLabs/mainflux/consumers/notifiers/postgres"
	mfsmpp "github.com/MainfluxLabs/mainflux/consumers/notifiers/smpp"
	"github.com/MainfluxLabs/mainflux/consumers/notifiers/tracing"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfevents "github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
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
	svcName      = "smpp-notifier"
	stopWaitTime = 5 * time.Second
	thingsStream = "mainflux.things"
	esGroupName  = svcName

	defLogLevel          = "error"
	defFrom              = ""
	defJaegerURL         = ""
	defBrokerURL         = "nats://localhost:4222"
	defDBHost            = "localhost"
	defDBPort            = "5432"
	defDBUser            = "mainflux"
	defDBPass            = "mainflux"
	defDB                = "smpp-notifiers"
	defDBSSLMode         = "disable"
	defDBSSLCert         = ""
	defDBSSLKey          = ""
	defDBSSLRootCert     = ""
	defThingsTLS         = "false"
	defThingsCACerts     = ""
	defHTTPPort          = "9024"
	defServerCert        = ""
	defServerKey         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defESURL             = "redis://localhost:6379/0"
	defESConsumerName    = svcName

	defAddress    = ""
	defUsername   = ""
	defPassword   = ""
	defSystemType = ""
	defSrcAddrTON = "0"
	defDstAddrTON = "0"
	defSrcAddrNPI = "0"
	defDstAddrNPI = "0"

	envLogLevel          = "MF_SMPP_NOTIFIER_LOG_LEVEL"
	envFrom              = "MF_SMPP_NOTIFIER_SOURCE_ADDR"
	envJaegerURL         = "MF_JAEGER_URL"
	envBrokerURL         = "MF_BROKER_URL"
	envDBHost            = "MF_SMPP_NOTIFIER_DB_HOST"
	envDBPort            = "MF_SMPP_NOTIFIER_DB_PORT"
	envDBUser            = "MF_SMPP_NOTIFIER_DB_USER"
	envDBPass            = "MF_SMPP_NOTIFIER_DB_PASS"
	envDB                = "MF_SMPP_NOTIFIER_DB"
	envDBSSLMode         = "MF_SMPP_NOTIFIER_DB_SSL_MODE"
	envDBSSLCert         = "MF_SMPP_NOTIFIER_DB_SSL_CERT"
	envDBSSLKey          = "MF_SMPP_NOTIFIER_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_SMPP_NOTIFIER_DB_SSL_ROOT_CERT"
	envThingsTLS         = "MF_SMPP_NOTIFIER_THINGS_TLS"
	envThingsCACerts     = "MF_SMPP_NOTIFIER_THINGS_CA_CERTS"
	envHTTPPort          = "MF_SMPP_NOTIFIER_PORT"
	envServerCert        = "MF_SMPP_NOTIFIER_SERVER_CERT"
	envServerKey         = "MF_SMPP_NOTIFIER_SERVER_KEY"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envESURL             = "MF_SMPP_NOTIFIER_ES_URL"
	envESConsumerName    = "MF_SMPP_NOTIFIER_EVENT_CONSUMER"

	envAddress    = "MF_SMPP_ADDRESS"
	envUsername   = "MF_SMPP_USERNAME"
	envPassword   = "MF_SMPP_PASSWORD"
	envSystemType = "MF_SMPP_SYSTEM_TYPE"
	envSrcAddrTON = "MF_SMPP_SRC_ADDR_TON"
	envDstAddrTON = "MF_SMPP_DST_ADDR_TON"
	envSrcAddrNPI = "MF_SMPP_SRC_ADDR_NPI"
	envDstAddrNPI = "MF_SMPP_DST_ADDR_NPI"
)

type config struct {
	brokerURL         string
	logLevel          string
	dbConfig          postgres.Config
	httpConfig        servers.Config
	thingsConfig      clients.Config
	smppConf          mfsmpp.Config
	from              string
	jaegerURL         string
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
		log.Fatalf(err.Error())
	}

	pubSub, err := brokers.NewPubSub(cfg.brokerURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	notifiersTracer, notifiersCloser := jaeger.Init(svcName, cfg.jaegerURL, logger)
	defer notifiersCloser.Close()

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	thingsTracer, thingsCloser := jaeger.Init("smpp_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	thConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer thConn.Close()

	tc := thingsapi.NewClient(thConn, thingsTracer, cfg.thingsGRPCTimeout)

	dbTracer, dbCloser := jaeger.Init("smpp_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(cfg, logger, dbTracer, db, tc)

	if err = subscribeToThingsES(ctx, svc, cfg, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to subscribe to things event store: %s", err))
		return
	}

	if err = consumers.Start(svcName, pubSub, svc, nats.SubjectSmpp); err != nil {
		logger.Error(fmt.Sprintf("Failed to create SMPP notifier: %s", err))
	}

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(notifiersTracer, svc, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("SMPP notifier service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("SMPP notifier service terminated: %s", err))
	}

}

func loadConfig() config {
	thingsGRPCTimeout, err := time.ParseDuration(mainflux.Env(envThingsGRPCTimeout, defThingsGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsGRPCTimeout, err.Error())
	}

	thingsTLS, err := strconv.ParseBool(mainflux.Env(envThingsTLS, defThingsTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envThingsTLS)
	}

	saton, err := strconv.ParseUint(mainflux.Env(envSrcAddrTON, defSrcAddrTON), 10, 8)
	if err != nil {
		log.Fatalf("Invalid value passed for %s", envSrcAddrTON)
	}
	daton, err := strconv.ParseUint(mainflux.Env(envDstAddrTON, defDstAddrTON), 10, 8)
	if err != nil {
		log.Fatalf("Invalid value passed for %s", envDstAddrTON)
	}
	sanpi, err := strconv.ParseUint(mainflux.Env(envSrcAddrNPI, defSrcAddrNPI), 10, 8)
	if err != nil {
		log.Fatalf("Invalid value passed for %s", envSrcAddrNPI)
	}
	danpi, err := strconv.ParseUint(mainflux.Env(envDstAddrNPI, defDstAddrNPI), 10, 8)
	if err != nil {
		log.Fatalf("Invalid value passed for %s", envDstAddrNPI)
	}

	smppConf := mfsmpp.Config{
		Address:       mainflux.Env(envAddress, defAddress),
		Username:      mainflux.Env(envUsername, defUsername),
		Password:      mainflux.Env(envPassword, defPassword),
		SystemType:    mainflux.Env(envSystemType, defSystemType),
		SourceAddrTON: uint8(saton),
		DestAddrTON:   uint8(daton),
		SourceAddrNPI: uint8(sanpi),
		DestAddrNPI:   uint8(danpi),
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
		ClientTLS:  thingsTLS,
		CaCerts:    mainflux.Env(envThingsCACerts, defThingsCACerts),
		URL:        mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		ClientName: clients.Things,
	}

	return config{
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		smppConf:          smppConf,
		dbConfig:          dbConfig,
		httpConfig:        httpConfig,
		thingsConfig:      thingsConfig,
		from:              mainflux.Env(envFrom, defFrom),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
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

func subscribeToThingsES(ctx context.Context, svc notifiers.Service, cfg config, logger logger.Logger) error {
	subscriber, err := mfevents.NewSubscriber(cfg.esURL, thingsStream, esGroupName, cfg.esConsumerName, logger)
	if err != nil {
		return err
	}

	handler := events.NewEventHandler(svc)

	if err := subscriber.Subscribe(ctx, handler); err != nil {
		return err
	}

	logger.Info("Subscribed to Redis Event Store")

	return nil
}

func newService(c config, logger logger.Logger, dbTracer opentracing.Tracer, db *sqlx.DB, tc protomfx.ThingsServiceClient) notifiers.Service {
	idp := uuid.New()
	database := dbutil.NewDatabase(db)

	notifier := mfsmpp.New(c.smppConf, c.from)
	notifierRepo := postgres.NewNotifierRepository(database)
	notifierRepo = tracing.NotifierRepositoryMiddleware(dbTracer, notifierRepo)
	svc := notifiers.New(idp, notifier, notifierRepo, tc)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "notifier",
			Subsystem: "smpp",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "notifier",
			Subsystem: "smpp",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}
