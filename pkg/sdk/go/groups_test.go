package sdk_test

import (
	"fmt"
	"net/http"
	"testing"

	sdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	group1 = sdk.Group{
		Name: "test",
	}
	group2 = sdk.Group{
		Name: "test2",
	}
)

func TestDeleteGroups(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id1, err := mainfluxSDK.CreateGroup(group1, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	id2, err := mainfluxSDK.CreateGroup(group2, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	grIDs := []string{id1, id2}

	cases := []struct {
		desc     string
		groupIDs []string
		token    string
		err      error
	}{
		{
			desc:     "delete groups with invalid token",
			groupIDs: grIDs,
			token:    wrongValue,
			err:      createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:     "delete non-existing groups",
			groupIDs: []string{wrongID},
			token:    token,
			err:      createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
		{
			desc:     "delete groups without group ids",
			groupIDs: []string{},
			token:    token,
			err:      createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:     "delete groups with empty group ids",
			groupIDs: []string{""},
			token:    token,
			err:      createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:     "delete groups with empty token",
			groupIDs: grIDs,
			token:    "",
			err:      createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:     "delete existing groups",
			groupIDs: grIDs,
			token:    token,
			err:      nil,
		},
		{
			desc:     "delete deleted groups",
			groupIDs: grIDs,
			token:    token,
			err:      createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.DeleteGroups(tc.groupIDs, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
