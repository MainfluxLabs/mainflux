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
	"github.com/MainfluxLabs/mainflux/coap"
	"github.com/MainfluxLabs/mainflux/coap/api"
	logger "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	opentracing "github.com/opentracing/opentracing-go"
	gocoap "github.com/plgd-dev/go-coap/v2"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"golang.org/x/sync/errgroup"
)

const (
	stopWaitTime = 5 * time.Second
	svcCoap      = "coap-adapter"
	svcThings    = "things"

	defPort              = "5683"
	defBrokerURL         = "nats://localhost:4222"
	defLogLevel          = "error"
	defClientTLS         = "false"
	defCACerts           = ""
	defJaegerURL         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"

	envPort              = "MF_COAP_ADAPTER_PORT"
	envBrokerURL         = "MF_BROKER_URL"
	envLogLevel          = "MF_COAP_ADAPTER_LOG_LEVEL"
	envClientTLS         = "MF_COAP_ADAPTER_CLIENT_TLS"
	envCACerts           = "MF_COAP_ADAPTER_CA_CERTS"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
)

type config struct {
	coapConfig        servers.Config
	thingsConfig      clients.Config
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

	conn := clients.Connect(cfg.thingsConfig, svcThings, logger)
	defer conn.Close()

	thingsTracer, thingsCloser := initJaeger(svcThings, cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsGRPCTimeout)

	nps, err := brokers.NewPubSub(cfg.brokerURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer nps.Close()

	svc := coap.New(tc, nps)

	svc = api.LoggingMiddleware(svc, logger)

	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "coap_adapter",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "coap_adapter",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	g.Go(func() error {
		return servers.StartHTTPServer(ctx, svcCoap, api.MakeHTTPHandler(), cfg.coapConfig, logger)
	})

	g.Go(func() error {
		return startCOAPServer(ctx, cfg, svc, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("CoAP adapter service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("CoAP adapter service terminated: %s", err))
	}
}

func loadConfig() config {
	coapConfig := servers.Config{
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

	thingsConfig := clients.Config{
		ClientTLS: tls,
		CaCerts:   mainflux.Env(envCACerts, defCACerts),
		GrpcURL:   mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
	}

	return config{
		coapConfig:        coapConfig,
		thingsConfig:      thingsConfig,
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
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
		logger.Error(fmt.Sprintf("Failed to init Jaeger client: %s", err))
		os.Exit(1)
	}

	return tracer, closer
}

func startCOAPServer(ctx context.Context, cfg config, svc coap.Service, l logger.Logger) error {
	p := fmt.Sprintf(":%s", cfg.coapConfig.Port)
	errCh := make(chan error)
	l.Info(fmt.Sprintf("CoAP adapter service started, exposed port %s", cfg.coapConfig.Port))
	go func() {
		errCh <- gocoap.ListenAndServe("udp", p, api.MakeCoAPHandler(svc, l))
	}()
	select {
	case <-ctx.Done():
		l.Info(fmt.Sprintf("CoAP adapter service shutdown of http at %s", p))
		return nil
	case err := <-errCh:
		return err
	}
}
