// modbus-adapter/cmd/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/jaeger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	servershttp "github.com/MainfluxLabs/mainflux/pkg/servers/http"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	stopWaitTime = 5 * time.Second
	svcName      = "modbus-adapter"

	defLogLevel  = "error"
	defPort      = "8180"
	defBrokerURL = "nats://localhost:4222"
	defJaegerURL = ""

	envLogLevel  = "MF_MODBUS_ADAPTER_LOG_LEVEL"
	envPort      = "MF_MODBUS_ADAPTER_PORT"
	envBrokerURL = "MF_BROKER_URL"
	envJaegerURL = "MF_JAEGER_URL"
)

type config struct {
	httpConfig servers.Config
	brokerURL  string
	logLevel   string
	jaegerURL  string
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	tracer, closer := jaeger.Init(svcName, cfg.jaegerURL, logger)
	defer closer.Close()

	pub, err := brokers.NewPublisher(cfg.brokerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		os.Exit(1)
	}
	defer pub.Close()

	svc := modbus.New(pub, logger)
	svc = modbus.LoggingMiddleware(svc, logger)
	svc = modbus.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "modbus_adapter",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "modbus_adapter",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	handler := modbus.MakeHandler(svc, tracer, logger)

	g.Go(func() error {
		return servershttp.Start(ctx, handler, cfg.httpConfig, logger)
	})
	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Modbus adapter service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Modbus adapter service terminated: %s", err))
	}
}

func loadConfig() config {
	httpConfig := servers.Config{
		ServerName:   svcName,
		Port:         mainflux.Env(envPort, defPort),
		StopWaitTime: stopWaitTime,
	}

	return config{
		httpConfig: httpConfig,
		brokerURL:  mainflux.Env(envBrokerURL, defBrokerURL),
		logLevel:   mainflux.Env(envLogLevel, defLogLevel),
		jaegerURL:  mainflux.Env(envJaegerURL, defJaegerURL),
	}
}
