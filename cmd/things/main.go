// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/api"
	authgrpcapi "github.com/MainfluxLabs/mainflux/things/api/auth/grpc"
	authhttpapi "github.com/MainfluxLabs/mainflux/things/api/auth/http"
	thhttpapi "github.com/MainfluxLabs/mainflux/things/api/things/http"
	"github.com/MainfluxLabs/mainflux/things/postgres"
	rediscache "github.com/MainfluxLabs/mainflux/things/redis"
	localusers "github.com/MainfluxLabs/mainflux/things/standalone"
	"github.com/MainfluxLabs/mainflux/things/tracing"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	stopWaitTime = 5 * time.Second

	defLogLevel        = "error"
	defDBHost          = "localhost"
	defDBPort          = "5432"
	defDBUser          = "mainflux"
	defDBPass          = "mainflux"
	defDB              = "things"
	defDBSSLMode       = "disable"
	defDBSSLCert       = ""
	defDBSSLKey        = ""
	defDBSSLRootCert   = ""
	defClientTLS       = "false"
	defCACerts         = ""
	defCacheURL        = "localhost:6379"
	defCachePass       = ""
	defCacheDB         = "0"
	defESURL           = "localhost:6379"
	defESPass          = ""
	defESDB            = "0"
	defHTTPPort        = "8182"
	defAuthHTTPPort    = "8989"
	defAuthGRPCPort    = "8181"
	defServerCert      = ""
	defServerKey       = ""
	defStandaloneEmail = ""
	defStandaloneToken = ""
	defJaegerURL       = ""
	defAuthGRPCURL     = "localhost:8181"
	defAuthTimeout     = "1s"

	envLogLevel        = "MF_THINGS_LOG_LEVEL"
	envDBHost          = "MF_THINGS_DB_HOST"
	envDBPort          = "MF_THINGS_DB_PORT"
	envDBUser          = "MF_THINGS_DB_USER"
	envDBPass          = "MF_THINGS_DB_PASS"
	envDB              = "MF_THINGS_DB"
	envDBSSLMode       = "MF_THINGS_DB_SSL_MODE"
	envDBSSLCert       = "MF_THINGS_DB_SSL_CERT"
	envDBSSLKey        = "MF_THINGS_DB_SSL_KEY"
	envDBSSLRootCert   = "MF_THINGS_DB_SSL_ROOT_CERT"
	envClientTLS       = "MF_THINGS_CLIENT_TLS"
	envCACerts         = "MF_THINGS_CA_CERTS"
	envCacheURL        = "MF_THINGS_CACHE_URL"
	envCachePass       = "MF_THINGS_CACHE_PASS"
	envCacheDB         = "MF_THINGS_CACHE_DB"
	envESURL           = "MF_THINGS_ES_URL"
	envESPass          = "MF_THINGS_ES_PASS"
	envESDB            = "MF_THINGS_ES_DB"
	envHTTPPort        = "MF_THINGS_HTTP_PORT"
	envAuthHTTPPort    = "MF_THINGS_AUTH_HTTP_PORT"
	envAuthGRPCPort    = "MF_THINGS_AUTH_GRPC_PORT"
	envServerCert      = "MF_THINGS_SERVER_CERT"
	envServerKey       = "MF_THINGS_SERVER_KEY"
	envStandaloneEmail = "MF_THINGS_STANDALONE_EMAIL"
	envStandaloneToken = "MF_THINGS_STANDALONE_TOKEN"
	envJaegerURL       = "MF_JAEGER_URL"
	envAuthGRPCURL     = "MF_AUTH_GRPC_URL"
	envAuthTimeout     = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	logLevel        string
	dbConfig        postgres.Config
	clientTLS       bool
	caCerts         string
	cacheURL        string
	cachePass       string
	cacheDB         string
	esURL           string
	esPass          string
	esDB            string
	httpPort        string
	authHTTPPort    string
	authGRPCPort    string
	serverCert      string
	serverKey       string
	standaloneEmail string
	standaloneToken string
	jaegerURL       string
	authGRPCURL     string
	authTimeout     time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	cacheClient := connectToRedis(cfg.cacheURL, cfg.cachePass, cfg.cacheDB, logger)

	esClient := connectToRedis(cfg.esURL, cfg.esPass, cfg.esDB, logger)

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	authTracer, authCloser := initJaeger("auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	auth, close := createAuthClient(cfg, authTracer, logger)
	if close != nil {
		defer close()
	}

	dbTracer, dbCloser := initJaeger("things_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	cacheTracer, cacheCloser := initJaeger("things_cache", cfg.jaegerURL, logger)
	defer cacheCloser.Close()

	svc := newService(auth, dbTracer, cacheTracer, db, cacheClient, esClient, logger)

	g.Go(func() error {
		return startHTTPServer(ctx, "thing-http", thhttpapi.MakeHandler(thingsTracer, svc, logger), cfg.httpPort, cfg, logger)
	})

	g.Go(func() error {
		return startHTTPServer(ctx, "auth-http", authhttpapi.MakeHandler(thingsTracer, svc, logger), cfg.authHTTPPort, cfg, logger)
	})

	g.Go(func() error {
		return startGRPCServer(ctx, svc, thingsTracer, cfg, logger)
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

	authTimeout, err := time.ParseDuration(mainflux.Env(envAuthTimeout, defAuthTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthTimeout, err.Error())
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

	return config{
		logLevel:        mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:        dbConfig,
		clientTLS:       tls,
		caCerts:         mainflux.Env(envCACerts, defCACerts),
		cacheURL:        mainflux.Env(envCacheURL, defCacheURL),
		cachePass:       mainflux.Env(envCachePass, defCachePass),
		cacheDB:         mainflux.Env(envCacheDB, defCacheDB),
		esURL:           mainflux.Env(envESURL, defESURL),
		esPass:          mainflux.Env(envESPass, defESPass),
		esDB:            mainflux.Env(envESDB, defESDB),
		httpPort:        mainflux.Env(envHTTPPort, defHTTPPort),
		authHTTPPort:    mainflux.Env(envAuthHTTPPort, defAuthHTTPPort),
		authGRPCPort:    mainflux.Env(envAuthGRPCPort, defAuthGRPCPort),
		serverCert:      mainflux.Env(envServerCert, defServerCert),
		serverKey:       mainflux.Env(envServerKey, defServerKey),
		standaloneEmail: mainflux.Env(envStandaloneEmail, defStandaloneEmail),
		standaloneToken: mainflux.Env(envStandaloneToken, defStandaloneToken),
		jaegerURL:       mainflux.Env(envJaegerURL, defJaegerURL),
		authGRPCURL:     mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		authTimeout:     authTimeout,
	}
}

func initJaeger(svcName, url string, logger logger.Logger) (opentracing.Tracer, io.Closer) {
	if url == "" {
		return opentracing.NoopTracer{}, ioutil.NopCloser(nil)
	}

	tracer, closer, err := jconfig.Configuration{
		ServiceName: svcName,
		Sampler: &jconfig.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jconfig.ReporterConfig{
			LocalAgentHostPort: url,
			LogSpans:           true,
		},
	}.NewTracer()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to init Jaeger client: %s", err))
		os.Exit(1)
	}

	return tracer, closer
}

func connectToRedis(cacheURL, cachePass string, cacheDB string, logger logger.Logger) *redis.Client {
	db, err := strconv.Atoi(cacheDB)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to cache: %s", err))
		os.Exit(1)
	}

	return redis.NewClient(&redis.Options{
		Addr:     cacheURL,
		Password: cachePass,
		DB:       db,
	})
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func createAuthClient(cfg config, tracer opentracing.Tracer, logger logger.Logger) (mainflux.AuthServiceClient, func() error) {
	if cfg.standaloneEmail != "" && cfg.standaloneToken != "" {
		return localusers.NewAuthService(cfg.standaloneEmail, cfg.standaloneToken), nil
	}

	conn := connectToAuth(cfg, logger)
	return authapi.NewClient(tracer, conn, cfg.authTimeout), conn.Close
}

func connectToAuth(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.clientTLS {
		if cfg.caCerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.caCerts, "")
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to create tls credentials: %s", err))
				os.Exit(1)
			}
			opts = append(opts, grpc.WithTransportCredentials(tpc))
		}
	} else {
		opts = append(opts, grpc.WithInsecure())
		logger.Info("gRPC communication is not encrypted")
	}

	conn, err := grpc.Dial(cfg.authGRPCURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to auth service: %s", err))
		os.Exit(1)
	}

	return conn
}

