// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/consumers/writers/api"
	"github.com/MainfluxLabs/mainflux/consumers/writers/influxdb"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "influxdb-writer"
	stopWaitTime = 5 * time.Second

	defBrokerURL = "nats://localhost:4222"
	defLogLevel  = "error"
	defPort      = "8180"
	defDBHost    = "localhost"
	defDBPort    = "8086"
	defDBUser    = "mainflux"
	defDBPass    = "mainflux"
	defDBBucket  = "mainflux-bucket"
	defDBOrg     = "mainflux"
	defDBToken   = "mainflux-token"

	envBrokerURL = "MF_BROKER_URL"
	envLogLevel  = "MF_INFLUX_WRITER_LOG_LEVEL"
	envPort      = "MF_INFLUX_WRITER_PORT"
	envDBHost    = "MF_INFLUXDB_HOST"
	envDBPort    = "MF_INFLUXDB_PORT"
	envDBUser    = "MF_INFLUXDB_ADMIN_USER"
	envDBPass    = "MF_INFLUXDB_ADMIN_PASSWORD"
	envDBBucket  = "MF_INFLUXDB_BUCKET"
	envDBOrg     = "MF_INFLUXDB_ORG"
	envDBToken   = "MF_INFLUXDB_TOKEN"
)

type config struct {
	httpConfig servers.Config
	brokerURL  string
	logLevel   string
	dbHost     string
	dbPort     string
	dbUser     string
	dbPass     string
	dbBucket   string
	dbOrg      string
	dbToken    string
	dbUrl      string
}

func main() {
	cfg, repoCfg := loadConfigs()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	pubSub, err := brokers.NewPubSub(cfg.brokerURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	client, err := connectToInfluxDB(cfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create InfluxDB client: %s", err))
		os.Exit(1)
	}
	defer client.Close()

	repo := influxdb.New(client, repoCfg)
	counter, latency := makeMetrics()
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(repo, counter, latency)

	if err := consumers.Start(svcName, pubSub, repo, brokers.SubjectSenML, brokers.SubjectJSON); err != nil {
		logger.Error(fmt.Sprintf("Failed to start InfluxDB writer: %s", err))
		os.Exit(1)
	}

	g.Go(func() error {
		return servershttp.Start(ctx, api.MakeHandler(svcName), cfg.httpConfig, logger)
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
	client := influxdb2.NewClient(cfg.dbUrl, cfg.dbToken)
	_, err := client.Ping(context.Background())
	return client, err
}

func loadConfigs() (config, influxdb.RepoConfig) {
	httpConfig := servers.Config{
		ServerName:   svcName,
		Port:         mainflux.Env(envPort, defPort),
		StopWaitTime: stopWaitTime,
	}

	cfg := config{
		httpConfig: httpConfig,
		brokerURL:  mainflux.Env(envBrokerURL, defBrokerURL),
		logLevel:   mainflux.Env(envLogLevel, defLogLevel),
		dbHost:     mainflux.Env(envDBHost, defDBHost),
		dbPort:     mainflux.Env(envDBPort, defDBPort),
		dbUser:     mainflux.Env(envDBUser, defDBUser),
		dbPass:     mainflux.Env(envDBPass, defDBPass),
		dbBucket:   mainflux.Env(envDBBucket, defDBBucket),
		dbOrg:      mainflux.Env(envDBOrg, defDBOrg),
		dbToken:    mainflux.Env(envDBToken, defDBToken),
	}
	cfg.dbUrl = fmt.Sprintf("http://%s:%s", cfg.dbHost, cfg.dbPort)

	repoCfg := influxdb.RepoConfig{
		Bucket: cfg.dbBucket,
		Org:    cfg.dbOrg,
	}
	return cfg, repoCfg
}

func makeMetrics() (*kitprometheus.Counter, *kitprometheus.Summary) {
	counter := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "influxdb",
		Subsystem: "message_writer",
		Name:      "request_count",
		Help:      "Number of database inserts.",
	}, []string{"method"})

	latency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "influxdb",
		Subsystem: "message_writer",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of inserts in microseconds.",
	}, []string{"method"})

	return counter, latency
}
