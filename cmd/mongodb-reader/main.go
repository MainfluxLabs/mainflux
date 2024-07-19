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
	"github.com/MainfluxLabs/mainflux/readers/mongodb"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	stopWaitTime = 5 * time.Second
	svcName      = "mongodb-reader"

	defLogLevel          = "error"
	defPort              = "8180"
	defDB                = "mainflux"
	defDBHost            = "localhost"
	defDBPort            = "27017"
	defClientTLS         = "false"
	defCACerts           = ""
	defServerCert        = ""
	defServerKey         = ""
	defJaegerURL         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defAuthGRPCURL       = "localhost:8181"
	defAuthGRPCTimeout   = "1s"

	envLogLevel          = "MF_MONGO_READER_LOG_LEVEL"
	envPort              = "MF_MONGO_READER_PORT"
	envDB                = "MF_MONGO_READER_DB"
	envDBHost            = "MF_MONGO_READER_DB_HOST"
	envDBPort            = "MF_MONGO_READER_DB_PORT"
	envClientTLS         = "MF_MONGO_READER_CLIENT_TLS"
	envCACerts           = "MF_MONGO_READER_CA_CERTS"
	envServerCert        = "MF_MONGO_READER_SERVER_CERT"
	envServerKey         = "MF_MONGO_READER_SERVER_KEY"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envAuthGRPCURL       = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout   = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	httpConfig        servers.Config
	logLevel          string
	dbName            string
	dbHost            string
	dbPort            string
	clientTLS         bool
	caCerts           string
	jaegerURL         string
	thingsGRPCURL     string
	authGRPCURL       string
	thingsGRPCTimeout time.Duration
	authGRPCTimeout   time.Duration
}

func main() {
	cfg := loadConfigs()
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

	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsGRPCTimeout)

	authTracer, authCloser := initJaeger("auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	authConn := connectToAuth(cfg, logger)
	defer authConn.Close()

	auth := authapi.NewClient(authTracer, authConn, cfg.authGRPCTimeout)

	db := connectToMongoDB(cfg.dbHost, cfg.dbPort, cfg.dbName, logger)

	repo := newService(db, logger)

	g.Go(func() error {
		return servers.StartHTTPServer(ctx, svcName, api.MakeHandler(repo, tc, auth, svcName, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("MongoDB reader service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("MongoDB reader service terminated: %s", err))
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

func loadConfigs() config {
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
		log.Fatalf("Invalid %s value: %s", envAuthGRPCTimeout, err.Error())
	}

	httpConfig := servers.Config{
		ServerCert:   mainflux.Env(envServerCert, defServerCert),
		ServerKey:    mainflux.Env(envServerKey, defServerKey),
		Port:         mainflux.Env(envPort, defPort),
		StopWaitTime: stopWaitTime,
	}

	return config{
		httpConfig:        httpConfig,
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		dbName:            mainflux.Env(envDB, defDB),
		dbHost:            mainflux.Env(envDBHost, defDBHost),
		dbPort:            mainflux.Env(envDBPort, defDBPort),
		clientTLS:         tls,
		caCerts:           mainflux.Env(envCACerts, defCACerts),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCURL:     mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		authGRPCURL:       mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
		authGRPCTimeout:   authGRPCTimeout,
	}
}

func connectToMongoDB(host, port, name string, logger logger.Logger) *mongo.Database {
	addr := fmt.Sprintf("mongodb://%s:%s", host, port)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to database: %s", err))
		os.Exit(1)
	}

	return client.Database(name)
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
	logger.Info(fmt.Sprintf("Established gRPC connection to things via gRPC: %s", cfg.thingsGRPCURL))
	return conn
}

func newService(db *mongo.Database, logger logger.Logger) readers.MessageRepository {
	repo := mongodb.New(db)
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(
		repo,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "mongodb",
			Subsystem: "message_reader",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "mongodb",
			Subsystem: "message_reader",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return repo
}
