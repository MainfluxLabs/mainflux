//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

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

	"github.com/mainflux/mainflux"
	authapi "github.com/mainflux/mainflux/auth/api/grpc"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/re"
	"github.com/mainflux/mainflux/re/api"
	rehttpapi "github.com/mainflux/mainflux/re/api/re/http"
	localusers "github.com/mainflux/mainflux/things/users"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	mfSDK "github.com/mainflux/mainflux/pkg/sdk/go"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
)

const (
	defLogLevel        = "info"
	defHTTPPort        = "9099"
	defKuiperURL       = "http://localhost:9081"
	defJaegerURL       = ""
	defServerCert      = ""
	defServerKey       = ""
	defSingleUserEmail = ""
	defSingleUserToken = ""
	defAuthURL         = "localhost:8181"
	defAuthTimeout     = "1s"
	defClientTLS       = "false"
	defCACerts         = ""
	defThingsLocation  = "http://localhost"
	defMfBSURL         = "http://localhost:8202/things/configs"
	defMfCertsURL      = "http://localhost:8204"
	defTLS             = "false"

	envLogLevel        = "MF_RE_LOG_LEVEL"
	envHTTPPort        = "MF_RE_HTTP_PORT"
	envKuiperURL       = "MF_KUIPER_URL"
	envJaegerURL       = "MF_JAEGER_URL"
	envServerCert      = "MF_RE_SERVER_CERT"
	envServerKey       = "MF_RE_SERVER_KEY"
	envSingleUserEmail = "MF_RE_SINGLE_USER_EMAIL"
	envSingleUserToken = "MF_RE_SINGLE_USER_TOKEN"
	envAuthURL         = "MF_AUTH_GRPC_URL"
	envAuthTimeout     = "MF_AUTH_GRPC_TIMEOUT"
	envClientTLS       = "MF_RE_CLIENT_TLS"
	envCACerts         = "MF_RE_CA_CERTS"
	envThingsLocation  = "MF_RE_THINGS_LOCATION"
	envMfBSURL         = "MF_RE_BS_SVC_URL"
	envMfCertsURL      = "MF_RE_CERTS_SVC_URL"
	envTLS             = "MF_RE_ENV_CLIENTS_TLS"
)

type config struct {
	logLevel        string
	httpPort        string
	kuiperURL       string
	authHTTPPort    string
	authGRPCPort    string
	jaegerURL       string
	serverCert      string
	serverKey       string
	singleUserEmail string
	singleUserToken string
	authURL         string
	authTimeout     time.Duration
	clientTLS       bool
	caCerts         string

	ThingsLocation string
	MfBSURL        string
	MfCertsURL     string
	TLS            bool
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	authTracer, authCloser := initJaeger("auth", cfg.jaegerURL, logger)
	defer authCloser.Close()
	auth, _ := createAuthClient(cfg, authTracer, logger)

	tracer, reCloser := initJaeger("re", cfg.jaegerURL, logger)
	defer reCloser.Close()

	SDKCfg := mfSDK.Config{
		BaseURL:           cfg.ThingsLocation,
		BootstrapURL:      cfg.MfBSURL,
		CertsURL:          cfg.MfCertsURL,
		HTTPAdapterPrefix: "http",
		MsgContentType:    "application/json",
		TLSVerification:   cfg.TLS,
	}
	SDK := mfSDK.NewSDK(SDKCfg)

	svc := newService(cfg.kuiperURL, auth, SDK, logger)
	errs := make(chan error, 2)

	go startHTTPServer(rehttpapi.MakeHandler(tracer, svc), cfg.httpPort, cfg, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Re service terminated: %s", err))
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	authTimeout, err := time.ParseDuration(mainflux.Env(envAuthTimeout, defAuthTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthTimeout, err.Error())
	}

	return config{
		logLevel:        mainflux.Env(envLogLevel, defLogLevel),
		httpPort:        mainflux.Env(envHTTPPort, defHTTPPort),
		kuiperURL:       mainflux.Env(envKuiperURL, defKuiperURL),
		serverCert:      mainflux.Env(envServerCert, defServerCert),
		serverKey:       mainflux.Env(envServerKey, defServerKey),
		jaegerURL:       mainflux.Env(envJaegerURL, defJaegerURL),
		singleUserEmail: mainflux.Env(envSingleUserEmail, defSingleUserEmail),
		singleUserToken: mainflux.Env(envSingleUserToken, defSingleUserToken),
		authURL:         mainflux.Env(envAuthURL, defAuthURL),
		authTimeout:     authTimeout,
		clientTLS:       tls,
		caCerts:         mainflux.Env(envCACerts, defCACerts),
		MfBSURL:         mainflux.Env(envMfBSURL, defMfBSURL),
		MfCertsURL:      mainflux.Env(envMfCertsURL, defMfCertsURL),
		ThingsLocation:  mainflux.Env(envThingsLocation, defThingsLocation),
		TLS:             tls,
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

func newService(kuiperURL string, auth mainflux.AuthServiceClient, sdk mfSDK.SDK, logger logger.Logger) re.Service {
	svc := re.New(kuiperURL, auth, sdk, logger)

	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "re",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "re",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}

func startHTTPServer(handler http.Handler, port string, cfg config, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	if cfg.serverCert != "" || cfg.serverKey != "" {
		logger.Info(fmt.Sprintf("Re service started using https on port %s with cert %s key %s",
			port, cfg.serverCert, cfg.serverKey))
		errs <- http.ListenAndServeTLS(p, cfg.serverCert, cfg.serverKey, handler)
		return
	}
	logger.Info(fmt.Sprintf("Re service started using http on port %s", cfg.httpPort))
	errs <- http.ListenAndServe(p, handler)
}

func createAuthClient(cfg config, tracer opentracing.Tracer, logger logger.Logger) (mainflux.AuthServiceClient, func() error) {
	if cfg.singleUserEmail != "" && cfg.singleUserToken != "" {
		return localusers.NewSingleUserService(cfg.singleUserEmail, cfg.singleUserToken), nil
	}

	conn := connectToAuth(cfg, logger)
	return authapi.NewClient(tracer, conn, cfg.authTimeout), conn.Close
}

func connectToAuth(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
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

	conn, err := grpc.Dial(cfg.authURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to auth service: %s", err))
		os.Exit(1)
	}

	return conn
}
