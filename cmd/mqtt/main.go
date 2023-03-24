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
	"time"

	"github.com/MainfluxLabs/mainflux"
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/mqtt"
	mqttapi "github.com/MainfluxLabs/mainflux/mqtt/api"
	mqttapihttp "github.com/MainfluxLabs/mainflux/mqtt/api/http"
	"github.com/MainfluxLabs/mainflux/mqtt/postgres"
	mqttredis "github.com/MainfluxLabs/mainflux/mqtt/redis"
	"github.com/MainfluxLabs/mainflux/pkg/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	mqttpub "github.com/MainfluxLabs/mainflux/pkg/messaging/mqtt"
	"github.com/MainfluxLabs/mainflux/pkg/ulid"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/auth/grpc"
	"github.com/MainfluxLabs/mproxy/logger"
	mp "github.com/MainfluxLabs/mproxy/pkg/mqtt"
	"github.com/MainfluxLabs/mproxy/pkg/session"
	ws "github.com/MainfluxLabs/mproxy/pkg/websocket"
	"github.com/cenkalti/backoff/v4"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	svcName       = "mqtt"
	httpProtocol  = "http"
	httpsProtocol = "https"
	stopWaitTime  = 5 * time.Second

	defLogLevel          = "error"
	defMQTTPort          = "1883"
	defTargetHost        = "0.0.0.0"
	defTargetPort        = "1883"
	defTimeout           = "30s" // 30 seconds
	defTargetHealthCheck = ""
	defHTTPPort          = "8080"
	defHTTPTargetHost    = "localhost"
	defHTTPTargetPort    = "8080"
	defHTTPTargetPath    = "/mqtt"
	defWSPort            = "8285"
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defBrokerURL         = "nats://localhost:4222"
	defJaegerURL         = ""
	defClientTLS         = "false"
	defCACerts           = ""
	defInstance          = ""
	defESURL             = "localhost:6379"
	defESPass            = ""
	defESDB              = "0"
	defAuthcacheURL      = "localhost:6379"
	defAuthCachePass     = ""
	defAuthCacheDB       = "0"
	defDBHost            = "localhost"
	defAuthGRPCURL       = "localhost:8181"
	defDBPort            = "5432"
	defDBUser            = "mainflux"
	defDBPass            = "mainflux"
	defDB                = "subscriptions"
	defDBSSLMode         = "disable"
	defDBSSLCert         = ""
	defDBSSLKey          = ""
	defDBSSLRootCert     = ""
	defServerKey         = ""
	defServerCert        = ""
	defAuthGRPCTimeout   = "1s"

	envLogLevel          = "MF_MQTT_ADAPTER_LOG_LEVEL"
	envMQTTPort          = "MF_MQTT_ADAPTER_MQTT_PORT"
	envTargetHost        = "MF_MQTT_ADAPTER_MQTT_TARGET_HOST"
	envTargetPort        = "MF_MQTT_ADAPTER_MQTT_TARGET_PORT"
	envTargetHealthCheck = "MF_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK"
	envTimeout           = "MF_MQTT_ADAPTER_FORWARDER_TIMEOUT"
	envHTTPPort          = "MF_MQTT_ADAPTER_HTTP_PORT"
	envHTTPTargetHost    = "MF_MQTT_ADAPTER_WS_TARGET_HOST"
	envHTTPTargetPort    = "MF_MQTT_ADAPTER_WS_TARGET_PORT"
	envHTTPTargetPath    = "MF_MQTT_ADAPTER_WS_TARGET_PATH"
	envWSPort            = "MF_MQTT_ADAPTER_WS_PORT"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envBrokerURL         = "MF_BROKER_URL"
	envJaegerURL         = "MF_JAEGER_URL"
	envClientTLS         = "MF_MQTT_ADAPTER_CLIENT_TLS"
	envCACerts           = "MF_MQTT_ADAPTER_CA_CERTS"
	envInstance          = "MF_MQTT_ADAPTER_INSTANCE"
	envESURL             = "MF_MQTT_ADAPTER_ES_URL"
	envESPass            = "MF_MQTT_ADAPTER_ES_PASS"
	envESDB              = "MF_MQTT_ADAPTER_ES_DB"
	envAuthCacheURL      = "MF_AUTH_CACHE_URL"
	envAuthCachePass     = "MF_AUTH_CACHE_PASS"
	envAuthCacheDB       = "MF_AUTH_CACHE_DB"
	envServerCert        = "MF_MQTT_ADAPTER_SERVER_CERT"
	envServerKey         = "MF_MQTT_ADAPTER_SERVER_KEY"
	envDBHost            = "MF_MQTT_ADAPTER_DB_HOST"
	envDBPort            = "MF_MQTT_ADAPTER_DB_PORT"
	envDBUser            = "MF_MQTT_ADAPTER_DB_USER"
	envDBPass            = "MF_MQTT_ADAPTER_DB_PASS"
	envDB                = "MF_MQTT_ADAPTER_DB"
	envDBSSLMode         = "MF_MQTT_ADAPTER_DB_SSL_MODE"
	envDBSSLCert         = "MF_MQTT_ADAPTER_DB_SSL_CERT"
	envDBSSLKey          = "MF_MQTT_ADAPTER_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_MQTT_ADAPTER_DB_SSL_ROOT_CERT"
	envAuthGRPCURL       = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout   = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	port              string
	targetHost        string
	targetPort        string
	timeout           time.Duration
	targetHealthCheck string
	httpPort          string
	wsPort            string
	httpTargetHost    string
	httpTargetPort    string
	httpTargetPath    string
	jaegerURL         string
	logLevel          string
	thingsGRPCURL     string
	thingsGRPCTimeout time.Duration
	brokerURL         string
	authGRPCURL       string
	clientTLS         bool
	caCerts           string
	instance          string
	esURL             string
	esPass            string
	esDB              string
	authCacheURL      string
	authPass          string
	authCacheDB       string
	serverCert        string
	serverKey         string
	authGRPCTimeout   time.Duration
	dbConfig          postgres.Config
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	if cfg.targetHealthCheck != "" {
		notify := func(e error, next time.Duration) {
			logger.Info(fmt.Sprintf("Broker not ready: %s, next try in %s", e.Error(), next))
		}

		err := backoff.RetryNotify(healthcheck(cfg), backoff.NewExponentialBackOff(), notify)
		if err != nil {
			logger.Info(fmt.Sprintf("MQTT healthcheck limit exceeded, exiting. %s ", err.Error()))
			os.Exit(1)
		}
	}

	conn := connectToThings(cfg, logger)
	defer conn.Close()

	ec := connectToRedis(cfg.esURL, cfg.esPass, cfg.esDB, logger)
	defer ec.Close()

	nps, err := brokers.NewPubSub(cfg.brokerURL, "mqtt", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer nps.Close()

	mpub, err := mqttpub.NewPublisher(fmt.Sprintf("%s:%s", cfg.targetHost, cfg.targetPort), cfg.timeout)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create MQTT publisher: %s", err))
		os.Exit(1)
	}

	fwd := mqtt.NewForwarder(brokers.SubjectAllChannels, logger)
	if err := fwd.Forward(svcName, nps, mpub); err != nil {
		logger.Error(fmt.Sprintf("Failed to forward message broker messages: %s", err))
		os.Exit(1)
	}

	np, err := brokers.NewPublisher(cfg.brokerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer np.Close()

	es := mqttredis.NewEventStore(ec, cfg.instance)

	ac := connectToRedis(cfg.authCacheURL, cfg.authPass, cfg.authCacheDB, logger)
	defer ac.Close()

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	tracer, closer := initJaeger("mqtt_adapter", cfg.jaegerURL, logger)

	defer closer.Close()

	usersAuthTracer, authCloser := initJaeger("auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	usersAuthConn := connectToAuth(cfg, logger)
	defer usersAuthConn.Close()

	usersAuth := authapi.NewClient(usersAuthTracer, usersAuthConn, cfg.authGRPCTimeout)
	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsGRPCTimeout)

	authClient := auth.New(ac, tc)

	svc := newService(usersAuth, db, logger)

	// Event handler for MQTT hooks
	h := mqtt.NewHandler([]messaging.Publisher{np}, es, logger, authClient, svc)

	logger.Info(fmt.Sprintf("Starting MQTT proxy on port %s", cfg.port))
	g.Go(func() error {
		return proxyMQTT(ctx, cfg, logger, h)
	})

	logger.Info(fmt.Sprintf("Starting MQTT over WS  proxy on port %s", cfg.httpPort))
	g.Go(func() error {
		return proxyWS(ctx, cfg, logger, h)
	})

	errs := make(chan error, 2)
	g.Go(func() error {
		return startHTTPServer(ctx, svc, tracer, cfg, logger, errs)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("mProxy shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("mProxy terminated: %s", err))
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

	mqttTimeout, err := time.ParseDuration(mainflux.Env(envTimeout, defTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envTimeout, err.Error())
	}

	authGRPCTimeout, err := time.ParseDuration(mainflux.Env(envAuthGRPCTimeout, defAuthGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthGRPCTimeout, err.Error())
	}

	dbConfig := postgres.Config{
		Host:        mainflux.Env(envDBHost, defDBHost),
		Port:        mainflux.Env(envDBPort, defDBPort),
		User:        mainflux.Env(envDBUser, defDBUser),
		Pass:        mainflux.Env(envDBPass, defDBPass),
		Name:        mainflux.Env(envDB, defDB),
		SSLMode:     mainflux.Env(envDBSSLMode, defDBSSLMode),
		SSLCert:     mainflux.Env(envDBSSLCert, defDBSSLCert),
		SSLKey:      mainflux.Env(envDBSSLKey, defDBSSLKey),
		SSLRootCert: mainflux.Env(envDBSSLRootCert, defDBSSLRootCert),
	}

	return config{
		port:              mainflux.Env(envMQTTPort, defMQTTPort),
		targetHost:        mainflux.Env(envTargetHost, defTargetHost),
		targetPort:        mainflux.Env(envTargetPort, defTargetPort),
		timeout:           mqttTimeout,
		targetHealthCheck: mainflux.Env(envTargetHealthCheck, defTargetHealthCheck),
		httpPort:          mainflux.Env(envHTTPPort, defHTTPPort),
		wsPort:            mainflux.Env(envWSPort, defWSPort),
		httpTargetHost:    mainflux.Env(envHTTPTargetHost, defHTTPTargetHost),
		httpTargetPort:    mainflux.Env(envHTTPTargetPort, defHTTPTargetPort),
		httpTargetPath:    mainflux.Env(envHTTPTargetPath, defHTTPTargetPath),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCURL:     mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		authGRPCURL:       mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		clientTLS:         tls,
		caCerts:           mainflux.Env(envCACerts, defCACerts),
		instance:          mainflux.Env(envInstance, defInstance),
		esURL:             mainflux.Env(envESURL, defESURL),
		esPass:            mainflux.Env(envESPass, defESPass),
		esDB:              mainflux.Env(envESDB, defESDB),
		authCacheURL:      mainflux.Env(envAuthCacheURL, defAuthcacheURL),
		authPass:          mainflux.Env(envAuthCachePass, defAuthCachePass),
		authCacheDB:       mainflux.Env(envAuthCacheDB, defAuthCacheDB),
		serverCert:        mainflux.Env(envServerCert, defServerCert),
		serverKey:         mainflux.Env(envServerKey, defServerKey),
		authGRPCTimeout:   authGRPCTimeout,
		dbConfig:          dbConfig,
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

	conn, err := grpc.Dial(cfg.thingsGRPCURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to things service: %s", err))
		os.Exit(1)
	}
	return conn
}

func connectToRedis(redisURL, redisPass, redisDB string, logger logger.Logger) *redis.Client {
	db, err := strconv.Atoi(redisDB)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to redis: %s", err))
		os.Exit(1)
	}

	return redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: redisPass,
		DB:       db,
	})
}

func proxyMQTT(ctx context.Context, cfg config, logger logger.Logger, handler session.Handler) error {
	address := fmt.Sprintf(":%s", cfg.port)
	target := fmt.Sprintf("%s:%s", cfg.targetHost, cfg.targetPort)
	mp := mp.New(address, target, handler, logger)

	errCh := make(chan error)
	go func() {
		errCh <- mp.Listen()
	}()

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy MQTT shutdown at %s", target))
		return nil
	case err := <-errCh:
		return err
	}

}
func proxyWS(ctx context.Context, cfg config, logger logger.Logger, handler session.Handler) error {
	target := fmt.Sprintf("%s:%s", cfg.httpTargetHost, cfg.httpTargetPort)
	wp := ws.New(target, cfg.httpTargetPath, "ws", handler, logger)
	http.Handle("/mqtt", wp.Handler())

	errCh := make(chan error)

	go func() {
		errCh <- wp.Listen(cfg.wsPort)
	}()

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy MQTT WS shutdown at %s", target))
		return nil
	case err := <-errCh:
		return err
	}
}

