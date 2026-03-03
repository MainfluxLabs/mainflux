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
	"github.com/MainfluxLabs/mainflux/pkg/email"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfevents "github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	serversgrpc "github.com/MainfluxLabs/mainflux/pkg/servers/grpc"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/api"
	httpapi "github.com/MainfluxLabs/mainflux/things/api/http"
	"github.com/MainfluxLabs/mainflux/things/emailer"
	"github.com/MainfluxLabs/mainflux/things/postgres"
	rediscache "github.com/MainfluxLabs/mainflux/things/redis"
	"github.com/MainfluxLabs/mainflux/things/redis/events"
	localusers "github.com/MainfluxLabs/mainflux/things/standalone"
	"github.com/MainfluxLabs/mainflux/things/tracing"
	usersapi "github.com/MainfluxLabs/mainflux/users/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	stopWaitTime = 5 * time.Second
	svcName      = "things"
	esGroupName  = svcName

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
	defClientTLS       = "false"
	defCACerts         = ""
	defCacheURL        = "redis://localhost:6379/0"
	defESConsumerName  = svcName
	defESURL           = "redis://localhost:6379/0"
	defHTTPPort        = "8182"
	defAuthHTTPPort    = "8989"
	defAuthGRPCPort    = "8183"
	defServerCert      = ""
	defServerKey       = ""
	defStandaloneEmail = ""
	defStandaloneToken = ""
	defJaegerURL       = ""
	defAuthGRPCURL     = "localhost:8181"
	defAuthGRPCTimeout = "1s"
	defUsersCACerts    = ""
	defUsersClientTLS  = "false"
	defUsersGRPCURL    = "localhost:8184"
	defTimeout         = "1s"

	defEmailHost         = "localhost"
	defEmailPort         = "25"
	defEmailUsername     = "root"
	defEmailPassword     = ""
	defEmailFromAddress  = ""
	defEmailFromName     = ""
	defEmailBaseTemplate = "base.tmpl"

	defHost = "http://localhost"

	envLogLevel         = "MF_THINGS_LOG_LEVEL"
	envDBHost           = "MF_THINGS_DB_HOST"
	envDBPort           = "MF_THINGS_DB_PORT"
	envDBUser           = "MF_THINGS_DB_USER"
	envDBPass           = "MF_THINGS_DB_PASS"
	envDB               = "MF_THINGS_DB"
	envDBSSLMode        = "MF_THINGS_DB_SSL_MODE"
	envDBSSLCert        = "MF_THINGS_DB_SSL_CERT"
	envDBSSLKey         = "MF_THINGS_DB_SSL_KEY"
	envDBSSLRootCert    = "MF_THINGS_DB_SSL_ROOT_CERT"
	envClientTLS        = "MF_THINGS_CLIENT_TLS"
	envCACerts          = "MF_THINGS_CA_CERTS"
	envCacheURL         = "MF_THINGS_CACHE_URL"
	envESConsumerName   = "MF_THINGS_EVENT_CONSUMER"
	envESURL            = "MF_THINGS_ES_URL"
	envHTTPPort         = "MF_THINGS_HTTP_PORT"
	envAuthHTTPPort     = "MF_THINGS_AUTH_HTTP_PORT"
	envAuthGRPCPort     = "MF_THINGS_AUTH_GRPC_PORT"
	envServerCert       = "MF_THINGS_SERVER_CERT"
	envServerKey        = "MF_THINGS_SERVER_KEY"
	envStandaloneEmail  = "MF_THINGS_STANDALONE_EMAIL"
	envStandaloneToken  = "MF_THINGS_STANDALONE_TOKEN"
	envJaegerURL        = "MF_JAEGER_URL"
	envAuthGRPCURL      = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout  = "MF_AUTH_GRPC_TIMEOUT"
	envUsersGRPCURL     = "MF_USERS_GRPC_URL"
	envUsersCACerts     = "MF_USERS_CA_CERTS"
	envUsersClientTLS   = "MF_USERS_CLIENT_TLS"
	envUsersGRPCTimeout = "MF_USERS_GRPC_TIMEOUT"

	envEmailHost         = "MF_EMAIL_HOST"
	envEmailPort         = "MF_EMAIL_PORT"
	envEmailUsername     = "MF_EMAIL_USERNAME"
	envEmailPassword     = "MF_EMAIL_PASSWORD"
	envEmailFromAddress  = "MF_EMAIL_FROM_ADDRESS"
	envEmailFromName     = "MF_EMAIL_FROM_NAME"
	envEmailBaseTemplate = "MF_EMAIL_BASE_TEMPLATE"

	envHost = "MF_HOST"
)

