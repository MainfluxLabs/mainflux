package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/readers/api"
	"github.com/MainfluxLabs/mainflux/readers/influxdb"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	stopWaitTime = 5 * time.Second
	svcName      = "influxdb-reader"

	defLogLevel          = "error"
	defPort              = "8180"
	defDB                = "mainflux"
	defDBHost            = "localhost"
	defDBPort            = "8086"
	defDBUser            = "mainflux"
	defDBPass            = "mainflux"
	defDBBucket          = "mainflux-bucket"
	defDBOrg             = "mainflux"
	defDBToken           = "mainflux-token"
	defClientTLS         = "false"
	defCACerts           = ""
	defServerCert        = ""
	defServerKey         = ""
	defJaegerURL         = ""
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"
	defAuthGRPCURL       = "localhost:8181"
	defAuthGRPCTimeout   = "1s"

	envLogLevel          = "MF_INFLUX_READER_LOG_LEVEL"
	envPort              = "MF_INFLUX_READER_PORT"
	envDB                = "MF_INFLUXDB_DB"
	envDBHost            = "MF_INFLUXDB_HOST"
	envDBPort            = "MF_INFLUXDB_PORT"
	envDBUser            = "MF_INFLUXDB_ADMIN_USER"
	envDBPass            = "MF_INFLUXDB_ADMIN_PASSWORD"
	envDBBucket          = "MF_INFLUXDB_BUCKET"
	envDBOrg             = "MF_INFLUXDB_ORG"
	envDBToken           = "MF_INFLUXDB_TOKEN"
	envClientTLS         = "MF_INFLUX_READER_CLIENT_TLS"
	envCACerts           = "MF_INFLUX_READER_CA_CERTS"
	envServerCert        = "MF_INFLUX_READER_SERVER_CERT"
	envServerKey         = "MF_INFLUX_READER_SERVER_KEY"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsGRPCURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	envAuthGRPCURL       = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout   = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	logLevel          string
	httpConfig        servers.Config
	authConfig        clients.Config
	thingsConfig      clients.Config
	dbName            string
	dbHost            string
	dbPort            string
	dbUser            string
	dbPass            string
	dbBucket          string
	dbOrg             string
	dbToken           string
	dbUrl             string
	jaegerURL         string
	thingsGRPCTimeout time.Duration
	authGRPCTimeout   time.Duration
}

func main() {
	cfg, repoCfg := loadConfigs()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}
	conn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer conn.Close()

	thingsTracer, thingsCloser := jaeger.Init("influxdb_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsGRPCTimeout)

	authTracer, authCloser := jaeger.Init("influxdb_auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	authConn := clientsgrpc.Connect(cfg.authConfig, logger)
	defer authConn.Close()

	auth := authapi.NewClient(authConn, authTracer, cfg.authGRPCTimeout)

	client, err := connectToInfluxDB(cfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create InfluxDB client: %s", err))
		os.Exit(1)
	}
	defer client.Close()

	repo := newService(client, repoCfg, logger)

	g.Go(func() error {
		return servershttp.Start(ctx, api.MakeHandler(repo, tc, auth, svcName, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("InfluxDB reader service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("InfluxDB reader service terminated: %s", err))
	}
}

func connectToInfluxDB(cfg config) (influxdb2.Client, error) {
	client := influxdb2.NewClientWithOptions(cfg.dbUrl, cfg.dbToken, influxdb2.DefaultOptions().SetHTTPRequestTimeout(90))
	_, err := client.Ping(context.Background())
	return client, err
}

func loadConfigs() (config, influxdb.RepoConfig) {
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

	httpConfig := servers.Config{
		ServerName:   svcName,
		ServerCert:   mainflux.Env(envServerCert, defServerCert),
		ServerKey:    mainflux.Env(envServerKey, defServerKey),
		Port:         mainflux.Env(envPort, defPort),
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

	cfg := config{
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		httpConfig:        httpConfig,
		thingsConfig:      thingsConfig,
		authConfig:        authConfig,
		dbName:            mainflux.Env(envDB, defDB),
		dbHost:            mainflux.Env(envDBHost, defDBHost),
		dbPort:            mainflux.Env(envDBPort, defDBPort),
		dbUser:            mainflux.Env(envDBUser, defDBUser),
		dbPass:            mainflux.Env(envDBPass, defDBPass),
		dbBucket:          mainflux.Env(envDBBucket, defDBBucket),
		dbOrg:             mainflux.Env(envDBOrg, defDBOrg),
		dbToken:           mainflux.Env(envDBToken, defDBToken),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsGRPCTimeout: thingsGRPCTimeout,
		authGRPCTimeout:   authGRPCTimeout,
	}

	cfg.dbUrl = fmt.Sprintf("http://%s:%s", cfg.dbHost, cfg.dbPort)

	repoCfg := influxdb.RepoConfig{
		Bucket: cfg.dbBucket,
		Org:    cfg.dbOrg,
	}
	return cfg, repoCfg
}

func newService(client influxdb2.Client, repoCfg influxdb.RepoConfig, logger logger.Logger) readers.MessageRepository {
	repo := influxdb.New(client, repoCfg)
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(
		repo,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "influxdb",
			Subsystem: "message_reader",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "influxdb",
			Subsystem: "message_reader",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return repo
}
