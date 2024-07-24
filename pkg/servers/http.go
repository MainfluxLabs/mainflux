package servers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
)

type Config struct {
	ServerCert   string
	ServerKey    string
	Port         string
	StopWaitTime time.Duration
}

func StartHTTPServer(ctx context.Context, svcName string, handler http.Handler, cfg Config, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", cfg.Port)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: handler}

	switch {
	case cfg.ServerCert != "" || cfg.ServerKey != "":
		logger.Info(fmt.Sprintf("%s service started using https on port %s with cert %s key %s",
			svcName, cfg.Port, cfg.ServerCert, cfg.ServerKey))
		go func() {
			errCh <- server.ListenAndServeTLS(cfg.ServerCert, cfg.ServerKey)
		}()
	default:
		logger.Info(fmt.Sprintf("%s service started using http on port %s", svcName, cfg.Port))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), cfg.StopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("%s service error occurred during shutdown at %s: %s", svcName, p, err))
			return fmt.Errorf("%s service occurred during shutdown at %s: %w", svcName, p, err)
		}
		logger.Info(fmt.Sprintf("%s service  shutdown of http at %s", svcName, p))
		return nil
	case err := <-errCh:
		return err
	}

}