func healthcheck(cfg config) func() error {
	return func() error {
		res, err := http.Get(cfg.targetHealthCheck)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		if res.StatusCode != http.StatusOK {
			return errors.New(string(body))
		}
		return nil
	}
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func newService(ac mainflux.AuthServiceClient, db *sqlx.DB, logger logger.Logger) mqtt.Service {
	subscriptions := postgres.NewRepository(db)
	idp := ulid.New()
	svc := mqtt.NewMqttService(ac, subscriptions, idp)

	svc = mqttapi.LoggingMiddleware(svc, logger)
	svc = mqttapi.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "mqtt_adapter",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "mqtt_adapter",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}

func startHTTPServer(ctx context.Context, svc mqtt.Service, tracer opentracing.Tracer, cfg config, logger logger.Logger, errs chan error) error {
	p := fmt.Sprintf(":%s", cfg.httpPort)
	errCh := make(chan error)
	protocol := httpProtocol
	server := &http.Server{Addr: p, Handler: mqttapihttp.MakeHandler(tracer, svc, logger)}

	switch {
	case cfg.serverCert != "" || cfg.serverKey != "":
		logger.Info(fmt.Sprintf("mqtt-adapter service started using https on port %s with cert %s key %s",
			cfg.httpPort, cfg.serverCert, cfg.serverKey))
		go func() {
			errCh <- server.ListenAndServeTLS(cfg.serverCert, cfg.serverKey)
		}()
		protocol = httpsProtocol

	default:
		logger.Info(fmt.Sprintf("mqtt-adapter service started using http on port %s", cfg.httpPort))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("mqtt-adapter %s service error occurred during shutdown at %s: %s", protocol, p, err))
			return fmt.Errorf("mqtt-adapter %s service error occurred during shutdown at %s: %w", protocol, p, err)
		}
		logger.Info(fmt.Sprintf("mqtt-adapter %s service shutdown of http at %s", protocol, p))
		return nil
	case err := <-errCh:
		return err
	}
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

	conn, err := grpc.Dial(cfg.authGRPCURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to auth service: %s", err))
		os.Exit(1)
	}

	return conn
}
