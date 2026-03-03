package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/auth"
	api "github.com/MainfluxLabs/mainflux/auth/api"
	httpapi "github.com/MainfluxLabs/mainflux/auth/api/http"
	"github.com/MainfluxLabs/mainflux/auth/emailer"
	"github.com/MainfluxLabs/mainflux/auth/jwt"
	"github.com/MainfluxLabs/mainflux/auth/postgres"
	rediscache "github.com/MainfluxLabs/mainflux/auth/redis"
	"github.com/MainfluxLabs/mainflux/auth/tracing"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/email"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	serversgrpc "github.com/MainfluxLabs/mainflux/pkg/servers/grpc"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	usersapi "github.com/MainfluxLabs/mainflux/users/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	stopWaitTime = 5 * time.Second
	svcName      = "auth"

	defLogLevel        = "error"
	defDBHost          = "localhost"
	defDBPort          = "5432"
	defDBUser          = "mainflux"
	defDBPass          = "mainflux"
	defDB              = "auth"
	defDBSSLMode       = "disable"
	defDBSSLCert       = ""
	defDBSSLKey        = ""
	defDBSSLRootCert   = ""
	defHTTPPort        = "8180"
	defGRPCPort        = "8181"
	defSecret          = "auth"
	defServerCert      = ""
	defServerKey       = ""
	defJaegerURL       = ""
	defLoginDuration   = "10h"
	defInviteDuration  = "168h"
	defAdminEmail      = ""
	defTimeout         = "1s"
	defThingsGRPCURL   = "localhost:8183"
	defThingsCACerts   = ""
	defThingsClientTLS = "false"
	defUsersCACerts    = ""
	defUsersClientTLS  = "false"
	defUsersGRPCURL    = "localhost:8184"
	defESURL           = "redis://localhost:6379/0"

	defEmailHost         = "localhost"
	defEmailPort         = "25"
	defEmailUsername     = "root"
	defEmailPassword     = ""
	defEmailFromAddress  = ""
	defEmailFromName     = ""
	defEmailBaseTemplate = "base.tmpl"

	defHost = "http://localhost"

	envLogLevel        = "MF_AUTH_LOG_LEVEL"
	envDBHost          = "MF_AUTH_DB_HOST"
	envDBPort          = "MF_AUTH_DB_PORT"
	envDBUser          = "MF_AUTH_DB_USER"
	envDBPass          = "MF_AUTH_DB_PASS"
	envDB              = "MF_AUTH_DB"
	envDBSSLMode       = "MF_AUTH_DB_SSL_MODE"
	envDBSSLCert       = "MF_AUTH_DB_SSL_CERT"
	envDBSSLKey        = "MF_AUTH_DB_SSL_KEY"
	envDBSSLRootCert   = "MF_AUTH_DB_SSL_ROOT_CERT"
	envHTTPPort        = "MF_AUTH_HTTP_PORT"
	envGRPCPort        = "MF_AUTH_GRPC_PORT"
	envTimeout         = "MF_AUTH_GRPC_TIMEOUT"
	envSecret          = "MF_AUTH_SECRET"
	envServerCert      = "MF_AUTH_SERVER_CERT"
	envServerKey       = "MF_AUTH_SERVER_KEY"
	envJaegerURL       = "MF_JAEGER_URL"
	envLoginDuration   = "MF_AUTH_LOGIN_TOKEN_DURATION"
	envInviteDuration  = "MF_INVITE_DURATION"
	envAdminEmail      = "MF_USERS_ADMIN_EMAIL"
	envThingsGRPCURL   = "MF_THINGS_AUTH_GRPC_URL"
	envThingsCACerts   = "MF_THINGS_CA_CERTS"
	envThingsClientTLS = "MF_THINGS_CLIENT_TLS"
	envUsersGRPCURL    = "MF_USERS_GRPC_URL"
	envUsersCACerts    = "MF_USERS_CA_CERTS"
	envUsersClientTLS  = "MF_USERS_CLIENT_TLS"
	envESURL           = "MF_AUTH_ES_URL"

	envEmailHost         = "MF_EMAIL_HOST"
	envEmailPort         = "MF_EMAIL_PORT"
	envEmailUsername     = "MF_EMAIL_USERNAME"
	envEmailPassword     = "MF_EMAIL_PASSWORD"
	envEmailFromAddress  = "MF_EMAIL_FROM_ADDRESS"
	envEmailFromName     = "MF_EMAIL_FROM_NAME"
	envEmailBaseTemplate = "MF_EMAIL_BASE_TEMPLATE"

	envHost = "MF_HOST"
)

