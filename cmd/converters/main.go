// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	adapter "github.com/MainfluxLabs/mainflux/converters"
	"github.com/MainfluxLabs/mainflux/converters/api"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	stopWaitTime = 5 * time.Second
	svcName      = "converters"

	defLogLevel          = "error"
	defClientTLS         = "false"
	defCACerts           = ""
	defPort              = "8180"
	defJaegerURL         = ""
	defBrokerURL         = "nats://localhost:4222"
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"

	envLogLevel          = "MF_CONVERTERS_LOG_LEVEL"
	envClientTLS         = "MF_CONVERTERS_CLIENT_TLS"
	envCACerts           = "MF_CONVERTERS_CA_CERTS"
	envPort              = "MF_CONVERTERS_PORT"
	envJaegerURL         = "MF_JAEGER_URL"
	envBrokerURL         = "MF_BROKER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
)

type config struct {
	logLevel          string
	jaegerURL         string
	brokerURL         string
	httpConfig        servers.Config
	thingsConfig      clients.Config
	thingsGRPCTimeout time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err)
	}

	// Create temp dir to support uploading large files
	tempDir := os.TempDir()
	if err := os.MkdirAll(tempDir, 1777); err != nil {
		log.Fatalf("Failed to create temporary directory %s: %s", tempDir, err)
	}

	conn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer conn.Close()

	httpTracer, closer := jaeger.Init(svcName, cfg.jaegerURL, logger)
	defer closer.Close()

	thingsTracer, thingsCloser := jaeger.Init(svcName+"_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	pub, err := brokers.NewPublisher(cfg.brokerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pub.Close()

	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsGRPCTimeout)
	svc := adapter.New(pub, tc)

	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: svcName,
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: svcName,
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	g.Go(func() error {
		return servershttp.Start(ctx, api.MakeHandler(svc, httpTracer, logger), cfg.httpConfig, logger)
	})
	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("HTTP adapter service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("HTTP adapter service terminated: %s", err))
	}

}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	thingsGRPCTimeout, err := time.ParseDuration(mainflux.Env(envThingsGRPCTimeout, defThingsGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsGRPCTimeout, err.Error())
	}

	httpConfig := servers.Config{
		ServerName:   svcName,
		Port:         mainflux.Env(envPort, defPort),
		StopWaitTime: stopWaitTime,
	}

	thingsConfig := clients.Config{
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envCACerts, defCACerts),
		URL:        mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		ClientName: clients.Things,
	}

	return config{
		httpConfig:        httpConfig,
		thingsConfig:      thingsConfig,
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
	}
}
