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
	"github.com/MainfluxLabs/mainflux/consumers/notifiers/postgres"
	mfsmpp "github.com/MainfluxLabs/mainflux/consumers/notifiers/smpp"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
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
	svcName      = "smpp-notifier"
	stopWaitTime = 5 * time.Second

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
	defAuthTLS           = "false"
	defAuthCACerts       = ""
	defThingsTLS         = "false"
	defThingsCACerts     = ""
	defHTTPPort          = "8907"
	defServerCert        = ""
	defServerKey         = ""
	defAuthGRPCURL       = "localhost:8181"
	defAuthGRPCTimeout   = "1s"
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"

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
	envAuthTLS           = "MF_SMPP_NOTIFIER_AUTH_TLS"
	envAuthCACerts       = "MF_SMPP_NOTIFIER_AUTH_CA_CERTS"
	envThingsTLS         = "MF_SMPP_NOTIFIER_THINGS_TLS"
	envThingsCACerts     = "MF_SMPP_NOTIFIER_THINGS_CA_CERTS"
	envHTTPPort          = "MF_SMPP_NOTIFIER_PORT"
	envServerCert        = "MF_SMPP_NOTIFIER_SERVER_CERT"
	envServerKey         = "MF_SMPP_NOTIFIER_SERVER_KEY"
	envAuthGRPCURL       = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout   = "MF_AUTH_GRPC_TIMEOUT"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"

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
	authTLS           bool
	authCACerts       string
	thingsTLS         bool
	thingsCACerts     string
	httpPort          string
	smppConf          mfsmpp.Config
	from              string
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

	notifiersTracer, notifiersCloser := initJaeger("smpp-notifier", cfg.jaegerURL, logger)
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

	if err = consumers.Start(svcName, pubSub, svc, brokers.SubjectSmpp); err != nil {
		logger.Error(fmt.Sprintf("Failed to create SMPP notifier: %s", err))
	}

	g.Go(func() error {
		return startHTTPServer(ctx, "smpp-notifier-http", httpapi.MakeHandler(notifiersTracer, svc, logger), cfg.httpPort, cfg, logger)
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

	return config{
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		smppConf:          smppConf,
		dbConfig:          dbConfig,
		from:              mainflux.Env(envFrom, defFrom),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		authTLS:           authTLS,
		authCACerts:       mainflux.Env(envAuthCACerts, defAuthCACerts),
		thingsTLS:         thingsTLS,
		thingsCACerts:     mainflux.Env(envThingsCACerts, defThingsCACerts),
		httpPort:          mainflux.Env(envHTTPPort, defHTTPPort),
		serverCert:        mainflux.Env(envServerCert, defServerCert),
		serverKey:         mainflux.Env(envServerKey, defServerKey),
		authGRPCURL:       mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
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
	idp := uuid.New()
	database := postgres.NewDatabase(db)

	notifier := mfsmpp.New(c.smppConf)
	notifierRepo := postgres.NewNotifierRepository(database)
	svc := notifiers.New(ac, idp, notifier, c.from, notifierRepo, tc)
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

func startHTTPServer(ctx context.Context, name string, handler http.Handler, port string, cfg config, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", port)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: handler}

	switch {
	case cfg.serverCert != "" || cfg.serverKey != "":
		logger.Info(fmt.Sprintf("SMPP notifiers %s service started using https on port %s with cert %s key %s",
			name, port, cfg.serverCert, cfg.serverKey))
		go func() {
			errCh <- server.ListenAndServeTLS(cfg.serverCert, cfg.serverKey)
		}()
	default:
		logger.Info(fmt.Sprintf("SMPP notifiers %s service started using http on port %s", name, cfg.httpPort))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("SMPP notifiers %s service error occurred during shutdown at %s: %s", name, p, err))
			return fmt.Errorf("SMPP notifiers %s service occurred during shutdown at %s: %w", name, p, err)
		}
		logger.Info(fmt.Sprintf("SMPP notifiers %s service  shutdown of http at %s", name, p))
		return nil
	case err := <-errCh:
		return err
	}
}
