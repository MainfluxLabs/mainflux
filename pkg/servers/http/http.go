// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
)

func Start(ctx context.Context, handler http.Handler, cfg servers.Config, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", cfg.Port)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: handler}

	switch {
	case cfg.ServerCert != "" || cfg.ServerKey != "":
		logger.Info(fmt.Sprintf("%s service started using https on port %s with cert %s key %s",
			cfg.ServerName, cfg.Port, cfg.ServerCert, cfg.ServerKey))
		go func() {
			errCh <- server.ListenAndServeTLS(cfg.ServerCert, cfg.ServerKey)
		}()
	default:
		logger.Info(fmt.Sprintf("%s service started using http on port %s", cfg.ServerName, cfg.Port))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), cfg.StopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("%s service error occurred during shutdown at %s: %s", cfg.ServerName, p, err))
			return fmt.Errorf("%s service occurred during shutdown at %s: %w", cfg.ServerName, p, err)
		}
		logger.Info(fmt.Sprintf("%s service  shutdown of http at %s", cfg.ServerName, p))
		return nil
	case err := <-errCh:
		return err
	}

}
