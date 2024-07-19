// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/readers/api"
	"github.com/MainfluxLabs/mainflux/readers/timescale"
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
	svcName      = "timescaledb-reader"
	stopWaitTime = 5 * time.Second

	defLogLevel          = "error"
	defPort              = "8911"
	defClientTLS         = "false"
	defCACerts           = ""
	defDBHost            = "localhost"
	defDBPort            = "5432"
	defDBUser            = "mainflux"
	defDBPass            = "mainflux"
	defDB                = "mainflux"
	defDBSSLMode         = "disable"
	defDBSSLCert         = ""
	defDBSSLKey          = ""
	defDBSSLRootCert     = ""
	defJaegerURL         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defAuthGRPCURL       = "localhost:8181"
	defAuthGRPCTimeout   = "1s"

	envLogLevel          = "MF_TIMESCALE_READER_LOG_LEVEL"
	envPort              = "MF_TIMESCALE_READER_PORT"
	envClientTLS         = "MF_TIMESCALE_READER_CLIENT_TLS"
	envCACerts           = "MF_TIMESCALE_READER_CA_CERTS"
	envDBHost            = "MF_TIMESCALE_READER_DB_HOST"
	envDBPort            = "MF_TIMESCALE_READER_DB_PORT"
	envDBUser            = "MF_TIMESCALE_READER_DB_USER"
	envDBPass            = "MF_TIMESCALE_READER_DB_PASS"
	envDB                = "MF_TIMESCALE_READER_DB"
	envDBSSLMode         = "MF_TIMESCALE_READER_DB_SSL_MODE"
	envDBSSLCert         = "MF_TIMESCALE_READER_DB_SSL_CERT"
	envDBSSLKey          = "MF_TIMESCALE_READER_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_TIMESCALE_READER_DB_SSL_ROOT_CERT"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envAuthGRPCURL       = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout   = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	logLevel          string
	clientTLS         bool
	caCerts           string
	dbConfig          timescale.Config
	httpConfig        servers.Config
	jaegerURL         string
	thingsGRPCURL     string
	authGRPCURL       string
	thingsGRPCTimeout time.Duration
	authGRPCTimeout   time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	conn := connectToThings(cfg, logger)
	defer conn.Close()

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	authTracer, authCloser := initJaeger("auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	authConn := connectToAuth(cfg, logger)
	defer authConn.Close()
	auth := authapi.NewClient(authTracer, authConn, cfg.authGRPCTimeout)

	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsGRPCTimeout)

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	repo := newService(db, logger)

	g.Go(func() error {
		return servers.StartHTTPServer(ctx, svcName, api.MakeHandler(repo, tc, auth, svcName, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Timescale reader service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Timescale reader service terminated: %s", err))
	}
}

func loadConfig() config {
	dbConfig := timescale.Config{
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
		Port:         mainflux.Env(envPort, defPort),
		StopWaitTime: stopWaitTime,
	}

	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	thingsGRPCTimeout, err := time.ParseDuration(mainflux.Env(envThingsGRPCTimeout, defThingsGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsGRPCTimeout, err.Error())
	}

	authGRPCTimeout, err := time.ParseDuration(mainflux.Env(envAuthGRPCTimeout, defAuthGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsGRPCTimeout, err.Error())
	}

	return config{
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		clientTLS:         tls,
		caCerts:           mainflux.Env(envCACerts, defCACerts),
		dbConfig:          dbConfig,
		httpConfig:        httpConfig,
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCURL:     mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
		authGRPCURL:       mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		authGRPCTimeout:   authGRPCTimeout,
	}
}

func connectToAuth(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	logger.Info("Connecting to auth via gRPC")
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
	logger.Info(fmt.Sprintf("Established gRPC connection to auth via gRPC: %s", cfg.authGRPCURL))
	return conn
}

func connectToDB(dbConfig timescale.Config, logger logger.Logger) *sqlx.DB {
	db, err := timescale.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Timescale: %s", err))
		os.Exit(1)
	}
	return db
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

func connectToThings(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.clientTLS {
		if cfg.caCerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.caCerts, "")
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to load certs: %s", err))
				os.Exit(1)
			}
			opts = append(opts, grpc.WithTransportCredentials(tpc))
		}
	} else {
		logger.Info("gRPC communication is not encrypted")
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(cfg.thingsGRPCURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to things service: %s", err))
		os.Exit(1)
	}
	return conn
}

func newService(db *sqlx.DB, logger logger.Logger) readers.MessageRepository {
	svc := timescale.New(db)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "timescale",
			Subsystem: "message_reader",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "timescale",
			Subsystem: "message_reader",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}
