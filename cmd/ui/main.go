// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"google.golang.org/grpc/credentials"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	thingsapi "github.com/mainflux/mainflux/things/api/auth/grpc"
	"github.com/mainflux/mainflux/ui"
	"github.com/mainflux/mainflux/ui/api"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
)

const (
	defLogLevel          = "info"
	defClientTLS         = "false"
	defCACerts           = ""
	defPort              = "9090"
	defRedirectURL       = "http://localhost:9090/"
	defJaegerURL         = ""
	defThingsAuthURL     = "localhost:8181"
	defAuthURL           = "localhost:8189"
	defThingsAuthTimeout = "1s"

	envLogLevel          = "MF_GUI_LOG_LEVEL"
	envClientTLS         = "MF_GUI_CLIENT_TLS"
	envCACerts           = "MF_GUI_CA_CERTS"
	envPort              = "MF_GUI_PORT"
	envRedirectURL       = "MF_GUI_REDIRECT_URL"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsAuthURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsAuthTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
)

type config struct {
	logLevel          string
	port              string
	redirectURL       string
	clientTLS         bool
	caCerts           string
	jaegerURL         string
	thingsAuthURL     string
	thingsAuthTimeout time.Duration
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	conn := connectToThings(cfg, logger)
	defer conn.Close()

	tracer, closer := initJaeger("ui", cfg.jaegerURL, logger)
	defer closer.Close()

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	msgContentType := string(sdk.CTJSONSenML)
	sdkConf := sdk.Config{
		ThingsURL:       "http://localhost:8182",
		AuthURL:         "http://localhost:8189",
		MsgContentType:  sdk.ContentType(msgContentType),
		TLSVerification: false,
	}
	sdk := sdk.NewSDK(sdkConf)

	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsAuthTimeout)

	svc := ui.New(tc, sdk)

	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "ui",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "ui",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	errs := make(chan error, 2)

	go func() {
		p := fmt.Sprintf(":%s", cfg.port)
		logger.Info(fmt.Sprintf("GUI service started on port %s", cfg.port))
		errs <- http.ListenAndServe(p, api.MakeHandler(svc, cfg.redirectURL, tracer))
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("GUI service terminated: %s", err))
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	authTimeout, err := time.ParseDuration(mainflux.Env(envThingsAuthTimeout, defThingsAuthTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsAuthTimeout, err.Error())
	}

	return config{
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		port:              mainflux.Env(envPort, defPort),
		redirectURL:       mainflux.Env(envRedirectURL, defRedirectURL),
		clientTLS:         tls,
		caCerts:           mainflux.Env(envCACerts, defCACerts),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsAuthURL:     mainflux.Env(envThingsAuthURL, defThingsAuthURL),
		thingsAuthTimeout: authTimeout,
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

	conn, err := grpc.Dial(cfg.thingsAuthURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to things service: %s", err))
		os.Exit(1)
	}
	return conn
}
