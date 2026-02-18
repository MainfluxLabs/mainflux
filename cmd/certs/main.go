// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	authapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/MainfluxLabs/mainflux/certs/api"
	"github.com/MainfluxLabs/mainflux/certs/pki"
	"github.com/MainfluxLabs/mainflux/certs/postgres"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	clientsgrpc "github.com/MainfluxLabs/mainflux/pkg/clients/grpc"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	thingsapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	stopWaitTime = 5 * time.Second
	svcName      = "certs"

	defLogLevel          = "error"
	defDBHost            = "localhost"
	defDBPort            = "5432"
	defDBUser            = "mainflux"
	defDBPass            = "mainflux"
	defDB                = "certs"
	defDBSSLMode         = "disable"
	defDBSSLCert         = ""
	defDBSSLKey          = ""
	defDBSSLRootCert     = ""
	defClientTLS         = "false"
	defCACerts           = ""
	defPort              = "8204"
	defServerCert        = ""
	defServerKey         = ""
	defCertsURL          = "http://localhost"
	defJaegerURL         = ""
	defAuthGRPCURL       = "localhost:8181"
	defAuthGRPCTimeout   = "1s"
	defThingsGRPCURL     = "localhost:8183"
	defThingsGRPCTimeout = "1s"

	defSignCAPath     = "ca.crt"
	defSignCAKeyPath  = "ca.key"
	defSignHoursValid = "2048h"
	defSignRSABits    = ""

	defVaultHost       = ""
	defVaultRole       = "mainflux"
	defVaultToken      = ""
	defVaultPKIIntPath = "pki_int"

	envPort              = "MF_CERTS_HTTP_PORT"
	envLogLevel          = "MF_CERTS_LOG_LEVEL"
	envDBHost            = "MF_CERTS_DB_HOST"
	envDBPort            = "MF_CERTS_DB_PORT"
	envDBUser            = "MF_CERTS_DB_USER"
	envDBPass            = "MF_CERTS_DB_PASS"
	envDB                = "MF_CERTS_DB"
	envDBSSLMode         = "MF_CERTS_DB_SSL_MODE"
	envDBSSLCert         = "MF_CERTS_DB_SSL_CERT"
	envDBSSLKey          = "MF_CERTS_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_CERTS_DB_SSL_ROOT_CERT"
	envClientTLS         = "MF_CERTS_CLIENT_TLS"
	envCACerts           = "MF_CERTS_CA_CERTS"
	envServerCert        = "MF_CERTS_SERVER_CERT"
	envServerKey         = "MF_CERTS_SERVER_KEY"
	envCertsURL          = "MF_SDK_CERTS_URL"
	envJaegerURL         = "MF_JAEGER_URL"
	envAuthGRPCURL       = "MF_AUTH_GRPC_URL"
	envAuthGRPCTimeout   = "MF_AUTH_GRPC_TIMEOUT"
	envThingsGRPCURL     = "MF_THINGS_GRPC_URL"
	envThingsGRPCTimeout = "MF_THINGS_GRPC_TIMEOUT"
	envSignCAPath        = "MF_CERTS_SIGN_CA_PATH"
	envSignCAKey         = "MF_CERTS_SIGN_CA_KEY_PATH"
	envSignHoursValid    = "MF_CERTS_SIGN_HOURS_VALID"
	envSignRSABits       = "MF_CERTS_SIGN_RSA_BITS"

	envVaultHost       = "MF_CERTS_VAULT_HOST"
	envVaultPKIIntPath = "MF_VAULT_PKI_INT_PATH"
	envVaultRole       = "MF_VAULT_CA_ROLE_NAME"
	envVaultToken      = "MF_VAULT_TOKEN"
)

var (
	errFailedCertLoading     = errors.New("failed to load certificate")
	errFailedCertDecode      = errors.New("failed to decode certificate")
	errCACertificateNotExist = errors.New("CA certificate does not exist")
	errCAKeyNotExist         = errors.New("CA certificate key does not exist")
)

