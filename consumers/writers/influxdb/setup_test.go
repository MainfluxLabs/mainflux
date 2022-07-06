// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package influxdb_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	dockertest "github.com/ory/dockertest/v3"
)

const (
	dbToken       = "test-token"
	dbOrg         = "test-org"
	dbAdmin       = "test-admin"
	dbPass        = "test-password"
	dbBucket      = "test-bucket"
	dbInitMode    = "setup"
	dbFluxEnabled = "true"
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		testLog.Error(fmt.Sprintf("Could not connect to docker: %s", err))
	}

	cfg := []string{
		fmt.Sprintf("DOCKER_INFLUXDB_INIT_MODE=%s", dbInitMode),
		fmt.Sprintf("DOCKER_INFLUXDB_INIT_USERNAME=%s", dbAdmin),
		fmt.Sprintf("DOCKER_INFLUXDB_INIT_PASSWORD=%s", dbPass),
		fmt.Sprintf("DOCKER_INFLUXDB_INIT_ORG=%s", dbOrg),
		fmt.Sprintf("DOCKER_INFLUXDB_INIT_BUCKET=%s", dbBucket),
		fmt.Sprintf("DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=%s", dbToken),
		fmt.Sprintf("INFLUXDB_HTTP_FLUX_ENABLED=%s", dbFluxEnabled),
	}
	container, err := pool.Run("influxdb", "2.2-alpine", cfg)
	if err != nil {
		testLog.Error(fmt.Sprintf("Could not start container: %s", err))
	}

	port := container.GetPort("8086/tcp")
	dbUrl := fmt.Sprintf("http://localhost:%s", port)

	if err := pool.Retry(func() error {
		client = influxdb2.NewClientWithOptions(dbUrl, dbToken, influxdb2.DefaultOptions())
		_, err = client.Ping(context.Background())
		return err
	}); err != nil {
		testLog.Error(fmt.Sprintf("Could not connect to docker: %s", err))
	}

	code := m.Run()

	if err := pool.Purge(container); err != nil {
		testLog.Error(fmt.Sprintf("Could not purge container: %s", err))
	}

	os.Exit(code)
}
