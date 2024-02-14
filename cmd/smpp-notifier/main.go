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
	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/consumers/notifiers/api"
	mfsmpp "github.com/MainfluxLabs/mainflux/consumers/notifiers/smpp"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/ulid"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
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
	defLogLevel  = "error"
	defFrom      = ""
	defJaegerURL = ""
	defBrokerURL = "nats://localhost:4222"

	defAddress    = ""
	defUsername   = ""
	defPassword   = ""
	defSystemType = ""
	defSrcAddrTON = "0"
	defDstAddrTON = "0"
	defSrcAddrNPI = "0"
	defDstAddrNPI = "0"

	defAuthTLS         = "false"
	defAuthCACerts     = ""
	defAuthGRPCURL     = "localhost:8181"
	defAuthGRPCTimeout = "1s"

	envLogLevel  = "MF_SMPP_NOTIFIER_LOG_LEVEL"
	envFrom      = "MF_SMPP_NOTIFIER_SOURCE_ADDR"
	envJaegerURL = "MF_JAEGER_URL"
	envBrokerURL = "MF_BROKER_URL"

	envAddress    = "MF_SMPP_ADDRESS"
	envUsername   = "MF_SMPP_USERNAME"
	envPassword   = "MF_SMPP_PASSWORD"
	envSystemType = "MF_SMPP_SYSTEM_TYPE"
	envSrcAddrTON = "MF_SMPP_SRC_ADDR_TON"
	envDstAddrTON = "MF_SMPP_DST_ADDR_TON"
	envSrcAddrNPI = "MF_SMPP_SRC_ADDR_NPI"
	envDstAddrNPI = "MF_SMPP_DST_ADDR_NPI"

	envAuthTLS         = "MF_AUTH_CLIENT_TLS"
	envAuthCACerts     = "MF_AUTH_CA_CERTS"
	envAuthGRPCURL     = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	brokerURL       string
	logLevel        string
	smppConf        mfsmpp.Config
	from            string
	jaegerURL       string
	authTLS         bool
	authCACerts     string
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

	pubSub, err := brokers.NewPubSub(cfg.brokerURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	authTracer, closer := initJaeger("auth", cfg.jaegerURL, logger)
	defer closer.Close()

	auth, close := connectToAuth(cfg, authTracer, logger)
	if close != nil {
		defer close()
	}

	svc := newService(auth, cfg, logger)

	if err = consumers.Start(svcName, pubSub, svc, brokers.SubjectSmpp); err != nil {
		logger.Error(fmt.Sprintf("Failed to create Postgres writer: %s", err))
	}

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

	tls, err := strconv.ParseBool(mainflux.Env(envAuthTLS, defAuthTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envAuthTLS)
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

	return config{
		logLevel:        mainflux.Env(envLogLevel, defLogLevel),
		brokerURL:       mainflux.Env(envBrokerURL, defBrokerURL),
		smppConf:        smppConf,
		from:            mainflux.Env(envFrom, defFrom),
		jaegerURL:       mainflux.Env(envJaegerURL, defJaegerURL),
		authTLS:         tls,
		authCACerts:     mainflux.Env(envAuthCACerts, defAuthCACerts),
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
		logger.Error(fmt.Sprintf("Failed to init Jaeger: %s", err))
		os.Exit(1)
	}

	return tracer, closer
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

func newService(ac mainflux.AuthServiceClient, c config, logger logger.Logger) notifiers.Service {
	idp := ulid.New()
	notifier := mfsmpp.New(c.smppConf)
	svc := notifiers.New(ac, idp, notifier, c.from)
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
