// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/internal/email"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	serversgrpc "github.com/MainfluxLabs/mainflux/pkg/servers/grpc"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	httpapi "github.com/MainfluxLabs/mainflux/users/api/http"
	"github.com/MainfluxLabs/mainflux/users/bcrypt"
	"github.com/MainfluxLabs/mainflux/users/emailer"
	"github.com/MainfluxLabs/mainflux/users/postgres"
	"github.com/MainfluxLabs/mainflux/users/tracing"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	stopWaitTime = 5 * time.Second
	svcName      = "users"

	defLogLevel      = "error"
	defDBHost        = "localhost"
	defDBPort        = "5432"
	defDBUser        = "mainflux"
	defDBPass        = "mainflux"
	defDB            = "users"
	defDBSSLMode     = "disable"
	defDBSSLCert     = ""
	defDBSSLKey      = ""
	defDBSSLRootCert = ""
	defHTTPPort      = "8180"
	defServerCert    = ""
	defServerKey     = ""
	defJaegerURL     = ""

	defEmailHost        = "localhost"
	defEmailPort        = "25"
	defEmailUsername    = "root"
	defEmailPassword    = ""
	defEmailFromAddress = ""
	defEmailFromName    = ""
	defEmailTemplate    = "email.tmpl"
	defAdminEmail       = ""
	defAdminPassword    = ""
	defPassRegex        = "^.{8,}$"

	defTokenResetEndpoint = "/reset-request" // URL where user lands after click on the reset link from email

	defAuthTLS         = "false"
	defAuthCACerts     = ""
	defAuthGRPCURL     = "localhost:8181"
	defAuthGRPCTimeout = "1s"
	defGRPCPort        = "8184"

	defSelfRegister = "true" // By default, everybody can create a user. Otherwise, only admin can create a user.

	envLogLevel      = "MF_USERS_LOG_LEVEL"
	envDBHost        = "MF_USERS_DB_HOST"
	envDBPort        = "MF_USERS_DB_PORT"
	envDBUser        = "MF_USERS_DB_USER"
	envDBPass        = "MF_USERS_DB_PASS"
	envDB            = "MF_USERS_DB"
	envDBSSLMode     = "MF_USERS_DB_SSL_MODE"
	envDBSSLCert     = "MF_USERS_DB_SSL_CERT"
	envDBSSLKey      = "MF_USERS_DB_SSL_KEY"
	envDBSSLRootCert = "MF_USERS_DB_SSL_ROOT_CERT"
	envHTTPPort      = "MF_USERS_HTTP_PORT"
	envServerCert    = "MF_USERS_SERVER_CERT"
	envServerKey     = "MF_USERS_SERVER_KEY"
	envJaegerURL     = "MF_JAEGER_URL"

	envAdminEmail    = "MF_USERS_ADMIN_EMAIL"
	envAdminPassword = "MF_USERS_ADMIN_PASSWORD"
	envPassRegex     = "MF_USERS_PASS_REGEX"

	envEmailHost        = "MF_EMAIL_HOST"
	envEmailPort        = "MF_EMAIL_PORT"
	envEmailUsername    = "MF_EMAIL_USERNAME"
	envEmailPassword    = "MF_EMAIL_PASSWORD"
	envEmailFromAddress = "MF_EMAIL_FROM_ADDRESS"
	envEmailFromName    = "MF_EMAIL_FROM_NAME"
	envEmailTemplate    = "MF_EMAIL_TEMPLATE"

	envTokenResetEndpoint = "MF_TOKEN_RESET_ENDPOINT"

	envAuthTLS         = "MF_AUTH_CLIENT_TLS"
	envAuthCACerts     = "MF_AUTH_CA_CERTS"
	envAuthGRPCURL     = "MF_AUTH_GRPC_URL"
	envauthGRPCTimeout = "MF_AUTH_GRPC_TIMEOUT"
	envGRPCPort        = "MF_USERS_GRPC_PORT"

	envSelfRegister = "MF_USERS_ALLOW_SELF_REGISTER"
)

