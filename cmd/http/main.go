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
	adapter "github.com/MainfluxLabs/mainflux/http"
	"github.com/MainfluxLabs/mainflux/http/api"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	rulesapi "github.com/MainfluxLabs/mainflux/rules/api/grpc"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	stopWaitTime = 5 * time.Second
	svcName      = "http-adapter"

	defLogLevel          = "error"
	defClientTLS         = "false"
	defCACerts           = ""
	defPort              = "8180"
	defJaegerURL         = ""
	defBrokerURL         = "nats://localhost:4222"
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defRulesGRPCURL      = "localhost:8186"
	defRulesGRPCTimeout  = "1s"

	envLogLevel          = "MF_HTTP_ADAPTER_LOG_LEVEL"
	envClientTLS         = "MF_HTTP_ADAPTER_CLIENT_TLS"
	envCACerts           = "MF_HTTP_ADAPTER_CA_CERTS"
	envPort              = "MF_HTTP_ADAPTER_PORT"
	envJaegerURL         = "MF_JAEGER_URL"
	envBrokerURL         = "MF_BROKER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envRulesGRPCURL      = "MF_RULES_GRPC_URL"
	envRulesGRPCTimeout  = "MF_RULES_GRPC_TIMEOUT"
)

type config struct {
	httpConfig        servers.Config
	thingsConfig      clients.Config
	rulesConfig       clients.Config
	logLevel          string
	brokerURL         string
	jaegerURL         string
	thingsGRPCTimeout time.Duration
	rulesGRPCTimeout  time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	tConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer tConn.Close()

	httpTracer, closer := jaeger.Init("http_adapter", cfg.jaegerURL, logger)
	defer closer.Close()

	thingsTracer, thingsCloser := jaeger.Init("http_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	rConn := clientsgrpc.Connect(cfg.rulesConfig, logger)
	defer rConn.Close()

	rulesTracer, rulesCloser := jaeger.Init("http_rules", cfg.jaegerURL, logger)
	defer rulesCloser.Close()

	pub, err := brokers.NewPublisher(cfg.brokerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pub.Close()

	tc := thingsapi.NewClient(tConn, thingsTracer, cfg.thingsGRPCTimeout)
	rc := rulesapi.NewClient(rConn, rulesTracer, cfg.rulesGRPCTimeout)
	svc := adapter.New(pub, tc, rc, logger)

	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "http_adapter",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "http_adapter",
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

	rulesGRPCTimeout, err := time.ParseDuration(mainflux.Env(envRulesGRPCTimeout, defRulesGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envRulesGRPCTimeout, err.Error())
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

	rulesConfig := clients.Config{
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envCACerts, defCACerts),
		URL:        mainflux.Env(envRulesGRPCURL, defRulesGRPCURL),
		ClientName: clients.Rules,
	}

	return config{
		httpConfig:        httpConfig,
		thingsConfig:      thingsConfig,
		rulesConfig:       rulesConfig,
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
		rulesGRPCTimeout:  rulesGRPCTimeout,
	}
}
