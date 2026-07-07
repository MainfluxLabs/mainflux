// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/MainfluxLabs/mainflux/certs/postgres"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveByThing(t *testing.T) {
	repo := postgres.NewRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	thingID := "11111111-1111-1111-1111-111111111111"
	otherThing := "22222222-2222-2222-2222-222222222222"

	for i := 0; i < 3; i++ {
		_, err := repo.Save(ctx, certs.Cert{
			ThingID:   thingID,
			Serial:    fmt.Sprintf("rbt-a-%d", i),
			ExpiresAt: time.Now().Add(time.Hour),
		})
		require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))
	}
	_, err := repo.Save(ctx, certs.Cert{
		ThingID:   otherThing,
		Serial:    "rbt-b-0",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))

	err = repo.RemoveByThing(ctx, thingID)
	require.Nil(t, err, fmt.Sprintf("unexpected RemoveByThing error: %s", err))

	// The thing's certs are gone.
	page, err := repo.RetrieveByThing(ctx, thingID, certs.PageMetadata{Limit: 10})
	require.Nil(t, err, fmt.Sprintf("unexpected retrieve error: %s", err))
	assert.Equal(t, uint64(0), page.Total, "expected the thing's certs to be removed")

	// They were revoked, not just deleted.
	revoked, err := repo.RetrieveRevokedCerts(ctx)
	require.Nil(t, err, fmt.Sprintf("unexpected retrieve revoked error: %s", err))
	assert.Equal(t, 3, len(revoked), "expected removed certs to be recorded as revoked")

	// Another thing's certs are untouched.
	page, err = repo.RetrieveByThing(ctx, otherThing, certs.PageMetadata{Limit: 10})
	require.Nil(t, err, fmt.Sprintf("unexpected retrieve error: %s", err))
	assert.Equal(t, uint64(1), page.Total, "expected other thing's certs to remain")

	// Removing again is an idempotent no-op (no error, no duplicate revoked rows).
	err = repo.RemoveByThing(ctx, thingID)
	require.Nil(t, err, fmt.Sprintf("unexpected RemoveByThing error on retry: %s", err))
	revoked, err = repo.RetrieveRevokedCerts(ctx)
	require.Nil(t, err, fmt.Sprintf("unexpected retrieve revoked error: %s", err))
	assert.Equal(t, 3, len(revoked), "retry must not create duplicate revoked rows")
}
