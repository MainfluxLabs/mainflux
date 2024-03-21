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
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/auth/grpc"
	"github.com/MainfluxLabs/mainflux/webhooks"
	"github.com/MainfluxLabs/mainflux/webhooks/api"
	httpapi "github.com/MainfluxLabs/mainflux/webhooks/api/http"
	"github.com/MainfluxLabs/mainflux/webhooks/postgres"
	"github.com/MainfluxLabs/mainflux/webhooks/tracing"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	subject              = "webhook"
	svcName              = "webhooks"
	stopWaitTime         = 5 * time.Second
	defBrokerURL         = "nats://localhost:4222"
	defLogLevel          = "error"
	defDBHost            = "localhost"
	defDBPort            = "5432"
	defDBUser            = "mainflux"
	defDBPass            = "mainflux"
	defDB                = "webhooks"
	defDBSSLMode         = "disable"
	defDBSSLCert         = ""
	defDBSSLKey          = ""
	defDBSSLRootCert     = ""
	defClientTLS         = "false"
	defCACerts           = ""
	defHTTPPort          = "9021"
	defAuthGRPCPort      = "8181"
	defJaegerURL         = ""
	defServerCert        = ""
	defServerKey         = ""
	defAuthGRPCURL       = "localhost:8181"
	defAuthGRPCTimeout   = "1s"
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"

	envBrokerURL         = "MF_BROKER_URL"
	envLogLevel          = "MF_WEBHOOKS_LOG_LEVEL"
	envDBHost            = "MF_WEBHOOKS_DB_HOST"
	envDBPort            = "MF_WEBHOOKS_DB_PORT"
	envDBUser            = "MF_WEBHOOKS_DB_USER"
	envDBPass            = "MF_WEBHOOKS_DB_PASS"
	envDB                = "MF_WEBHOOKS_DB"
	envDBSSLMode         = "MF_WEBHOOKS_DB_SSL_MODE"
	envDBSSLCert         = "MF_WEBHOOKS_DB_SSL_CERT"
	envDBSSLKey          = "MF_WEBHOOKS_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_WEBHOOKS_DB_SSL_ROOT_CERT"
	envClientTLS         = "MF_WEBHOOKS_CLIENT_TLS"
	envCACerts           = "MF_WEBHOOKS_CA_CERTS"
	envAuthGRPCPort      = "MF_WEBHOOKS_AUTH_GRPC_PORT"
	envHTTPPort          = "MF_WEBHOOKS_HTTP_PORT"
	envServerCert        = "MF_WEBHOOKS_SERVER_CERT"
	envServerKey         = "MF_WEBHOOKS_SERVER_KEY"
	envJaegerURL         = "MF_JAEGER_URL"
	envAuthGRPCURL       = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout   = "MF_AUTH_GRPC_TIMEOUT"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
)

type config struct {
	brokerURL         string
	logLevel          string
	dbConfig          postgres.Config
	clientTLS         bool
	caCerts           string
	httpPort          string
	authGRPCPort      string
	serverCert        string
	serverKey         string
	jaegerURL         string
	authGRPCURL       string
	authGRPCTimeout   time.Duration
	thingsGRPCURL     string
	thingsGRPCTimeout time.Duration
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

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	things, close := createThingsClient(cfg, thingsTracer, logger)
	if close != nil {
		defer close()
	}

	dbTracer, dbCloser := initJaeger("webhooks_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(auth, things, dbTracer, db, logger)

	if err = webhooks.Start(ctx, svcName, subject, svc, pubSub); err != nil {
		logger.Error(fmt.Sprintf("Failed to create SMTP notifier: %s", err))
	}

	g.Go(func() error {
		return startHTTPServer(ctx, "webhook-http", httpapi.MakeHandler(webhooksTracer, svc, logger), cfg.httpPort, cfg, logger)
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

	authGRPCTimeout, err := time.ParseDuration(mainflux.Env(envAuthGRPCTimeout, defAuthGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthGRPCTimeout, err.Error())
	}

	thingsAuthGRPCTimeout, err := time.ParseDuration(mainflux.Env(envThingsGRPCTimeout, defThingsGRPCTimeout))
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
	return config{
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:          dbConfig,
		clientTLS:         tls,
		caCerts:           mainflux.Env(envCACerts, defCACerts),
		httpPort:          mainflux.Env(envHTTPPort, defHTTPPort),
		authGRPCPort:      mainflux.Env(envAuthGRPCPort, defAuthGRPCPort),
		serverCert:        mainflux.Env(envServerCert, defServerCert),
		serverKey:         mainflux.Env(envServerKey, defServerKey),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		authGRPCURL:       mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		authGRPCTimeout:   authGRPCTimeout,
		thingsGRPCURL:     mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		thingsGRPCTimeout: thingsAuthGRPCTimeout,
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

func createThingsClient(cfg config, tracer opentracing.Tracer, logger logger.Logger) (mainflux.ThingsServiceClient, func() error) {
	conn := connectToThings(cfg, logger)
	return thingsapi.NewClient(conn, tracer, cfg.thingsGRPCTimeout), conn.Close
}

func connectToThings(cfg config, logger logger.Logger) *grpc.ClientConn {
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

	conn, err := grpc.Dial(cfg.thingsGRPCURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to things service: %s", err))
		os.Exit(1)
	}

	return conn
}

func newService(ac mainflux.AuthServiceClient, ts mainflux.ThingsServiceClient, dbTracer opentracing.Tracer, db *sqlx.DB, logger logger.Logger) webhooks.Service {
	database := postgres.NewDatabase(db)
	webhooksRepo := postgres.NewWebhookRepository(database)
	webhooksRepo = tracing.WebhookRepositoryMiddleware(dbTracer, webhooksRepo)

	svc := webhooks.New(ac, ts, webhooksRepo)
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

func startHTTPServer(ctx context.Context, name string, handler http.Handler, port string, cfg config, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", port)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: handler}

	switch {
	case cfg.serverCert != "" || cfg.serverKey != "":
		logger.Info(fmt.Sprintf("Webhooks %s service started using https on port %s with cert %s key %s",
			name, port, cfg.serverCert, cfg.serverKey))
		go func() {
			errCh <- server.ListenAndServeTLS(cfg.serverCert, cfg.serverKey)
		}()
	default:
		logger.Info(fmt.Sprintf("Webhooks %s service started using http on port %s", name, cfg.httpPort))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("Webhooks %s service error occurred during shutdown at %s: %s", name, p, err))
			return fmt.Errorf("webhooks %s service occurred during shutdown at %s: %w", name, p, err)
		}
		logger.Info(fmt.Sprintf("Webhooks %s service  shutdown of http at %s", name, p))
		return nil
	case err := <-errCh:
		return err
	}
}
