// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/stretchr/testify/assert"
)

func TestRefreshCRLWithoutPKIAgent(t *testing.T) {
	crlPath := filepath.Join(t.TempDir(), "crl.pem")
	cs := NewCertScheduler(nil, nil, nil, nil, crlPath, logger.NewMock())

	assert.NotPanics(t, func() { cs.refreshCRL(context.Background()) },
		"refreshing the CRL without a PKI agent must not panic")

	_, err := os.Stat(crlPath)
	assert.True(t, os.IsNotExist(err), "expected no CRL file without a PKI agent")
}
