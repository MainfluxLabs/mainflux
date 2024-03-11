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
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	localusers "github.com/MainfluxLabs/mainflux/things/standalone"
	"github.com/MainfluxLabs/mainflux/webhooks/postgres"
	"github.com/MainfluxLabs/mainflux/webhooks/tracing"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/webhooks"
	"github.com/MainfluxLabs/mainflux/webhooks/api"
	webhookshttpapi "github.com/MainfluxLabs/mainflux/webhooks/api/webhooks/http"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
)

const (
	stopWaitTime       = 5 * time.Second
	defLogLevel        = "error"
	defDBHost          = "localhost"
	defDBPort          = "5432"
	defDBUser          = "mainflux"
	defDBPass          = "mainflux"
	defDB              = "webhooks"
	defDBSSLMode       = "disable"
	defDBSSLCert       = ""
	defDBSSLKey        = ""
	defDBSSLRootCert   = ""
	defClientTLS       = "false"
	defCACerts         = ""
	defHTTPPort        = "9021"
	defAuthHTTPPort    = "8989"
	defAuthGRPCPort    = "8181"
	defJaegerURL       = ""
	defServerCert      = ""
	defServerKey       = ""
	defStandaloneEmail = ""
	defStandaloneToken = ""
	defAuthGRPCURL     = "localhost:8181"
	defAuthGRPCTimeout = "1s"

	envLogLevel        = "MF_WEBHOOKS_LOG_LEVEL"
	envDBHost          = "MF_WEBHOOKS_DB_HOST"
	envDBPort          = "MF_WEBHOOKS_DB_PORT"
	envDBUser          = "MF_WEBHOOKS_DB_USER"
	envDBPass          = "MF_WEBHOOKS_DB_PASS"
	envDB              = "MF_WEBHOOKS_DB"
	envDBSSLMode       = "MF_WEBHOOKS_DB_SSL_MODE"
	envDBSSLCert       = "MF_WEBHOOKS_DB_SSL_CERT"
	envDBSSLKey        = "MF_WEBHOOKS_DB_SSL_KEY"
	envDBSSLRootCert   = "MF_WEBHOOKS_DB_SSL_ROOT_CERT"
	envClientTLS       = "MF_WEBHOOKS_CLIENT_TLS"
	envCACerts         = "MF_WEBHOOKS_CA_CERTS"
	envAuthHTTPPort    = "MF_WEBHOOKS_AUTH_HTTP_PORT"
	envAuthGRPCPort    = "MF_WEBHOOKS_AUTH_GRPC_PORT"
	envHTTPPort        = "MF_WEBHOOKS_HTTP_PORT"
	envServerCert      = "MF_WEBHOOKS_SERVER_CERT"
	envServerKey       = "MF_WEBHOOKS_SERVER_KEY"
	envStandaloneEmail = "MF_WEBHOOKS_STANDALONE_EMAIL"
	envStandaloneToken = "MF_WEBHOOKS_STANDALONE_TOKEN"
	envJaegerURL       = "MF_JAEGER_URL"
	envAuthGRPCURL     = "MF_AUTH_GRPC_URL"
	envauthGRPCTimeout = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	logLevel        string
	dbConfig        postgres.Config
	clientTLS       bool
	caCerts         string
	httpPort        string
	authHTTPPort    string
	authGRPCPort    string
	serverCert      string
	serverKey       string
	standaloneEmail string
	standaloneToken string
	jaegerURL       string
	authGRPCURL     string
	authGRPCTimeout time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	webhooksTracer, webhooksCloser := initJaeger("webhooks", cfg.jaegerURL, logger)
	defer webhooksCloser.Close()

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	authTracer, authCloser := initJaeger("auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	auth, close := createAuthClient(cfg, authTracer, logger)
	if close != nil {
		defer close()
	}

	dbTracer, dbCloser := initJaeger("webhooks_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(auth, dbTracer, db, logger)

	g.Go(func() error {
		return startHTTPServer(ctx, "webhook-http", webhookshttpapi.MakeHandler(webhooksTracer, svc), cfg.httpPort, cfg, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Webhooks service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Webhooks service terminated: %s", err))
	}
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	authGRPCTimeout, err := time.ParseDuration(mainflux.Env(envauthGRPCTimeout, defAuthGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envauthGRPCTimeout, err.Error())
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
		httpPort:        mainflux.Env(envHTTPPort, defHTTPPort),
		authHTTPPort:    mainflux.Env(envAuthHTTPPort, defAuthHTTPPort),
		authGRPCPort:    mainflux.Env(envAuthGRPCPort, defAuthGRPCPort),
		serverCert:      mainflux.Env(envServerCert, defServerCert),
		serverKey:       mainflux.Env(envServerKey, defServerKey),
		standaloneEmail: mainflux.Env(envStandaloneEmail, defStandaloneEmail),
		standaloneToken: mainflux.Env(envStandaloneToken, defStandaloneToken),
		jaegerURL:       mainflux.Env(envJaegerURL, defJaegerURL),
		authGRPCURL:     mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		authGRPCTimeout: authGRPCTimeout,
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
	return authapi.NewClient(tracer, conn, cfg.authGRPCTimeout), conn.Close
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

func newService(ac mainflux.AuthServiceClient, dbTracer opentracing.Tracer, db *sqlx.DB, logger logger.Logger) webhooks.Service {
	database := postgres.NewDatabase(db)

	webhooksRepo := postgres.NewWebhookRepository(database)
	webhooksRepo = tracing.WebhookRepositoryMiddleware(dbTracer, webhooksRepo)
	idProvider := uuid.New()

	svc := webhooks.New(ac, webhooksRepo, idProvider)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "webhooks",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "webhooks",
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
		logger.Info(fmt.Sprintf("Webhooks %s service started using https on port %s with cert %s key %s",
			typ, port, cfg.serverCert, cfg.serverKey))
		go func() {
			errCh <- server.ListenAndServeTLS(cfg.serverCert, cfg.serverKey)
		}()
	default:
		logger.Info(fmt.Sprintf("Webhooks %s service started using http on port %s", typ, cfg.httpPort))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("Webhooks %s service error occurred during shutdown at %s: %s", typ, p, err))
			return fmt.Errorf("webhooks %s service occurred during shutdown at %s: %w", typ, p, err)
		}
		logger.Info(fmt.Sprintf("Webhooks %s service  shutdown of http at %s", typ, p))
		return nil
	case err := <-errCh:
		return err
	}
}