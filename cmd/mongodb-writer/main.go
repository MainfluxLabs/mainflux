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
	"github.com/MainfluxLabs/mainflux/consumers/writers/mongodb"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "mongodb-writer"
	stopWaitTime = 5 * time.Second

	defLogLevel  = "error"
	defBrokerURL = "nats://localhost:4222"
	defPort      = "8180"
	defDB        = "mainflux"
	defDBHost    = "localhost"
	defDBPort    = "27017"

	envBrokerURL = "MF_BROKER_URL"
	envLogLevel  = "MF_MONGO_WRITER_LOG_LEVEL"
	envPort      = "MF_MONGO_WRITER_PORT"
	envDB        = "MF_MONGO_WRITER_DB"
	envDBHost    = "MF_MONGO_WRITER_DB_HOST"
	envDBPort    = "MF_MONGO_WRITER_DB_PORT"
)

type config struct {
	httpConfig servers.Config
	brokerURL  string
	logLevel   string
	dbName     string
	dbHost     string
	dbPort     string
}

func main() {
	cfg := loadConfigs()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err)
	}

	pubSub, err := brokers.NewPubSub(cfg.brokerURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	addr := fmt.Sprintf("mongodb://%s:%s", cfg.dbHost, cfg.dbPort)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to database: %s", err))
		os.Exit(1)
	}

	db := client.Database(cfg.dbName)
	repo := mongodb.New(db)

	counter, latency := makeMetrics()
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(repo, counter, latency)

	if err := consumers.Start(svcName, pubSub, repo, brokers.SubjectSenML, brokers.SubjectJSON); err != nil {
		logger.Error(fmt.Sprintf("Failed to start MongoDB writer: %s", err))
		os.Exit(1)
	}

	g.Go(func() error {
		return servershttp.Start(ctx, api.MakeHandler(svcName), cfg.httpConfig, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("MongoDB reader service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("MongoDB writer service terminated: %s", err))
	}

}

func loadConfigs() config {
	httpConfig := servers.Config{
		ServerName:   svcName,
		Port:         mainflux.Env(envPort, defPort),
		StopWaitTime: stopWaitTime,
	}

	return config{
		httpConfig: httpConfig,
		brokerURL:  mainflux.Env(envBrokerURL, defBrokerURL),
		logLevel:   mainflux.Env(envLogLevel, defLogLevel),
		dbName:     mainflux.Env(envDB, defDB),
		dbHost:     mainflux.Env(envDBHost, defDBHost),
		dbPort:     mainflux.Env(envDBPort, defDBPort),
	}
}

func makeMetrics() (*kitprometheus.Counter, *kitprometheus.Summary) {
	counter := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "mongodb",
		Subsystem: "message_writer",
		Name:      "request_count",
		Help:      "Number of database inserts.",
	}, []string{"method"})

	latency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "mongodb",
		Subsystem: "message_writer",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of inserts in microseconds.",
	}, []string{"method"})

	return counter, latency
}