type config struct {
	logLevel         string
	dbConfig         postgres.Config
	httpConfig       servers.Config
	authHttpConfig   servers.Config
	grpcConfig       servers.Config
	authConfig       clients.Config
	usersConfig      clients.Config
	emailConfig      email.Config
	cacheURL         string
	esConsumerName   string
	esURL            string
	standaloneEmail  string
	standaloneToken  string
	jaegerURL        string
	authGRPCTimeout  time.Duration
	usersGRPCTimeout time.Duration
	host             string
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err)
	}

	thingsHttpTracer, thingsHttpCloser := jaeger.Init("things_http", cfg.jaegerURL, logger)
	defer thingsHttpCloser.Close()

	thingsGrpcTracer, thingsGrpcCloser := jaeger.Init("things_grpc", cfg.jaegerURL, logger)
	defer thingsGrpcCloser.Close()

	cacheClient := connectToRedis(cfg.cacheURL, logger)

	esClient := connectToRedis(cfg.esURL, logger)

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	authTracer, authCloser := jaeger.Init("things_auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	auth, close := createAuthClient(cfg, authTracer, logger)
	if close != nil {
		defer close()
	}

	dbTracer, dbCloser := jaeger.Init("things_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	cacheTracer, cacheCloser := jaeger.Init("things_cache", cfg.jaegerURL, logger)
	defer cacheCloser.Close()

	usrConn := clientsgrpc.Connect(cfg.usersConfig, logger)
	defer usrConn.Close()

	usersTracer, usersCloser := jaeger.Init("things_users", cfg.jaegerURL, logger)
	defer usersCloser.Close()

	users := usersapi.NewClient(usrConn, usersTracer, cfg.usersGRPCTimeout)

	svc := newService(auth, users, dbTracer, cacheTracer, db, cacheClient, esClient, logger, cfg)

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(svc, thingsHttpTracer, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(svc, thingsHttpTracer, logger), cfg.authHttpConfig, logger)
	})

	g.Go(func() error {
		return serversgrpc.Start(ctx, thingsGrpcTracer, svc, cfg.grpcConfig, logger)
	})

	g.Go(func() error {
		return subscribeToAuthES(ctx, svc, cfg, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Things service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Things service terminated: %s", err))
	}
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	authGRPCTimeout, err := time.ParseDuration(mainflux.Env(envAuthGRPCTimeout, defAuthGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthGRPCTimeout, err.Error())
	}

	usersClientTLS, err := strconv.ParseBool(mainflux.Env(envUsersClientTLS, defUsersClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	usersGRPCTimeout, err := time.ParseDuration(mainflux.Env(envUsersGRPCTimeout, defTimeout))
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

	authHttpConfig := servers.Config{
		ServerName:   svcName,
		ServerCert:   mainflux.Env(envServerCert, defServerCert),
		ServerKey:    mainflux.Env(envServerKey, defServerKey),
		Port:         mainflux.Env(envAuthHTTPPort, defAuthHTTPPort),
		StopWaitTime: stopWaitTime,
	}

	grpcConfig := servers.Config{
		ServerName:   svcName,
		ServerCert:   mainflux.Env(envServerCert, defServerCert),
		ServerKey:    mainflux.Env(envServerKey, defServerKey),
		Port:         mainflux.Env(envAuthGRPCPort, defAuthGRPCPort),
		StopWaitTime: stopWaitTime,
	}

	authConfig := clients.Config{
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envCACerts, defCACerts),
		URL:        mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		ClientName: clients.Auth,
	}

	usersConfig := clients.Config{
		ClientTLS:  usersClientTLS,
		CaCerts:    mainflux.Env(envUsersCACerts, defUsersCACerts),
		URL:        mainflux.Env(envUsersGRPCURL, defUsersGRPCURL),
		ClientName: clients.Users,
	}

	emailConfig := email.Config{
		FromAddress:      mainflux.Env(envEmailFromAddress, defEmailFromAddress),
		FromName:         mainflux.Env(envEmailFromName, defEmailFromName),
		Host:             mainflux.Env(envEmailHost, defEmailHost),
		Port:             mainflux.Env(envEmailPort, defEmailPort),
		Username:         mainflux.Env(envEmailUsername, defEmailUsername),
		Password:         mainflux.Env(envEmailPassword, defEmailPassword),
		BaseTemplatePath: mainflux.Env(envEmailBaseTemplate, defEmailBaseTemplate),
	}

	return config{
		logLevel:         mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:         dbConfig,
		httpConfig:       httpConfig,
		authHttpConfig:   authHttpConfig,
		grpcConfig:       grpcConfig,
		authConfig:       authConfig,
		usersConfig:      usersConfig,
		emailConfig:      emailConfig,
		cacheURL:         mainflux.Env(envCacheURL, defCacheURL),
		esConsumerName:   mainflux.Env(envESConsumerName, defESConsumerName),
		esURL:            mainflux.Env(envESURL, defESURL),
		standaloneEmail:  mainflux.Env(envStandaloneEmail, defStandaloneEmail),
		standaloneToken:  mainflux.Env(envStandaloneToken, defStandaloneToken),
		jaegerURL:        mainflux.Env(envJaegerURL, defJaegerURL),
		authGRPCTimeout:  authGRPCTimeout,
		usersGRPCTimeout: usersGRPCTimeout,
		host:             mainflux.Env(envHost, defHost),
	}
}

func connectToRedis(redisURL string, logger logger.Logger) *redis.Client {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to redis: %s", err))
		os.Exit(1)
	}

	return redis.NewClient(opts)
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func createAuthClient(cfg config, tracer opentracing.Tracer, logger logger.Logger) (protomfx.AuthServiceClient, func() error) {
	if cfg.standaloneEmail != "" && cfg.standaloneToken != "" {
		return localusers.NewAuthService(cfg.standaloneEmail, cfg.standaloneToken), nil
	}

	conn := clientsgrpc.Connect(cfg.authConfig, logger)
	return authapi.NewClient(conn, tracer, cfg.authGRPCTimeout), conn.Close
}

func subscribeToAuthES(ctx context.Context, svc things.Service, cfg config, logger logger.Logger) error {
	subscriber, err := mfevents.NewSubscriber(cfg.esURL, mfevents.AuthStream, esGroupName, cfg.esConsumerName, logger)
	if err != nil {
		return err
	}

	defer func() {
		if err := subscriber.Close(); err != nil {
			logger.Error(fmt.Sprintf("Failed to close auth subscriber: %s", err))
		}
	}()

	handler := events.NewEventHandler(svc)

	return subscriber.Subscribe(ctx, handler)
}

func newService(ac protomfx.AuthServiceClient, uc protomfx.UsersServiceClient, dbTracer opentracing.Tracer,
	cacheTracer opentracing.Tracer, db *sqlx.DB, cacheClient *redis.Client, esClient *redis.Client,
	logger logger.Logger, cfg config) things.Service {

	database := dbutil.NewDatabase(db)

	thingsRepo := postgres.NewThingRepository(database)
	thingsRepo = tracing.ThingRepositoryMiddleware(dbTracer, thingsRepo)

	profilesRepo := postgres.NewProfileRepository(database)
	profilesRepo = tracing.ProfileRepositoryMiddleware(dbTracer, profilesRepo)

	groupsRepo := postgres.NewGroupRepository(database)
	groupsRepo = tracing.GroupRepositoryMiddleware(dbTracer, groupsRepo)

	profileCache := rediscache.NewProfileCache(cacheClient)
	profileCache = tracing.ProfileCacheMiddleware(cacheTracer, profileCache)

	thingCache := rediscache.NewThingCache(cacheClient)
	thingCache = tracing.ThingCacheMiddleware(cacheTracer, thingCache)

	groupCache := rediscache.NewGroupCache(cacheClient)
	groupCache = tracing.GroupCacheMiddleware(cacheTracer, groupCache)
	idProvider := uuid.New()

	groupMembershipsRepo := postgres.NewGroupMembershipsRepository(db)
	groupMembershipsRepo = tracing.GroupMembershipsRepositoryMiddleware(dbTracer, groupMembershipsRepo)

	thingsEmailer, err := emailer.New(cfg.host, &cfg.emailConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to configure e-mailing util: %s", err.Error()))
	}

	thingsEmailer = emailer.LoggingMiddleware(thingsEmailer, logger)
	thingsEmailer = emailer.MetricsMiddleware(
		thingsEmailer,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "things",
			Subsystem: "email",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "things",
			Subsystem: "email",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	svc := things.New(ac, uc, thingsRepo, profilesRepo, groupsRepo, groupMembershipsRepo, profileCache, thingCache, groupCache, idProvider, thingsEmailer)

	svc = events.NewEventStoreMiddleware(svc, esClient)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "things",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "things",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)
	return svc
}