func newService(auth mainflux.AuthServiceClient, dbTracer opentracing.Tracer, cacheTracer opentracing.Tracer, db *sqlx.DB, cacheClient *redis.Client, esClient *redis.Client, logger logger.Logger) things.Service {
	database := postgres.NewDatabase(db)

	thingsRepo := postgres.NewThingRepository(database)
	thingsRepo = tracing.ThingRepositoryMiddleware(dbTracer, thingsRepo)

	channelsRepo := postgres.NewChannelRepository(database)
	channelsRepo = tracing.ChannelRepositoryMiddleware(dbTracer, channelsRepo)

	groupsRepo := postgres.NewGroupRepo(database)
	groupsRepo = tracing.GroupRepositoryMiddleware(dbTracer, groupsRepo)

	chanCache := rediscache.NewChannelCache(cacheClient)
	chanCache = tracing.ChannelCacheMiddleware(cacheTracer, chanCache)

	thingCache := rediscache.NewThingCache(cacheClient)
	thingCache = tracing.ThingCacheMiddleware(cacheTracer, thingCache)
	idProvider := uuid.New()

	svc := things.New(auth, thingsRepo, channelsRepo, groupsRepo, chanCache, thingCache, idProvider)
	svc = rediscache.NewEventStoreMiddleware(svc, esClient)
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

func startHTTPServer(ctx context.Context, typ string, handler http.Handler, port string, cfg config, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", port)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: handler}

	switch {
	case cfg.serverCert != "" || cfg.serverKey != "":
		logger.Info(fmt.Sprintf("Things %s service started using https on port %s with cert %s key %s",
			typ, port, cfg.serverCert, cfg.serverKey))
		go func() {
			errCh <- server.ListenAndServeTLS(cfg.serverCert, cfg.serverKey)
		}()
	default:
		logger.Info(fmt.Sprintf("Things %s service started using http on port %s", typ, cfg.httpPort))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("Things %s service error occurred during shutdown at %s: %s", typ, p, err))
			return fmt.Errorf("things %s service occurred during shutdown at %s: %w", typ, p, err)
		}
		logger.Info(fmt.Sprintf("Things %s service  shutdown of http at %s", typ, p))
		return nil
	case err := <-errCh:
		return err
	}

}

