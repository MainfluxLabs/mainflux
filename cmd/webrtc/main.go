// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux"
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	"github.com/MainfluxLabs/mainflux/webrtc"
	"github.com/MainfluxLabs/mainflux/webrtc/api"
	httpapi "github.com/MainfluxLabs/mainflux/webrtc/api/http"
	"github.com/MainfluxLabs/mainflux/webrtc/pion"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "webrtc"
	stopWaitTime = 5 * time.Second

	defBrokerURL         = "nats://localhost:4222"
	defLogLevel          = "error"
	defClientTLS         = "false"
	defCACerts           = ""
	defHTTPPort          = "9032"
	defJaegerURL         = ""
	defServerCert        = ""
	defServerKey         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defAuthGRPCURL       = "localhost:8181"
	defAuthGRPCTimeout   = "1s"
	defUDPPort           = "50000"
	defPublicIPs         = ""
	defSTUNURLs          = "stun:stun.l.google.com:19302"
	defTURNURLs          = ""
	defTURNSecret        = ""
	defTURNTTL           = "24h"

	envBrokerURL         = "MF_BROKER_URL"
	envLogLevel          = "MF_WEBRTC_LOG_LEVEL"
	envClientTLS         = "MF_WEBRTC_CLIENT_TLS"
	envCACerts           = "MF_WEBRTC_CA_CERTS"
	envHTTPPort          = "MF_WEBRTC_HTTP_PORT"
	envServerCert        = "MF_WEBRTC_SERVER_CERT"
	envServerKey         = "MF_WEBRTC_SERVER_KEY"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envAuthGRPCURL       = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout   = "MF_AUTH_GRPC_TIMEOUT"
	envUDPPort           = "MF_WEBRTC_UDP_PORT"
	envPublicIPs         = "MF_WEBRTC_PUBLIC_IPS"
	envSTUNURLs          = "MF_WEBRTC_STUN_URLS"
	envTURNURLs          = "MF_WEBRTC_TURN_URLS"
	envTURNSecret        = "MF_WEBRTC_TURN_SECRET"
	envTURNTTL           = "MF_WEBRTC_TURN_TTL"
)

type config struct {
	brokerURL         string
	logLevel          string
	httpConfig        servers.Config
	thingsConfig      clients.Config
	authConfig        clients.Config
	jaegerURL         string
	thingsGRPCTimeout time.Duration
	authGRPCTimeout   time.Duration
	udpPort           string
	publicIPs         []string
	webrtc            webrtc.Config
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err.Error())
	}

	pubSub, err := nats.NewPubSub(cfg.brokerURL, svcName, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	tracer, closer := jaeger.Init(svcName, cfg.jaegerURL, logger)
	defer closer.Close()

	thingsTracer, thingsCloser := jaeger.Init("webrtc_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	thingsConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer thingsConn.Close()

	things := thingsapi.NewClient(thingsConn, thingsTracer, cfg.thingsGRPCTimeout)

	authTracer, authCloser := jaeger.Init("webrtc_auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	authConn := clientsgrpc.Connect(cfg.authConfig, logger)
	defer authConn.Close()

	auth := authapi.NewClient(authConn, authTracer, cfg.authGRPCTimeout)

	sfu, err := pion.New(pion.Config{
		ListenAddr: ":" + cfg.udpPort,
		ICEServers: cfg.webrtc.STUNURLs,
		PublicIPs:  cfg.publicIPs,
	})
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create SFU: %s", err))
		os.Exit(1)
	}
	defer sfu.Close()

	svc := webrtc.New(things, auth, sfu, pubSub, uuid.New(), cfg.webrtc)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "webrtc",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "webrtc",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	if err := consumers.Messages(svcName, pubSub, svc, nats.SubjectWebRTC); err != nil {
		logger.Error(fmt.Sprintf("Failed to subscribe to webrtc messages: %s", err))
		os.Exit(1)
	}

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(tracer, svc, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("WebRTC service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("WebRTC service terminated: %s", err))
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

	authGRPCTimeout, err := time.ParseDuration(mainflux.Env(envAuthGRPCTimeout, defAuthGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthGRPCTimeout, err.Error())
	}

	turnTTL, err := time.ParseDuration(mainflux.Env(envTURNTTL, defTURNTTL))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envTURNTTL, err.Error())
	}

	httpConfig := servers.Config{
		ServerName:   svcName,
		ServerCert:   mainflux.Env(envServerCert, defServerCert),
		ServerKey:    mainflux.Env(envServerKey, defServerKey),
		Port:         mainflux.Env(envHTTPPort, defHTTPPort),
		StopWaitTime: stopWaitTime,
	}

	thingsConfig := clients.Config{
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envCACerts, defCACerts),
		URL:        mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		ClientName: clients.Things,
	}

	authConfig := clients.Config{
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envCACerts, defCACerts),
		URL:        mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		ClientName: clients.Auth,
	}

	return config{
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		httpConfig:        httpConfig,
		thingsConfig:      thingsConfig,
		authConfig:        authConfig,
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
		authGRPCTimeout:   authGRPCTimeout,
		udpPort:           mainflux.Env(envUDPPort, defUDPPort),
		publicIPs:         split(mainflux.Env(envPublicIPs, defPublicIPs)),
		webrtc: webrtc.Config{
			STUNURLs:   split(mainflux.Env(envSTUNURLs, defSTUNURLs)),
			TURNURLs:   split(mainflux.Env(envTURNURLs, defTURNURLs)),
			TURNSecret: mainflux.Env(envTURNSecret, defTURNSecret),
			TURNTTL:    turnTTL,
		},
	}
}

func split(s string) []string {
	var out []string
	for v := range strings.SplitSeq(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