type config struct {
	logLevel          string
	dbConfig          postgres.Config
	httpConfig        servers.Config
	authConfig        clients.Config
	thingsConfig      clients.Config
	certsURL          string
	jaegerURL         string
	authGRPCTimeout   time.Duration
	thingsGRPCTimeout time.Duration
	// Sign and issue certificates without 3rd party PKI
	signCAPath     string
	signCAKeyPath  string
	signRSABits    int
	signHoursValid string
	// 3rd party PKI API access settings
	pkiPath  string
	pkiToken string
	pkiHost  string
	pkiRole  string
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err)
	}

	certsHttpTracer, certsHttpCloser := jaeger.Init("certs_http", cfg.jaegerURL, logger)
	defer certsHttpCloser.Close()

	tlsCert, caCert, err := loadCertificates(cfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load CA certificates for issuing client certs: %s", err))
	}

	tlsConfig := configureTLSServer(cfg, caCert)
	cfg.httpConfig.TLSConfig = tlsConfig

	pkiAgent, err := pki.NewAgent(tlsCert)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create PKI agent: %s", err))
	}

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	authTracer, authCloser := jaeger.Init("certs_auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	authConn := clientsgrpc.Connect(cfg.authConfig, logger)
	defer authConn.Close()

	auth := authapi.NewClient(authConn, authTracer, cfg.authGRPCTimeout)

	thingsTracer, thingsCloser := jaeger.Init("certs_things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	thingsConn := clientsgrpc.Connect(cfg.thingsConfig, logger)
	defer thingsConn.Close()

	tc := thingsapi.NewClient(thingsConn, thingsTracer, cfg.thingsGRPCTimeout)

	svc := newService(auth, tc, db, logger, tlsCert, caCert, cfg, pkiAgent)

	g.Go(func() error {
		return servershttp.Start(ctx, api.MakeHandler(svc, certsHttpTracer, pkiAgent, logger), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Certs service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Certs service terminated: %s", err))
	}
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		tls = false
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
		Port:         mainflux.Env(envPort, defPort),
		StopWaitTime: stopWaitTime,
	}

	authGRPCTimeout, err := time.ParseDuration(mainflux.Env(envAuthGRPCTimeout, defAuthGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthGRPCTimeout, err.Error())
	}

	authConfig := clients.Config{
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envCACerts, defCACerts),
		URL:        mainflux.Env(envAuthGRPCURL, defAuthGRPCURL),
		ClientName: clients.Auth,
	}

	thingsGRPCTimeout, err := time.ParseDuration(mainflux.Env(envThingsGRPCTimeout, defThingsGRPCTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsGRPCTimeout, err.Error())
	}

	thingsConfig := clients.Config{
		ClientTLS:  tls,
		CaCerts:    mainflux.Env(envCACerts, defCACerts),
		URL:        mainflux.Env(envThingsGRPCURL, defThingsGRPCURL),
		ClientName: clients.Things,
	}

	signRSABits, err := strconv.Atoi(mainflux.Env(envSignRSABits, defSignRSABits))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envSignRSABits, err.Error())
	}

	return config{
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:          dbConfig,
		httpConfig:        httpConfig,
		authConfig:        authConfig,
		thingsConfig:      thingsConfig,
		certsURL:          mainflux.Env(envCertsURL, defCertsURL),
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		authGRPCTimeout:   authGRPCTimeout,
		thingsGRPCTimeout: thingsGRPCTimeout,

		signCAKeyPath:  mainflux.Env(envSignCAKey, defSignCAKeyPath),
		signCAPath:     mainflux.Env(envSignCAPath, defSignCAPath),
		signHoursValid: mainflux.Env(envSignHoursValid, defSignHoursValid),
		signRSABits:    signRSABits,

		pkiToken: mainflux.Env(envVaultToken, defVaultToken),
		pkiPath:  mainflux.Env(envVaultPKIIntPath, defVaultPKIIntPath),
		pkiRole:  mainflux.Env(envVaultRole, defVaultRole),
		pkiHost:  mainflux.Env(envVaultHost, defVaultHost),
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

func newService(ac protomfx.AuthServiceClient, tc protomfx.ThingsServiceClient, db *sqlx.DB, logger logger.Logger, tlsCert tls.Certificate, x509Cert *x509.Certificate, cfg config, pkiAgent pki.Agent) certs.Service {
	database := dbutil.NewDatabase(db)
	certsRepo := postgres.NewRepository(database)

	certsConfig := certs.Config{
		LogLevel:       cfg.logLevel,
		ClientTLS:      cfg.authConfig.ClientTLS,
		CaCerts:        cfg.authConfig.CaCerts,
		HTTPPort:       cfg.httpConfig.Port,
		ServerCert:     cfg.httpConfig.ServerCert,
		ServerKey:      cfg.httpConfig.ServerKey,
		CertsURL:       cfg.certsURL,
		JaegerURL:      cfg.jaegerURL,
		AuthURL:        cfg.authConfig.URL,
		AuthTimeout:    cfg.authGRPCTimeout,
		SignTLSCert:    tlsCert,
		SignX509Cert:   x509Cert,
		SignHoursValid: cfg.signHoursValid,
		SignRSABits:    cfg.signRSABits,
		PKIToken:       cfg.pkiToken,
		PKIHost:        cfg.pkiHost,
		PKIPath:        cfg.pkiPath,
		PKIRole:        cfg.pkiRole,
	}

	svc := certs.New(ac, tc, certsRepo, certsConfig, pkiAgent)
	svc = api.NewLoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "certs",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "certs",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)
	return svc
}

func loadCertificates(conf config) (tls.Certificate, *x509.Certificate, error) {
	var tlsCert tls.Certificate
	var caCert *x509.Certificate

	if conf.signCAPath == "" || conf.signCAKeyPath == "" {
		return tlsCert, caCert, errors.New("CA certificate paths not configured")
	}

	if _, err := os.Stat(conf.signCAPath); os.IsNotExist(err) {
		return tlsCert, caCert, errCACertificateNotExist
	}

	if _, err := os.Stat(conf.signCAKeyPath); os.IsNotExist(err) {
		return tlsCert, caCert, errCAKeyNotExist
	}

	tlsCert, err := tls.LoadX509KeyPair(conf.signCAPath, conf.signCAKeyPath)
	if err != nil {
		return tlsCert, caCert, errors.Wrap(errFailedCertLoading, err)
	}

	b, err := os.ReadFile(conf.signCAPath)
	if err != nil {
		return tlsCert, caCert, errors.Wrap(errFailedCertLoading, err)
	}

	block, _ := pem.Decode(b)
	if block == nil {
		return tlsCert, caCert, errors.New("no PEM data found, failed to decode CA")
	}

	caCert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return tlsCert, caCert, errors.Wrap(errFailedCertDecode, err)
	}

	return tlsCert, caCert, nil
}

func configureTLSServer(cfg config, caCert *x509.Certificate) *tls.Config {
	if cfg.httpConfig.ServerCert == "" {
		return nil
	}

	if caCert == nil {
		return nil
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert)

	return &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caCertPool,
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		},
	}

}
