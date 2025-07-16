package modbus

import (
	"context"
	"net/http"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/opentracing/opentracing-go"
)

func MakeHandler(svc Service, tracer opentracing.Tracer, logger logger.Logger) http.Handler {
	handler := http.NewServeMux()

	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("modbus-adapter is alive"))
	})

	// Start polling in background using top-level context
	go svc.StartPolling(context.Background())

	return handler
}
