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
	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/consumers/notifiers/api"
	httpapi "github.com/MainfluxLabs/mainflux/consumers/notifiers/api/http"
	"github.com/MainfluxLabs/mainflux/consumers/notifiers/smtp"
	"github.com/MainfluxLabs/mainflux/consumers/notifiers/smtp/postgres"
	"github.com/MainfluxLabs/mainflux/internal/email"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/ulid"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	svcName              = "smtp-notifier"
	stopWaitTime         = 5 * time.Second
	defLogLevel          = "error"
	defFrom              = ""
	defJaegerURL         = ""
	defBrokerURL         = "nats://localhost:4222"
	defDBHost            = "localhost"
	defDBPort            = "5432"
	defDBUser            = "mainflux"
	defDBPass            = "mainflux"
	defDB                = "smtp-notifiers"
	defDBSSLMode         = "disable"
	defDBSSLCert         = ""
	defDBSSLKey          = ""
	defDBSSLRootCert     = ""
	defThingsTLS         = "false"
	defThingsCACerts     = ""
	defHTTPPort          = "8906"
	defServerCert        = ""
	defServerKey         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"

	defEmailHost        = "localhost"
	defEmailPort        = "25"
	defEmailUsername    = "root"
	defEmailPassword    = ""
	defEmailFromAddress = ""
	defEmailFromName    = ""
	defEmailTemplate    = "email.tmpl"

	defAuthTLS         = "false"
	defAuthCACerts     = ""
	defAuthGRPCURL     = "localhost:8181"
	defAuthGRPCTimeout = "1s"

	envLogLevel          = "MF_SMTP_NOTIFIER_LOG_LEVEL"
	envFrom              = "MF_SMTP_NOTIFIER_FROM_ADDR"
	envJaegerURL         = "MF_JAEGER_URL"
	envBrokerURL         = "MF_BROKER_URL"
	envDBHost            = "MF_SMTP_NOTIFIER_DB_HOST"
	envDBPort            = "MF_SMTP_NOTIFIER_DB_PORT"
	envDBUser            = "MF_SMTP_NOTIFIER_DB_USER"
	envDBPass            = "MF_SMTP_NOTIFIER_DB_PASS"
	envDB                = "MF_SMTP_NOTIFIER_DB"
	envDBSSLMode         = "MF_SMTP_NOTIFIER_DB_SSL_MODE"
	envDBSSLCert         = "MF_SMTP_NOTIFIER_DB_SSL_CERT"
	envDBSSLKey          = "MF_SMTP_NOTIFIER_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_SMTP_NOTIFIER_DB_SSL_ROOT_CERT"
	envThingsTLS         = "MF_SMTP_NOTIFIER_THINGS_TLS"
	envThingsCACerts     = "MF_SMTP_NOTIFIER_THINGS_CA_CERTS"
	envHTTPPort          = "MF_SMTP_NOTIFIER_HTTP_PORT"
	envServerCert        = "MF_SMTP_NOTIFIER_SERVER_CERT"
	envServerKey         = "MF_SMTP_NOTIFIER_SERVER_KEY"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"

	envEmailHost        = "MF_EMAIL_HOST"
	envEmailPort        = "MF_EMAIL_PORT"
	envEmailUsername    = "MF_EMAIL_USERNAME"
	envEmailPassword    = "MF_EMAIL_PASSWORD"
	envEmailFromAddress = "MF_EMAIL_FROM_ADDRESS"
	envEmailFromName    = "MF_EMAIL_FROM_NAME"
	envEmailTemplate    = "MF_SMTP_NOTIFIER_TEMPLATE"

	envAuthTLS         = "MF_AUTH_CLIENT_TLS"
	envAuthCACerts     = "MF_AUTH_CA_CERTS"
	envAuthGRPCURL     = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	brokerURL         string
	logLevel          string
	dbConfig          postgres.Config
	thingsTLS         bool
	thingsCACerts     string
	httpPort          string
	emailConf         email.Config
	from              string
	serverCert        string
	serverKey         string
	jaegerURL         string
	authTLS           bool
	authCACerts       string
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

	notifiersTracer, notifiersCloser := initJaeger("smtp-notifier", cfg.jaegerURL, logger)
	defer notifiersCloser.Close()

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	authTracer, closer := initJaeger("auth", cfg.jaegerURL, logger)
	defer closer.Close()

	auth, close := connectToAuth(cfg, authTracer, logger)
	if close != nil {
		defer close()
	}

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	things, close := createThingsClient(cfg, thingsTracer, logger)
	if close != nil {
		defer close()
	}

	svc := newService(auth, cfg, logger, db, things)

	if err = consumers.Start(svcName, pubSub, svc, brokers.SubjectSmtp); err != nil {
		logger.Error(fmt.Sprintf("Failed to create SMTP notifier: %s", err))
	}

	g.Go(func() error {
		return startHTTPServer(ctx, "smtp-notifier-http", httpapi.MakeHandler(notifiersTracer, svc, logger), cfg.httpPort, cfg, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("SMTP notifier service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("SMTP notifier service terminated: %s", err))
	}

}

func loadConfig() config {
	authGRPCTimeout, err := time.ParseDuration(mainflux.Env(envAuthGRPCTimeout, defAuthGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthGRPCTimeout, err.Error())
	}

	thingsGRPCTimeout, err := time.ParseDuration(mainflux.Env(envThingsGRPCTimeout, defThingsGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthGRPCTimeout, err.Error())
	}

	authTLS, err := strconv.ParseBool(mainflux.Env(envAuthTLS, defAuthTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envAuthTLS)
	}

	thingsTLS, err := strconv.ParseBool(mainflux.Env(envThingsTLS, defThingsTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envThingsTLS)
	}

	emailConf := email.Config{
		FromAddress: mainflux.Env(envEmailFromAddress, defEmailFromAddress),
		FromName:    mainflux.Env(envEmailFromName, defEmailFromName),
		Host:        mainflux.Env(envEmailHost, defEmailHost),
		Port:        mainflux.Env(envEmailPort, defEmailPort),
		Username:    mainflux.Env(envEmailUsername, defEmailUsername),
		Password:    mainflux.Env(envEmailPassword, defEmailPassword),
		Template:    mainflux.Env(envEmailTemplate, defEmailTemplate),
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
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		emailConf:         emailConf,
		dbConfig:          dbConfig,
		from:              mainflux.Env(envFrom, defFrom),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		authTLS:           authTLS,
		authCACerts:       mainflux.Env(envAuthCACerts, defAuthCACerts),
		authGRPCURL:       mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		thingsTLS:         thingsTLS,
		thingsCACerts:     mainflux.Env(envThingsCACerts, defThingsCACerts),
		httpPort:          mainflux.Env(envHTTPPort, defHTTPPort),
		serverCert:        mainflux.Env(envServerCert, defServerCert),
		serverKey:         mainflux.Env(envServerKey, defServerKey),
		authGRPCTimeout:   authGRPCTimeout,
		thingsGRPCURL:     mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
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
		logger.Error(fmt.Sprintf("Failed to init Jaeger: %s", err))
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

func connectToAuth(cfg config, tracer opentracing.Tracer, logger logger.Logger) (mainflux.AuthServiceClient, func() error) {
	var opts []grpc.DialOption
	if cfg.authTLS {
		if cfg.authCACerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.authCACerts, "")
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

	return authapi.NewClient(tracer, conn, cfg.authGRPCTimeout), conn.Close
}

func createThingsClient(cfg config, tracer opentracing.Tracer, logger logger.Logger) (mainflux.ThingsServiceClient, func() error) {
	conn := connectToThings(cfg, logger)
	return thingsapi.NewClient(conn, tracer, cfg.thingsGRPCTimeout), conn.Close
}

func connectToThings(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.thingsTLS {
		if cfg.thingsCACerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.thingsCACerts, "")
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

func newService(ac mainflux.AuthServiceClient, c config, logger logger.Logger, db *sqlx.DB, tc mainflux.ThingsServiceClient) notifiers.Service {
	idp := ulid.New()
	database := postgres.NewDatabase(db)

	agent, err := email.New(&c.emailConf)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create email agent: %s", err))
		os.Exit(1)
	}

	notifier := smtp.New(agent)
	notifierRepo := postgres.NewSmtpNotifierRepository(database)
	svc := notifiers.New(ac, idp, notifier, c.from, notifierRepo, tc)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "notifier",
			Subsystem: "smtp",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "notifier",
			Subsystem: "smtp",
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
		logger.Info(fmt.Sprintf("SMTP notifiers %s service started using https on port %s with cert %s key %s",
			name, port, cfg.serverCert, cfg.serverKey))
		go func() {
			errCh <- server.ListenAndServeTLS(cfg.serverCert, cfg.serverKey)
		}()
	default:
		logger.Info(fmt.Sprintf("SMTP notifiers %s service started using http on port %s", name, cfg.httpPort))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("SMTP notifiers %s service error occurred during shutdown at %s: %s", name, p, err))
			return fmt.Errorf("SMTP notifiers %s service occurred during shutdown at %s: %w", name, p, err)
		}
		logger.Info(fmt.Sprintf("SMTP notifiers %s service  shutdown of http at %s", name, p))
		return nil
	case err := <-errCh:
		return err
	}
}
