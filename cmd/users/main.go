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
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/MainfluxLabs/mainflux/users/api"
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

	defEmailHost         = "localhost"
	defEmailPort         = "25"
	defEmailUsername     = "root"
	defEmailPassword     = ""
	defEmailFromAddress  = ""
	defEmailFromName     = ""
	defEmailBaseTemplate = "base.tmpl"
	defAdminEmail        = ""
	defAdminPassword     = ""
	defPassRegex         = `^\S{8,}$`

	defHost = "http://localhost"

	defInviteDuration = "168h"

	defAuthTLS         = "false"
	defAuthCACerts     = ""
	defAuthGRPCURL     = "localhost:8181"
	defAuthGRPCTimeout = "1s"
	defGRPCPort        = "8184"

	defSelfRegisterEnabled = "true" // By default, everybody can create a user. Otherwise, only admin can create a user.
	defEmailVerifyEnabled  = "false"

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

	envEmailHost         = "MF_EMAIL_HOST"
	envEmailPort         = "MF_EMAIL_PORT"
	envEmailUsername     = "MF_EMAIL_USERNAME"
	envEmailPassword     = "MF_EMAIL_PASSWORD"
	envEmailFromAddress  = "MF_EMAIL_FROM_ADDRESS"
	envEmailFromName     = "MF_EMAIL_FROM_NAME"
	envEmailBaseTemplate = "MF_EMAIL_BASE_TEMPLATE"

	envHost = "MF_HOST"

	envAuthTLS         = "MF_AUTH_CLIENT_TLS"
	envAuthCACerts     = "MF_AUTH_CA_CERTS"
	envAuthGRPCURL     = "MF_AUTH_GRPC_URL"
	envauthGRPCTimeout = "MF_AUTH_GRPC_TIMEOUT"
	envGRPCPort        = "MF_USERS_GRPC_PORT"

	envSelfRegisterEnabled = "MF_USERS_SELF_REGISTER_ENABLED"
	envEmailVerifyEnabled  = "MF_REQUIRE_EMAIL_VERIFICATION"

	envInviteDuration = "MF_INVITE_DURATION"
)

type config struct {
	logLevel            string
	dbConfig            postgres.Config
	httpConfig          servers.Config
	grpcConfig          servers.Config
	emailConf           email.Config
	authConfig          clients.Config
	jaegerURL           string
	authGRPCTimeout     time.Duration
	adminEmail          string
	adminPassword       string
	passRegex           *regexp.Regexp
	host                string
	selfRegisterEnabled bool
	emailVerifyEnabled  bool
	inviteDuration      time.Duration
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
		return servershttp.Start(ctx, httpapi.MakeHandler(svc, usersHttpTracer, logger, cfg.passRegex), cfg.httpConfig, logger)
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

	selfRegisterEnabled, err := strconv.ParseBool(mainflux.Env(envSelfRegisterEnabled, defSelfRegisterEnabled))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envSelfRegisterEnabled, err.Error())
	}

	emailVerifyEnabled, err := strconv.ParseBool(mainflux.Env(envEmailVerifyEnabled, defEmailVerifyEnabled))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envEmailVerifyEnabled, err.Error())
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
		FromAddress:      mainflux.Env(envEmailFromAddress, defEmailFromAddress),
		FromName:         mainflux.Env(envEmailFromName, defEmailFromName),
		Host:             mainflux.Env(envEmailHost, defEmailHost),
		Port:             mainflux.Env(envEmailPort, defEmailPort),
		Username:         mainflux.Env(envEmailUsername, defEmailUsername),
		Password:         mainflux.Env(envEmailPassword, defEmailPassword),
		BaseTemplatePath: mainflux.Env(envEmailBaseTemplate, defEmailBaseTemplate),
	}

	authConfig := clients.Config{
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envAuthCACerts, defAuthCACerts),
		URL:        mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		ClientName: clients.Auth,
	}

	inviteDuration, err := time.ParseDuration(mainflux.Env(envInviteDuration, defInviteDuration))
	if err != nil {
		log.Fatal(err)
	}

	return config{
		logLevel:            mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:            dbConfig,
		httpConfig:          httpConfig,
		grpcConfig:          grpcConfig,
		emailConf:           emailConf,
		authConfig:          authConfig,
		jaegerURL:           mainflux.Env(envJaegerURL, defJaegerURL),
		authGRPCTimeout:     authGRPCTimeout,
		adminEmail:          mainflux.Env(envAdminEmail, defAdminEmail),
		adminPassword:       mainflux.Env(envAdminPassword, defAdminPassword),
		host:                mainflux.Env(envHost, defHost),
		passRegex:           passRegex,
		selfRegisterEnabled: selfRegisterEnabled,
		emailVerifyEnabled:  emailVerifyEnabled,
		inviteDuration:      inviteDuration,
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
	database := dbutil.NewDatabase(db)
	hasher := bcrypt.New()
	userRepo := tracing.UserRepositoryMiddleware(postgres.NewUserRepo(database), tracer)
	verificationRepo := tracing.VerificationRepositoryMiddleware(postgres.NewEmailVerificationRepo(database), tracer)
	platformInvitesRepo := tracing.PlatformInvitesRepositoryMiddleware(postgres.NewPlatformInvitesRepo(database), tracer)

	svcEmailer, err := emailer.New(c.host, &c.emailConf)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to configure e-mailing util: %s", err.Error()))
	}

	svcEmailer = emailer.LoggingMiddleware(svcEmailer, logger)
	svcEmailer = emailer.MetricsMiddleware(
		svcEmailer,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "users",
			Subsystem: "email",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "users",
			Subsystem: "email",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	idProvider := uuid.New()

	svc := users.New(userRepo, verificationRepo, platformInvitesRepo, c.inviteDuration, c.emailVerifyEnabled, c.selfRegisterEnabled, hasher, ac, svcEmailer, idProvider)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
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
