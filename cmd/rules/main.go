package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/events/redis"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	serversgrpc "github.com/MainfluxLabs/mainflux/pkg/servers/grpc"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/MainfluxLabs/mainflux/rules/api"
	"github.com/MainfluxLabs/mainflux/rules/api/events"
	httpapi "github.com/MainfluxLabs/mainflux/rules/api/http"
	"github.com/MainfluxLabs/mainflux/rules/postgres"
	"github.com/MainfluxLabs/mainflux/rules/tracing"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "rules"
	stopWaitTime = 5 * time.Second
	thingsStream = "mainflux.things"
	esGroupName  = svcName

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
	defHTTPPort          = "9027"
	defGRPCPort          = "8186"
	defJaegerURL         = ""
	defServerCert        = ""
	defServerKey         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defESURL             = "redis://localhost:6379/0"
	defESConsumerName    = svcName

	envBrokerURL         = "MF_BROKER_URL"
	envLogLevel          = "MF_RULES_LOG_LEVEL"
	envDBHost            = "MF_RULES_DB_HOST"
	envDBPort            = "MF_RULES_DB_PORT"
	envDBUser            = "MF_RULES_DB_USER"
	envDBPass            = "MF_RULES_DB_PASS"
	envDB                = "MF_RULES_DB"
	envDBSSLMode         = "MF_RULES_DB_SSL_MODE"
	envDBSSLCert         = "MF_RULES_DB_SSL_CERT"
	envDBSSLKey          = "MF_RULES_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_RULES_DB_SSL_ROOT_CERT"
	envClientTLS         = "MF_RULES_CLIENT_TLS"
	envCACerts           = "MF_RULES_CA_CERTS"
	envHTTPPort          = "MF_RULES_HTTP_PORT"
	envGRPCPort          = "MF_RULES_GRPC_PORT"
	envServerCert        = "MF_RULES_SERVER_CERT"
	envServerKey         = "MF_RULES_SERVER_KEY"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envESURL             = "MF_RULES_ES_URL"
	envESConsumerName    = "MF_RULES_EVENT_CONSUMER"
)

type config struct {
	brokerURL         string
	logLevel          string
	dbConfig          postgres.Config
	httpConfig        servers.Config
	grpcConfig        servers.Config
	thingsConfig      clients.Config
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

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	rulesHttpTracer, rulesHttpCloser := jaeger.Init("rules_http", cfg.jaegerURL, logger)
	defer rulesHttpCloser.Close()

	rulesGrpcTracer, rulesGrpcCloser := jaeger.Init("rules_grpc", cfg.jaegerURL, logger)
	defer rulesGrpcCloser.Close()

	dbTracer, dbCloser := jaeger.Init("rules_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	thConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer thConn.Close()

	thingsTracer, thingsCloser := jaeger.Init("rules_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	pub, err := brokers.NewPublisher(cfg.brokerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pub.Close()

	tc := thingsapi.NewClient(thConn, thingsTracer, cfg.thingsGRPCTimeout)

	svc := newService(dbTracer, db, tc, pub, logger)

	if err = subscribeToThingsES(ctx, svc, cfg, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to subscribe to things event store: %s", err))
		return
	}

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(rulesHttpTracer, svc, logger), cfg.httpConfig, logger)
	})
	g.Go(func() error {
		return serversgrpc.Start(ctx, rulesGrpcTracer, svc, cfg.grpcConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Authentication service shutdown by signal: %s", sig))
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Authentication service terminated: %s", err))
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

	grpcConfig := servers.Config{
		ServerName:   svcName,
		ServerCert:   mainflux.Env(envServerCert, defServerCert),
		ServerKey:    mainflux.Env(envServerKey, defServerKey),
		Port:         mainflux.Env(envGRPCPort, defGRPCPort),
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
		grpcConfig:        grpcConfig,
		thingsConfig:      thingsConfig,
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

func subscribeToThingsES(ctx context.Context, svc rules.Service, cfg config, logger logger.Logger) error {
	subscriber, err := redis.NewSubscriber(cfg.esURL, thingsStream, esGroupName, cfg.esConsumerName, logger)
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

func newService(dbTracer opentracing.Tracer, db *sqlx.DB, tc protomfx.ThingsServiceClient, pub messaging.Publisher, logger logger.Logger) rules.Service {
	database := dbutil.NewDatabase(db)

	rulesRepo := postgres.NewRuleRepository(database)
	rulesRepo = tracing.RuleRepositoryMiddleware(dbTracer, rulesRepo)

	idProvider := uuid.New()
	svc := rules.New(rulesRepo, tc, pub, idProvider, logger)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "rules",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "rules",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)
	return svc
}