type config struct {
	logLevel       string
	dbConfig       postgres.Config
	httpConfig     servers.Config
	grpcConfig     servers.Config
	thingsConfig   clients.Config
	usersConfig    clients.Config
	emailConfig    email.Config
	secret         string
	jaegerURL      string
	loginDuration  time.Duration
	inviteDuration time.Duration
	timeout        time.Duration
	adminEmail     string
	host           string
	esURL          string
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err)
	}

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	authHttpTracer, authHttpCloser := jaeger.Init("auth_http", cfg.jaegerURL, logger)
	defer authHttpCloser.Close()

	authGrpcTracer, authGrpcCloser := jaeger.Init("auth_grpc", cfg.jaegerURL, logger)
	defer authGrpcCloser.Close()

	dbTracer, dbCloser := jaeger.Init("auth_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	esClient := connectToRedis(cfg.esURL, logger)

	usrConn := clientsgrpc.Connect(cfg.usersConfig, logger)
	defer usrConn.Close()

	usersTracer, usersCloser := jaeger.Init("auth_users", cfg.jaegerURL, logger)
	defer usersCloser.Close()

	uc := usersapi.NewClient(usrConn, usersTracer, cfg.timeout)

	thConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer thConn.Close()

	thingsTracer, thingsCloser := jaeger.Init("auth_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	tc := thingsapi.NewClient(thConn, thingsTracer, cfg.timeout)

	svc := newService(db, tc, uc, dbTracer, logger, cfg, esClient)

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(svc, authHttpTracer, logger), cfg.httpConfig, logger)
	})
	g.Go(func() error {
		return serversgrpc.Start(ctx, authGrpcTracer, svc, cfg.grpcConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Authentication service shutdown by signal: %s", sig))
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Authentication service terminated: %s", err))
	}
}

func loadConfig() config {
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

	grpcConfig := servers.Config{
		ServerName:   svcName,
		ServerCert:   mainflux.Env(envServerCert, defServerCert),
		ServerKey:    mainflux.Env(envServerKey, defServerKey),
		Port:         mainflux.Env(envGRPCPort, defGRPCPort),
		StopWaitTime: stopWaitTime,
	}

	usersClientTLS, err := strconv.ParseBool(mainflux.Env(envUsersClientTLS, defUsersClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envUsersClientTLS)
	}

	thingsClientTLS, err := strconv.ParseBool(mainflux.Env(envThingsClientTLS, defThingsClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envThingsClientTLS)
	}

	thingsConfig := clients.Config{
		ClientTLS:  thingsClientTLS,
		CaCerts:    mainflux.Env(envThingsCACerts, defThingsCACerts),
		URL:        mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		ClientName: clients.Things,
	}

	usersConfig := clients.Config{
		ClientTLS:  usersClientTLS,
		CaCerts:    mainflux.Env(envUsersCACerts, defUsersCACerts),
		URL:        mainflux.Env(envUsersGRPCURL, defUsersGRPCURL),
		ClientName: clients.Users,
	}

	emailConfig := email.Config{
		FromAddress:      mainflux.Env(envEmailFromAddress, defEmailFromAddress),
		FromName:         mainflux.Env(envEmailFromName, defEmailFromName),
		Host:             mainflux.Env(envEmailHost, defEmailHost),
		Port:             mainflux.Env(envEmailPort, defEmailPort),
		Username:         mainflux.Env(envEmailUsername, defEmailUsername),
		Password:         mainflux.Env(envEmailPassword, defEmailPassword),
		BaseTemplatePath: mainflux.Env(envEmailBaseTemplate, defEmailBaseTemplate),
	}

	loginDuration, err := time.ParseDuration(mainflux.Env(envLoginDuration, defLoginDuration))
	if err != nil {
		log.Fatal(err)
	}

	timeout, err := time.ParseDuration(mainflux.Env(envTimeout, defTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envTimeout, err.Error())
	}

	inviteDuration, err := time.ParseDuration(mainflux.Env(envInviteDuration, defInviteDuration))
	if err != nil {
		log.Fatal(err)
	}

	return config{
		logLevel:       mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:       dbConfig,
		httpConfig:     httpConfig,
		grpcConfig:     grpcConfig,
		thingsConfig:   thingsConfig,
		usersConfig:    usersConfig,
		emailConfig:    emailConfig,
		secret:         mainflux.Env(envSecret, defSecret),
		jaegerURL:      mainflux.Env(envJaegerURL, defJaegerURL),
		loginDuration:  loginDuration,
		inviteDuration: inviteDuration,
		timeout:        timeout,
		adminEmail:     mainflux.Env(envAdminEmail, defAdminEmail),
		host:           mainflux.Env(envHost, defHost),
		esURL:          mainflux.Env(envESURL, defESURL),
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

func connectToRedis(redisURL string, logger logger.Logger) *redis.Client {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to redis: %s", err))
		os.Exit(1)
	}

	return redis.NewClient(opts)
}

func newService(db *sqlx.DB, tc protomfx.ThingsServiceClient, uc protomfx.UsersServiceClient, tracer opentracing.Tracer, logger logger.Logger, cfg config, esClient *redis.Client) auth.Service {
	orgsRepo := postgres.NewOrgRepo(db)
	orgsRepo = tracing.OrgRepositoryMiddleware(tracer, orgsRepo)

	database := dbutil.NewDatabase(db)
	keysRepo := tracing.New(postgres.New(database), tracer)

	rolesRepo := postgres.NewRolesRepo(db)
	rolesRepo = tracing.RolesRepositoryMiddleware(tracer, rolesRepo)

	membsRepo := postgres.NewOrgMembershipsRepo(db)
	membsRepo = tracing.OrgMembershipsRepositoryMiddleware(tracer, membsRepo)

	invitesRepo := postgres.NewOrgInvitesRepo(db)
	invitesRepo = tracing.InvitesRepositoryMiddleware(tracer, invitesRepo)

	idProvider := uuid.New()
	t := jwt.New(cfg.secret)

	authEmailer, err := emailer.New(cfg.host, &cfg.emailConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to configure e-mailing util: %s", err.Error()))
	}

	authEmailer = emailer.LoggingMiddleware(authEmailer, logger)
	authEmailer = emailer.MetricsMiddleware(
		authEmailer,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "auth",
			Subsystem: "email",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "auth",
			Subsystem: "email",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	svc := auth.New(orgsRepo, tc, uc, keysRepo, rolesRepo, membsRepo, invitesRepo, authEmailer, idProvider, t, cfg.loginDuration, cfg.inviteDuration)
	svc = rediscache.NewEventStoreMiddleware(svc, esClient)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "auth",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "auth",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}