type config struct {
	logLevel        string
	dbConfig        postgres.Config
	httpConfig      servers.Config
	grpcConfig      servers.Config
	emailConf       email.Config
	authConfig      clients.Config
	jaegerURL       string
	resetURL        string
	authGRPCTimeout time.Duration
	adminEmail      string
	adminPassword   string
	passRegex       *regexp.Regexp
	selfRegister    bool
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

	usersHttpTracer, usersHttpCloser := jaeger.Init("users_http", cfg.jaegerURL, logger)
	defer usersHttpCloser.Close()

	usersGrpcTracer, usersGrpcCloser := jaeger.Init("users_grpc", cfg.jaegerURL, logger)
	defer usersGrpcCloser.Close()

	authTracer, closer := jaeger.Init("users_auth", cfg.jaegerURL, logger)
	defer closer.Close()

	authConn := clientsgrpc.Connect(cfg.authConfig, logger)
	defer authConn.Close()

	auth := authapi.NewClient(authConn, authTracer, cfg.authGRPCTimeout)

	dbTracer, dbCloser := jaeger.Init("users_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(db, dbTracer, auth, cfg, logger)

	g.Go(func() error {
		return servershttp.Start(ctx, httpapi.MakeHandler(svc, usersHttpTracer, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Users service shutdown by signal: %s", sig))
		}
		return nil
	})

	g.Go(func() error {
		return serversgrpc.Start(ctx, usersGrpcTracer, svc, cfg.grpcConfig, logger)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Users service terminated: %s", err))
	}
}

func loadConfig() config {
	authGRPCTimeout, err := time.ParseDuration(mainflux.Env(envauthGRPCTimeout, defAuthGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envauthGRPCTimeout, err.Error())
	}

	tls, err := strconv.ParseBool(mainflux.Env(envAuthTLS, defAuthTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envAuthTLS)
	}

	passRegex, err := regexp.Compile(mainflux.Env(envPassRegex, defPassRegex))
	if err != nil {
		log.Fatalf("Invalid password validation rules %s\n", envPassRegex)
	}

	selfRegister, err := strconv.ParseBool(mainflux.Env(envSelfRegister, defSelfRegister))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envSelfRegister, err.Error())
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
		ServerKey:    mainflux.Env(envServerKey, defServerKey),
		ServerCert:   mainflux.Env(envServerCert, defServerCert),
		Port:         mainflux.Env(envHTTPPort, defHTTPPort),
		StopWaitTime: stopWaitTime,
	}

	grpcConfig := servers.Config{
		ServerName:   svcName,
		ServerKey:    mainflux.Env(envServerKey, defServerKey),
		ServerCert:   mainflux.Env(envServerCert, defServerCert),
		Port:         mainflux.Env(envGRPCPort, defGRPCPort),
		StopWaitTime: stopWaitTime,
	}

	emailConf := email.Config{
		FromAddress: mainflux.Env(envEmailFromAddress, defEmailFromAddress),
		FromName:    mainflux.Env(envEmailFromName, defEmailFromName),
		Host:        mainflux.Env(envEmailHost, defEmailHost),
		Port:        mainflux.Env(envEmailPort, defEmailPort),
		Username:    mainflux.Env(envEmailUsername, defEmailUsername),
		Password:    mainflux.Env(envEmailPassword, defEmailPassword),
		Template:    mainflux.Env(envEmailTemplate, defEmailTemplate),
	}

	authConfig := clients.Config{
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envAuthCACerts, defAuthCACerts),
		URL:        mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		ClientName: clients.Auth,
	}

	return config{
		logLevel:        mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:        dbConfig,
		httpConfig:      httpConfig,
		grpcConfig:      grpcConfig,
		emailConf:       emailConf,
		authConfig:      authConfig,
		jaegerURL:       mainflux.Env(envJaegerURL, defJaegerURL),
		resetURL:        mainflux.Env(envTokenResetEndpoint, defTokenResetEndpoint),
		authGRPCTimeout: authGRPCTimeout,
		adminEmail:      mainflux.Env(envAdminEmail, defAdminEmail),
		adminPassword:   mainflux.Env(envAdminPassword, defAdminPassword),
		passRegex:       passRegex,
		selfRegister:    selfRegister,
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

func newService(db *sqlx.DB, tracer opentracing.Tracer, ac protomfx.AuthServiceClient, c config, logger logger.Logger) users.Service {
	database := postgres.NewDatabase(db)
	hasher := bcrypt.New()
	userRepo := tracing.UserRepositoryMiddleware(postgres.NewUserRepo(database), tracer)

	emailer, err := emailer.New(c.resetURL, &c.emailConf)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to configure e-mailing util: %s", err.Error()))
	}

	idProvider := uuid.New()

	svc := users.New(userRepo, hasher, ac, emailer, idProvider, c.passRegex)
	svc = httpapi.LoggingMiddleware(svc, logger)
	svc = httpapi.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "users",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "users",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)
	if err := createAdmin(svc, c); err != nil {
		logger.Error("failed to create root user: " + err.Error())
		os.Exit(1)
	}

	return svc
}

func createAdmin(svc users.Service, c config) error {
	user := users.User{
		Email:    c.adminEmail,
		Password: c.adminPassword,
	}

	if err := svc.RegisterAdmin(context.Background(), user); err != nil {
		return err
	}

	return nil
}
