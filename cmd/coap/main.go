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
	"github.com/MainfluxLabs/mainflux/coap"
	"github.com/MainfluxLabs/mainflux/coap/api"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	gocoap "github.com/plgd-dev/go-coap/v3"
	coapnet "github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v3/options"
	udpclient "github.com/plgd-dev/go-coap/v3/udp/client"
	udpserver "github.com/plgd-dev/go-coap/v3/udp/server"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	stopWaitTime = 5 * time.Second
	svcName      = "coap-adapter"

	defPort              = "5683"
	defPlainEnabled      = "true"
	defDTLSPort          = "5684"
	defDTLSIdleTimeout   = "5m"
	defBrokerURL         = "nats://localhost:4222"
	defLogLevel          = "error"
	defClientTLS         = "false"
	defCACerts           = ""
	defJaegerURL         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"

	envPort              = "MF_COAP_ADAPTER_PORT"
	envPlainEnabled      = "MF_COAP_ADAPTER_PLAIN_ENABLED"
	envDTLSPort          = "MF_COAP_ADAPTER_DTLS_PORT"
	envDTLSIdleTimeout   = "MF_COAP_ADAPTER_DTLS_IDLE_TIMEOUT"
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
	plainEnabled      bool
	dtlsPort          string
	dtlsIdleTimeout   time.Duration
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
		log.Fatal(err)
	}

	tConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer tConn.Close()

	thingsTracer, thingsCloser := jaeger.Init("coap_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	tc := thingsapi.NewClient(tConn, thingsTracer, cfg.thingsGRPCTimeout)

	nps, err := nats.NewPubSub(cfg.brokerURL, "", logger)
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
		return servershttp.Start(ctx, api.MakeHTTPHandler(), cfg.coapConfig, logger)
	})

	g.Go(func() error {
		return startCOAPServers(ctx, cfg, svc, tc, logger)
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
		ServerName:   svcName,
		Port:         mainflux.Env(envPort, defPort),
		StopWaitTime: stopWaitTime,
	}

	plainEnabled, err := strconv.ParseBool(mainflux.Env(envPlainEnabled, defPlainEnabled))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envPlainEnabled)
	}

	dtlsIdleTimeout, err := time.ParseDuration(mainflux.Env(envDTLSIdleTimeout, defDTLSIdleTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envDTLSIdleTimeout, err.Error())
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
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envCACerts, defCACerts),
		URL:        mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		ClientName: clients.Things,
	}

	return config{
		coapConfig:        coapConfig,
		plainEnabled:      plainEnabled,
		dtlsPort:          mainflux.Env(envDTLSPort, defDTLSPort),
		dtlsIdleTimeout:   dtlsIdleTimeout,
		thingsConfig:      thingsConfig,
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
	}
}

func startCOAPServers(ctx context.Context, cfg config, svc coap.Service, things domain.ThingsClient,
	l logger.Logger) error {
	handler := options.WithMux(api.MakeCoAPHandler(svc, l))
	errorLogger := options.WithErrors(func(err error) {
		l.Warn(fmt.Sprintf("CoAP server error: %s", err))
	})
	errCh := make(chan error, 2)

	if cfg.plainEnabled {
		l.Info(fmt.Sprintf("CoAP adapter service started, exposed port %s", cfg.coapConfig.Port))
		go func() {
			errCh <- gocoap.ListenAndServeWithOptions("udp", ":"+cfg.coapConfig.Port,
				handler,
				errorLogger,
				noInactivityMonitor{},
			)
		}()
	}

	dtlsConfig := coapnet.NewDTLSServerOptions(api.DTLSOptions(things, l)...)
	l.Info(fmt.Sprintf("CoAP adapter DTLS service started, exposed port %s", cfg.dtlsPort))
	go func() {
		errCh <- gocoap.ListenAndServeDTLSWithOptions("udp", ":"+cfg.dtlsPort, dtlsConfig,
			handler,
			errorLogger,
			options.WithOnNewConn(api.OnNewDTLSConn(things, l)),
			options.WithInactivityMonitor(cfg.dtlsIdleTimeout, func(conn *udpclient.Conn) {
				_ = conn.Close()
			}),
		)
	}()

	select {
	case <-ctx.Done():
		l.Info("CoAP adapter service shutdown")
		return nil
	case err := <-errCh:
		return err
	}
}

// noInactivityMonitor keeps idle connections open, since observers wait for commands silently.
type noInactivityMonitor struct{}

func (noInactivityMonitor) UDPServerApply(cfg *udpserver.Config) {
	cfg.CreateInactivityMonitor = func() udpclient.InactivityMonitor {
		return inactivity.NewNilMonitor[*udpclient.Conn]()
	}
}
