package main

import (
	"context"
	"fmt"
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
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	mqttpub "github.com/MainfluxLabs/mainflux/pkg/messaging/mqtt"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	"github.com/MainfluxLabs/mainflux/pkg/ulid"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	"github.com/MainfluxLabs/mproxy/logger"
	mp "github.com/MainfluxLabs/mproxy/pkg/mqtt"
	"github.com/MainfluxLabs/mproxy/pkg/session"
	ws "github.com/MainfluxLabs/mproxy/pkg/websocket"
	"github.com/cenkalti/backoff/v4"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "mqtt-adapter"
	stopWaitTime = 5 * time.Second

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
	httpConfig        servers.Config
	authConfig        clients.Config
	thingsConfig      clients.Config
	targetHost        string
	targetPort        string
	timeout           time.Duration
	targetHealthCheck string
	wsPort            string
	httpTargetHost    string
	httpTargetPort    string
	httpTargetPath    string
	jaegerURL         string
	logLevel          string
	thingsGRPCTimeout time.Duration
	brokerURL         string
	instance          string
	esURL             string
	esPass            string
	esDB              string
	authCacheURL      string
	authPass          string
	authCacheDB       string
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

	conn := clientsgrpc.Connect(cfg.thingsConfig, logger)
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

	subjects := []string{
		brokers.SubjectSenML,
		brokers.SubjectJSON,
	}

	fwd := mqtt.NewForwarder(subjects, logger)
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

	thingsTracer, thingsCloser := jaeger.Init("mqtt_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	mqttTracer, closer := jaeger.Init(svcName, cfg.jaegerURL, logger)

	defer closer.Close()

	authTracer, authCloser := jaeger.Init("mqtt_auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	authConn := clientsgrpc.Connect(cfg.authConfig, logger)
	defer authConn.Close()

	usersAuth := authapi.NewClient(authConn, authTracer, cfg.authGRPCTimeout)
	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsGRPCTimeout)

	authClient := auth.New(ac, tc)

	svc := newService(usersAuth, tc, db, logger)

	// Event handler for MQTT hooks
	h := mqtt.NewHandler([]messaging.Publisher{np}, es, logger, authClient, svc)

	logger.Info(fmt.Sprintf("Starting MQTT proxy on port %s", cfg.port))
	g.Go(func() error {
		return proxyMQTT(ctx, cfg, logger, h)
	})

	logger.Info(fmt.Sprintf("Starting MQTT over WS  proxy on port %s", cfg.httpConfig.Port))
	g.Go(func() error {
		return proxyWS(ctx, cfg, logger, h)
	})

	g.Go(func() error {
		return servershttp.Start(ctx, mqttapihttp.MakeHandler(mqttTracer, svc, logger), cfg.httpConfig, logger)
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
		port:              mainflux.Env(envMQTTPort, defMQTTPort),
		httpConfig:        httpConfig,
		authConfig:        authConfig,
		thingsConfig:      thingsConfig,
		targetHost:        mainflux.Env(envTargetHost, defTargetHost),
		targetPort:        mainflux.Env(envTargetPort, defTargetPort),
		timeout:           mqttTimeout,
		targetHealthCheck: mainflux.Env(envTargetHealthCheck, defTargetHealthCheck),
		wsPort:            mainflux.Env(envWSPort, defWSPort),
		httpTargetHost:    mainflux.Env(envHTTPTargetHost, defHTTPTargetHost),
		httpTargetPort:    mainflux.Env(envHTTPTargetPort, defHTTPTargetPort),
		httpTargetPath:    mainflux.Env(envHTTPTargetPath, defHTTPTargetPath),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
		brokerURL:         mainflux.Env(envBrokerURL, defBrokerURL),
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		instance:          mainflux.Env(envInstance, defInstance),
		esURL:             mainflux.Env(envESURL, defESURL),
		esPass:            mainflux.Env(envESPass, defESPass),
		esDB:              mainflux.Env(envESDB, defESDB),
		authCacheURL:      mainflux.Env(envAuthCacheURL, defAuthcacheURL),
		authPass:          mainflux.Env(envAuthCachePass, defAuthCachePass),
		authCacheDB:       mainflux.Env(envAuthCacheDB, defAuthCacheDB),
		authGRPCTimeout:   authGRPCTimeout,
		dbConfig:          dbConfig,
	}
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

func newService(ac protomfx.AuthServiceClient, tc protomfx.ThingsServiceClient, db *sqlx.DB, logger logger.Logger) mqtt.Service {
	subscriptions := postgres.NewRepository(db)
	idp := ulid.New()
	svc := mqtt.NewMqttService(ac, tc, subscriptions, idp)

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
