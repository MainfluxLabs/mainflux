package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/auth"
	api "github.com/MainfluxLabs/mainflux/auth/api"
	grpcapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	httpapi "github.com/MainfluxLabs/mainflux/auth/api/http"
	"github.com/MainfluxLabs/mainflux/auth/jwt"
	"github.com/MainfluxLabs/mainflux/auth/postgres"
	"github.com/MainfluxLabs/mainflux/auth/tracing"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	usersapi "github.com/MainfluxLabs/mainflux/users/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	stopWaitTime = 5 * time.Second

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
	defAdminEmail      = ""
	defTimeout         = "1s"
	defThingsGRPCURL   = "localhost:8183"
	defThingsCACerts   = ""
	defThingsClientTLS = "false"
	defUsersCACerts    = ""
	defUsersClientTLS  = "false"
	defUsersGRPCURL    = "localhost:8184"

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
	envAdminEmail      = "MF_USERS_ADMIN_EMAIL"
	envThingsGRPCURL   = "MF_THINGS_AUTH_GRPC_URL"
	envThingsCACerts   = "MF_THINGS_CA_CERTS"
	envThingsClientTLS = "MF_THINGS_CLIENT_TLS"
	envUsersGRPCURL    = "MF_USERS_GRPC_URL"
	envUsersCACerts    = "MF_USERS_CA_CERTS"
	envUsersClientTLS  = "MF_USERS_CLIENT_TLS"
)

type config struct {
	logLevel        string
	dbConfig        postgres.Config
	authHTTPServer  servers.Config
	authGRPCServer  servers.Config
	secret          string
	jaegerURL       string
	loginDuration   time.Duration
	timeout         time.Duration
	adminEmail      string
	thingsClientTLS bool
	thingsCACerts   string
	thingsGRPCURL   string
	usersClientTLS  bool
	usersCACerts    string
	usersGRPCURL    string
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

	tracer, closer := initJaeger("auth", cfg.jaegerURL, logger)
	defer closer.Close()

	dbTracer, dbCloser := initJaeger("auth_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	usrConn := connectToUsers(cfg, logger)
	defer usrConn.Close()

	usersTracer, usersCloser := initJaeger("users", cfg.jaegerURL, logger)
	defer usersCloser.Close()

	uc := usersapi.NewClient(usrConn, usersTracer, cfg.timeout)

	thConn := connectToThings(cfg, logger)
	defer thConn.Close()

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	tc := thingsapi.NewClient(thConn, thingsTracer, cfg.timeout)

	svc := newService(db, tc, uc, dbTracer, cfg.secret, logger, cfg.loginDuration)

	g.Go(func() error {
		return servers.StartHTTPServer(ctx, "auth-http", httpapi.MakeHandler(svc, tracer, logger), cfg.authHTTPServer, logger)
	})
	g.Go(func() error {
		return startGRPCServer(ctx, tracer, svc, cfg.authGRPCServer.Port, cfg.authGRPCServer.ServerCert, cfg.authGRPCServer.ServerKey, logger)
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

	authHTTPServer := servers.Config{
		ServerCert:   mainflux.Env(envServerCert, defServerCert),
		ServerKey:    mainflux.Env(envServerKey, defServerKey),
		Port:         mainflux.Env(envHTTPPort, defHTTPPort),
		StopWaitTime: stopWaitTime,
	}

	authGRPCServer := servers.Config{
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

	loginDuration, err := time.ParseDuration(mainflux.Env(envLoginDuration, defLoginDuration))
	if err != nil {
		log.Fatal(err)
	}

	timeout, err := time.ParseDuration(mainflux.Env(envTimeout, defTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envTimeout, err.Error())
	}

	return config{
		logLevel:        mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:        dbConfig,
		authHTTPServer:  authHTTPServer,
		authGRPCServer:  authGRPCServer,
		secret:          mainflux.Env(envSecret, defSecret),
		jaegerURL:       mainflux.Env(envJaegerURL, defJaegerURL),
		loginDuration:   loginDuration,
		timeout:         timeout,
		adminEmail:      mainflux.Env(envAdminEmail, defAdminEmail),
		thingsClientTLS: thingsClientTLS,
		thingsCACerts:   mainflux.Env(envThingsCACerts, defThingsCACerts),
		thingsGRPCURL:   mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		usersClientTLS:  usersClientTLS,
		usersCACerts:    mainflux.Env(envUsersCACerts, defUsersCACerts),
		usersGRPCURL:    mainflux.Env(envUsersGRPCURL, defUsersGRPCURL),
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

func connectToUsers(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.usersClientTLS {
		if cfg.usersCACerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.usersCACerts, "")
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

	conn, err := grpc.Dial(cfg.usersGRPCURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to users service: %s", err))
		os.Exit(1)
	}

	return conn
}

func connectToThings(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.thingsClientTLS {
		if cfg.thingsCACerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.thingsCACerts, "")
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

func newService(db *sqlx.DB, tc mainflux.ThingsServiceClient, uc mainflux.UsersServiceClient, tracer opentracing.Tracer, secret string, logger logger.Logger, duration time.Duration) auth.Service {
	orgsRepo := postgres.NewOrgRepo(db)
	orgsRepo = tracing.OrgRepositoryMiddleware(tracer, orgsRepo)

	database := postgres.NewDatabase(db)
	keysRepo := tracing.New(postgres.New(database), tracer)

	rolesRepo := postgres.NewRolesRepo(db)
	rolesRepo = tracing.RolesRepositoryMiddleware(tracer, rolesRepo)

	idProvider := uuid.New()
	t := jwt.New(secret)

	svc := auth.New(orgsRepo, tc, uc, keysRepo, rolesRepo, idProvider, t, duration)
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

func startGRPCServer(ctx context.Context, tracer opentracing.Tracer, svc auth.Service, port string, certFile string, keyFile string, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", port)
	errCh := make(chan error)

	listener, err := net.Listen("tcp", p)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	var server *grpc.Server
	switch {
	case certFile != "" || keyFile != "":
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("failed to load auth certificates: %w", err)
		}
		logger.Info(fmt.Sprintf("Authentication gRPC service started using https on port %s with cert %s key %s", port, certFile, keyFile))
		server = grpc.NewServer(grpc.Creds(creds))
	default:
		logger.Info(fmt.Sprintf("Authentication gRPC service started using http on port %s", port))
		server = grpc.NewServer()
	}

	mainflux.RegisterAuthServiceServer(server, grpcapi.NewServer(tracer, svc))
	logger.Info(fmt.Sprintf("Authentication gRPC service started, exposed port %s", port))
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
		logger.Info(fmt.Sprintf("Authentication gRPC service shutdown at %s", p))
		return nil
	case err := <-errCh:
		return err
	}
}
