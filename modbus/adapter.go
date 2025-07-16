// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package modbus contains the domain concept definitions needed to support
// Mainflux Modbus adapter service functionality.
package modbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/goburrow/modbus"
)

const pollInterval = 10 * time.Second

type Service interface {
	StartPolling(ctx context.Context) error
}

type adapterService struct {
	pub    messaging.Publisher
	logger logger.Logger
}

func New(pub messaging.Publisher, logger logger.Logger) Service {
	return &adapterService{
		pub:    pub,
		logger: logger,
	}
}

func (svc *adapterService) StartPolling(ctx context.Context) error {
	handler := modbus.NewTCPClientHandler("localhost")
	handler.Timeout = 10 * time.Second
	handler.SlaveId = 1

	err := handler.Connect()
	if err != nil {
		return errors.Wrap(errors.New("Error connecting to Modbus device"), err)
	}
	defer handler.Close()

	client := modbus.NewClient(handler)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.New("Modbus polling stopped")
		case <-ticker.C:
			results, err := client.ReadHoldingRegisters(0, 4)
			if err != nil {
				svc.logger.Error(fmt.Sprintf("Modbus read error: %v", err))
				continue
			}

			payload, err := json.Marshal(map[string]interface{}{
				"ts":   time.Now().Unix(),
				"data": results,
			})
			if err != nil {
				svc.logger.Error(fmt.Sprintf("Failed to serialize message: %v", err))
				continue
			}

			msg := protomfx.Message{
				Subtopic:  "",
				Publisher: "modbus-adapter",
				Protocol:  "modbus",
				Payload:   payload,
			}

			if err := svc.pub.Publish(msg); err != nil {
				return err
			}
		}
	}
}
