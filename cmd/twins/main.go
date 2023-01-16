// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	localusers "github.com/MainfluxLabs/mainflux/things/standalone"
	"github.com/MainfluxLabs/mainflux/twins"
	"github.com/MainfluxLabs/mainflux/twins/api"
	twapi "github.com/MainfluxLabs/mainflux/twins/api/http"
	twmongodb "github.com/MainfluxLabs/mainflux/twins/mongodb"
	rediscache "github.com/MainfluxLabs/mainflux/twins/redis"
	"github.com/MainfluxLabs/mainflux/twins/tracing"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-redis/redis/v8"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	svcName            = "twins"
	queue              = "twins"
	stopWaitTime       = 5 * time.Second
	defLogLevel        = "error"
	defHTTPPort        = "8180"
	defJaegerURL       = ""
	defServerCert      = ""
	defServerKey       = ""
	defDB              = "mainfluxlabs-twins"
	defDBHost          = "localhost"
	defDBPort          = "27017"
	defCacheURL        = "localhost:6379"
	defCachePass       = ""
	defCacheDB         = "0"
	defStandaloneEmail = ""
	defStandaloneToken = ""
	defClientTLS       = "false"
	defCACerts         = ""
	defChannelID       = ""
	defBrokerURL       = "nats://localhost:4222"
	defAuthURL         = "localhost:8181"
	defAuthTimeout     = "1s"

	envLogLevel        = "MF_TWINS_LOG_LEVEL"
	envHTTPPort        = "MF_TWINS_HTTP_PORT"
	envJaegerURL       = "MF_JAEGER_URL"
	envServerCert      = "MF_TWINS_SERVER_CERT"
	envServerKey       = "MF_TWINS_SERVER_KEY"
	envDB              = "MF_TWINS_DB"
	envDBHost          = "MF_TWINS_DB_HOST"
	envDBPort          = "MF_TWINS_DB_PORT"
	envCacheURL        = "MF_TWINS_CACHE_URL"
	envCachePass       = "MF_TWINS_CACHE_PASS"
	envCacheDB         = "MF_TWINS_CACHE_DB"
	envStandaloneEmail = "MF_TWINS_STANDALONE_EMAIL"
	envStandaloneToken = "MF_TWINS_STANDALONE_TOKEN"
	envClientTLS       = "MF_TWINS_CLIENT_TLS"
	envCACerts         = "MF_TWINS_CA_CERTS"
	envChannelID       = "MF_TWINS_CHANNEL_ID"
	envBrokerURL       = "MF_BROKER_URL"
	envAuthURL         = "MF_AUTH_GRPC_URL"
	envAuthTimeout     = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	logLevel        string
	httpPort        string
	jaegerURL       string
	serverCert      string
	serverKey       string
	dbCfg           twmongodb.Config
	cacheURL        string
	cachePass       string
	cacheDB         string
	standaloneEmail string
	standaloneToken string
	clientTLS       bool
	caCerts         string
	channelID       string
	brokerURL       string

	authURL     string
	authTimeout time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	cacheClient := connectToRedis(cfg.cacheURL, cfg.cachePass, cfg.cacheDB, logger)
	cacheTracer, cacheCloser := initJaeger("twins_cache", cfg.jaegerURL, logger)
	defer cacheCloser.Close()

	db, err := twmongodb.Connect(cfg.dbCfg, logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	dbTracer, dbCloser := initJaeger("twins_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	authTracer, authCloser := initJaeger("auth", cfg.jaegerURL, logger)
	defer authCloser.Close()
	auth, _ := createAuthClient(cfg, authTracer, logger)

	pubSub, err := brokers.NewPubSub(cfg.brokerURL, queue, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	svc := newService(svcName, pubSub, cfg.channelID, auth, dbTracer, db, cacheTracer, cacheClient, logger)

	tracer, closer := initJaeger("twins", cfg.jaegerURL, logger)
	defer closer.Close()

	g.Go(func() error {
		return startHTTPServer(ctx, twapi.MakeHandler(tracer, svc, logger), cfg.httpPort, cfg, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Twins service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Twins service terminated: %s", err))
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

	dbCfg := twmongodb.Config{
		Name: mainflux.Env(envDB, defDB),
		Host: mainflux.Env(envDBHost, defDBHost),
		Port: mainflux.Env(envDBPort, defDBPort),
	}

	return config{
		logLevel:        mainflux.Env(envLogLevel, defLogLevel),
		httpPort:        mainflux.Env(envHTTPPort, defHTTPPort),
		serverCert:      mainflux.Env(envServerCert, defServerCert),
		serverKey:       mainflux.Env(envServerKey, defServerKey),
		jaegerURL:       mainflux.Env(envJaegerURL, defJaegerURL),
		dbCfg:           dbCfg,
		cacheURL:        mainflux.Env(envCacheURL, defCacheURL),
		cachePass:       mainflux.Env(envCachePass, defCachePass),
		cacheDB:         mainflux.Env(envCacheDB, defCacheDB),
		standaloneEmail: mainflux.Env(envStandaloneEmail, defStandaloneEmail),
		standaloneToken: mainflux.Env(envStandaloneToken, defStandaloneToken),
		clientTLS:       tls,
		caCerts:         mainflux.Env(envCACerts, defCACerts),
		channelID:       mainflux.Env(envChannelID, defChannelID),
		brokerURL:       mainflux.Env(envBrokerURL, defBrokerURL),
		authURL:         mainflux.Env(envAuthURL, defAuthURL),
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

	conn, err := grpc.Dial(cfg.authURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to auth service: %s", err))
		os.Exit(1)
	}

	return conn
}

func connectToRedis(cacheURL, cachePass, cacheDB string, logger logger.Logger) *redis.Client {
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

func newService(id string, ps messaging.PubSub, chanID string, users mainflux.AuthServiceClient, dbTracer opentracing.Tracer, db *mongo.Database, cacheTracer opentracing.Tracer, cacheClient *redis.Client, logger logger.Logger) twins.Service {
	twinRepo := twmongodb.NewTwinRepository(db)
	twinRepo = tracing.TwinRepositoryMiddleware(dbTracer, twinRepo)

	stateRepo := twmongodb.NewStateRepository(db)
	stateRepo = tracing.StateRepositoryMiddleware(dbTracer, stateRepo)

	idProvider := uuid.New()
	twinCache := rediscache.NewTwinCache(cacheClient)
	twinCache = tracing.TwinCacheMiddleware(cacheTracer, twinCache)

	svc := twins.New(ps, users, twinRepo, twinCache, stateRepo, idProvider, chanID, logger)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "twins",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "twins",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)
	err := ps.Subscribe(id, brokers.SubjectAllChannels, handle(logger, chanID, svc))
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	return svc
}

func handle(logger logger.Logger, chanID string, svc twins.Service) handlerFunc {
	return func(msg messaging.Message) error {
		if msg.Channel == chanID {
			return nil
		}

		if err := svc.SaveStates(&msg); err != nil {
			logger.Error(fmt.Sprintf("State save failed: %s", err))
			return err
		}

		return nil
	}
}

func startHTTPServer(ctx context.Context, handler http.Handler, port string, cfg config, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", port)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: handler}

	switch {
	case cfg.serverCert != "" || cfg.serverKey != "":
		logger.Info(fmt.Sprintf("Twins service started using https on port %s with cert %s key %s",
			port, cfg.serverCert, cfg.serverKey))
		go func() {
			errCh <- server.ListenAndServeTLS(cfg.serverCert, cfg.serverKey)
		}()
	default:
		logger.Info(fmt.Sprintf("Twins service started using http on port %s", cfg.httpPort))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("Twins service error occurred during shutdown at %s: %s", p, err))
			return fmt.Errorf("twins service occurred during shutdown at %s: %w", p, err)
		}
		logger.Info(fmt.Sprintf("Twins service shutdown of http at %s", p))
		return nil
	case err := <-errCh:
		return err
	}
}

type handlerFunc func(msg messaging.Message) error

func (h handlerFunc) Handle(msg messaging.Message) error {
	return h(msg)
}

func (h handlerFunc) Cancel() error {
	return nil
}