func startGRPCServer(ctx context.Context, svc things.Service, tracer opentracing.Tracer, cfg config, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", cfg.authGRPCPort)
	errCh := make(chan error)
	var server *grpc.Server

	listener, err := net.Listen("tcp", p)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", cfg.authGRPCPort, err)
	}

	switch {
	case cfg.serverCert != "" || cfg.serverKey != "":
		creds, err := credentials.NewServerTLSFromFile(cfg.serverCert, cfg.serverKey)
		if err != nil {
			return fmt.Errorf("failed to load things certificates: %w", err)
		}
		logger.Info(fmt.Sprintf("Things gRPC service started using https on port %s with cert %s key %s",
			cfg.authGRPCPort, cfg.serverCert, cfg.serverKey))
		server = grpc.NewServer(grpc.Creds(creds))
	default:
		logger.Info(fmt.Sprintf("Things gRPC service started using http on port %s", cfg.authGRPCPort))
		server = grpc.NewServer()
	}

	mainflux.RegisterThingsServiceServer(server, authgrpcapi.NewServer(tracer, svc))
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		c := make(chan bool)
		go func() {
			defer close(c)
			server.GracefulStop()
		}()
		select {
		case <-c:
		case <-time.After(stopWaitTime):
		}
		logger.Info(fmt.Sprintf("Things gRPC service shutdown at %s", p))
		return nil
	case err := <-errCh:
		return err
	}
}
