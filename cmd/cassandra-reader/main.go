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
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux"
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/readers/api"
	"github.com/MainfluxLabs/mainflux/readers/cassandra"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/auth/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gocql/gocql"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	stopWaitTime = 5 * time.Second

	sep              = ","
	defLogLevel      = "error"
	defPort          = "8180"
	defCluster       = "127.0.0.1"
	defKeyspace      = "mainflux"
	defDBUser        = "mainflux"
	defDBPass        = "mainflux"
	defDBPort        = "9042"
	defClientTLS     = "false"
	defCACerts       = ""
	defServerCert    = ""
	defServerKey     = ""
	defJaegerURL     = ""
	defThingsGRPCURL = "localhost:8183"
	defThingsTimeout = "1s"
	defAuthGRPCURL   = "localhost:8181"
	defAuthTimeout   = "1s"

	envLogLevel      = "MF_CASSANDRA_READER_LOG_LEVEL"
	envPort          = "MF_CASSANDRA_READER_PORT"
	envCluster       = "MF_CASSANDRA_READER_DB_CLUSTER"
	envKeyspace      = "MF_CASSANDRA_READER_DB_KEYSPACE"
	envDBUser        = "MF_CASSANDRA_READER_DB_USER"
	envDBPass        = "MF_CASSANDRA_READER_DB_PASS"
	envDBPort        = "MF_CASSANDRA_READER_DB_PORT"
	envClientTLS     = "MF_CASSANDRA_READER_CLIENT_TLS"
	envCACerts       = "MF_CASSANDRA_READER_CA_CERTS"
	envServerCert    = "MF_CASSANDRA_READER_SERVER_CERT"
	envServerKey     = "MF_CASSANDRA_READER_SERVER_KEY"
	envJaegerURL     = "MF_JAEGER_URL"
	envThingsGRPCURL = "MF_THINGS_AUTH_GRPC_URL"
	envThingsTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envAuthGRPCURL   = "MF_AUTH_GRPC_URL"
	envAuthTimeout   = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	logLevel      string
	port          string
	dbCfg         cassandra.DBConfig
	clientTLS     bool
	caCerts       string
	serverCert    string
	serverKey     string
	jaegerURL     string
	thingsGRPCURL string
	authGRPCURL   string
	thingsTimeout time.Duration
	authTimeout   time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	session := connectToCassandra(cfg.dbCfg, logger)
	defer session.Close()

	conn := connectToThings(cfg, logger)
	defer conn.Close()

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsTimeout)
	authTracer, authCloser := initJaeger("auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	authConn := connectToAuth(cfg, logger)
	defer authConn.Close()

	auth := authapi.NewClient(authTracer, authConn, cfg.authTimeout)

	repo := newService(session, logger)

	g.Go(func() error {
		return startHTTPServer(ctx, repo, tc, auth, cfg, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Cassandra reader service shutdown by signal: %s", sig))
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Cassandra reader service terminated: %s", err))
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
	logger.Info(fmt.Sprintf("Established gRPC connection to things via gRPC: %s", cfg.authGRPCURL))
	return conn
}

func loadConfig() config {
	dbPort, err := strconv.Atoi(mainflux.Env(envDBPort, defDBPort))
	if err != nil {
		log.Fatal(err)
	}

	dbCfg := cassandra.DBConfig{
		Hosts:    strings.Split(mainflux.Env(envCluster, defCluster), sep),
		Keyspace: mainflux.Env(envKeyspace, defKeyspace),
		User:     mainflux.Env(envDBUser, defDBUser),
		Pass:     mainflux.Env(envDBPass, defDBPass),
		Port:     dbPort,
	}

	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	thingsTimeout, err := time.ParseDuration(mainflux.Env(envThingsTimeout, defThingsTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsTimeout, err.Error())
	}

	authTimeout, err := time.ParseDuration(mainflux.Env(envAuthTimeout, defAuthTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthTimeout, err.Error())
	}

	return config{
		logLevel:      mainflux.Env(envLogLevel, defLogLevel),
		port:          mainflux.Env(envPort, defPort),
		dbCfg:         dbCfg,
		clientTLS:     tls,
		caCerts:       mainflux.Env(envCACerts, defCACerts),
		serverCert:    mainflux.Env(envServerCert, defServerCert),
		serverKey:     mainflux.Env(envServerKey, defServerKey),
		jaegerURL:     mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCURL: mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		authGRPCURL:   mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		authTimeout:   authTimeout,
		thingsTimeout: thingsTimeout,
	}
}

func connectToCassandra(dbCfg cassandra.DBConfig, logger logger.Logger) *gocql.Session {
	session, err := cassandra.Connect(dbCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Cassandra cluster: %s", err))
		os.Exit(1)
	}

	return session
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

func newService(session *gocql.Session, logger logger.Logger) readers.MessageRepository {
	repo := cassandra.New(session)
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(
		repo,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "cassandra",
			Subsystem: "message_reader",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "cassandra",
			Subsystem: "message_reader",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return repo
}

func startHTTPServer(ctx context.Context, repo readers.MessageRepository, tc mainflux.ThingsServiceClient, ac mainflux.AuthServiceClient, cfg config, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", cfg.port)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: api.MakeHandler(repo, tc, ac, "cassandra-reader", logger)}
	switch {
	case cfg.serverCert != "" || cfg.serverKey != "":
		logger.Info(fmt.Sprintf("Cassandra reader service started using https on port %s with cert %s key %s", cfg.port, cfg.serverCert, cfg.serverKey))
		go func() {
			errCh <- server.ListenAndServeTLS(cfg.serverCert, cfg.serverKey)
		}()
	default:
		logger.Info(fmt.Sprintf("Cassandra reader service started, exposed port %s", cfg.port))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}
	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("Cassandra reader service error occurred during shutdown at %s: %s", p, err))
			return fmt.Errorf("cassandra reader service error occurred during shutdown at %s: %w", p, err)
		}
		logger.Info(fmt.Sprintf("Cassandra reader service shutdown of http at %s", p))
		return nil
	case err := <-errCh:
		return err
	}
}
