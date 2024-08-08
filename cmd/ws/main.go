// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"golang.org/x/sync/errgroup"

	logger "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	adapter "github.com/MainfluxLabs/mainflux/ws"
	"github.com/MainfluxLabs/mainflux/ws/api"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	stopWaitTime = 5 * time.Second

	defPort              = "8190"
	defBrokerURL         = "nats://localhost:4222"
	defLogLevel          = "error"
	defClientTLS         = "false"
	defCACerts           = ""
	defJaegerURL         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"

	envPort              = "MF_WS_ADAPTER_PORT"
	envBrokerURL         = "MF_BROKER_URL"
	envLogLevel          = "MF_WS_ADAPTER_LOG_LEVEL"
	envClientTLS         = "MF_WS_ADAPTER_CLIENT_TLS"
	envCACerts           = "MF_WS_ADAPTER_CA_CERTS"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
)

type config struct {
	thingsConfig      clients.Config
	port              string
	brokerURL         string
	logLevel          string
	jaegerURL         string
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

	conn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer conn.Close()

	thingsTracer, thingsCloser := jaeger.Init("ws_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsGRPCTimeout)

	nps, err := brokers.NewPubSub(cfg.brokerURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer nps.Close()

	svc := newService(tc, nps, logger)

	g.Go(func() error {
		return startWSServer(ctx, cfg, svc, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("WS adapter service shutdown by signal: %s", sig))
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("WS adapter service terminated: %s", err))
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

	thingsConfig := clients.Config{
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envCACerts, defCACerts),
		URL:        mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		ClientName: clients.Things,
	}

	return config{
		thingsConfig:      thingsConfig,
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		port:              mainflux.Env(envPort, defPort),
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
	}
}

func newService(tc protomfx.ThingsServiceClient, nps messaging.PubSub, logger logger.Logger) adapter.Service {
	svc := adapter.New(tc, nps)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "ws_adapter",
			Subsystem: "api",
			Name:      "reqeust_count",
			Help:      "Number of requests received",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "ws_adapter",
			Subsystem: "api",
			Name:      "request_latency_microsecond",
			Help:      "Total duration of requests in microseconds",
		}, []string{"method"}),
	)

	return svc
}

func startWSServer(ctx context.Context, cfg config, svc adapter.Service, l logger.Logger) error {
	p := fmt.Sprintf(":%s", cfg.port)
	errCh := make(chan error, 2)
	server := &http.Server{Addr: p, Handler: api.MakeHandler(svc, l)}
	l.Info(fmt.Sprintf("WS adapter service started, exposed port %s", cfg.port))

	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			l.Error(fmt.Sprintf("WS adapter service error occurred during shutdown at %s: %s", p, err))
			return fmt.Errorf("WS adapter service error occurred during shutdown at %s: %w", p, err)
		}
		l.Info(fmt.Sprintf("WS adapter service shutdown at %s", p))
		return nil
	case err := <-errCh:
		return err
	}
}
