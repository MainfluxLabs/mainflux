// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux/internal/email"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/MainfluxLabs/mainflux/users/bcrypt"
	"github.com/MainfluxLabs/mainflux/users/emailer"
	"github.com/MainfluxLabs/mainflux/users/tracing"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/MainfluxLabs/mainflux"
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/logger"
	grpcapi "github.com/MainfluxLabs/mainflux/users/api/grpc"
	httpapi "github.com/MainfluxLabs/mainflux/users/api/http"
	"github.com/MainfluxLabs/mainflux/users/postgres"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
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

	usersHttpTracer, usersHttpCloser := initJaeger("users_http", cfg.jaegerURL, logger)
	defer usersHttpCloser.Close()

	usersGrpcTracer, usersGrpcCloser := initJaeger("users_grpc", cfg.jaegerURL, logger)
	defer usersGrpcCloser.Close()

	authTracer, closer := initJaeger("users_auth", cfg.jaegerURL, logger)
	defer closer.Close()

	authConn := clients.Connect(cfg.authConfig, "auth", logger)
	defer authConn.Close()

	auth := authapi.NewClient(authTracer, authConn, cfg.authGRPCTimeout)

	dbTracer, dbCloser := initJaeger("users_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(db, dbTracer, auth, cfg, logger)

	g.Go(func() error {
		return servers.StartHTTPServer(ctx, svcName, httpapi.MakeHandler(svc, usersHttpTracer, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Users service shutdown by signal: %s", sig))
		}
		return nil
	})

	g.Go(func() error {
		return startGRPCServer(ctx, svc, usersGrpcTracer, cfg, logger)
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
		ServerKey:    mainflux.Env(envServerKey, defServerKey),
		ServerCert:   mainflux.Env(envServerCert, defServerCert),
		Port:         mainflux.Env(envHTTPPort, defHTTPPort),
		StopWaitTime: stopWaitTime,
	}

	grpcConfig := servers.Config{
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
		ClientTLS: tls,
		CaCerts:   mainflux.Env(envAuthCACerts, defAuthCACerts),
		GrpcURL:   mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
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
		logger.Error(fmt.Sprintf("Failed to init Jaeger: %s", err))
		os.Exit(1)
	}

	return tracer, closer
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func newService(db *sqlx.DB, tracer opentracing.Tracer, ac mainflux.AuthServiceClient, c config, logger logger.Logger) users.Service {
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

func startGRPCServer(ctx context.Context, svc users.Service, tracer opentracing.Tracer, cfg config, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", cfg.grpcConfig.Port)
	errCh := make(chan error)
	var server *grpc.Server

	listener, err := net.Listen("tcp", p)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", cfg.grpcConfig.Port, err)
	}

	switch {
	case cfg.grpcConfig.ServerCert != "" || cfg.grpcConfig.ServerKey != "":
		creds, err := credentials.NewServerTLSFromFile(cfg.grpcConfig.ServerCert, cfg.grpcConfig.ServerKey)
		if err != nil {
			return fmt.Errorf("failed to load users certificates: %w", err)
		}
		logger.Info(fmt.Sprintf("Users gRPC service started using https on port %s with cert %s key %s",
			cfg.grpcConfig.Port, cfg.grpcConfig.ServerCert, cfg.grpcConfig.ServerKey))
		server = grpc.NewServer(grpc.Creds(creds))
	default:
		logger.Info(fmt.Sprintf("Users gRPC service started using http on port %s", cfg.grpcConfig.Port))
		server = grpc.NewServer()
	}

	mainflux.RegisterUsersServiceServer(server, grpcapi.NewServer(tracer, svc))
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		c := make(chan bool)
		go func() {
			defer close(c)
			server.GracefulStop()
		}()
		select {
		case <-c:
		case <-time.After(stopWaitTime):
		}
		logger.Info(fmt.Sprintf("Users gRPC service shutdown at %s", p))
		return nil
	case err := <-errCh:
		return err
	}
}
