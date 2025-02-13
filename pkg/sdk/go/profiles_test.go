// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
)

const (
	name    = "name"
	groupID = "371106m2-131g-5286-2mc1-540295c29f95"
)

var (
	pr1      = sdk.Profile{Name: "test1"}
	pr2      = sdk.Profile{ID: "fe6b4e92-cc98-425e-b0aa-000000000001", Name: "test1"}
	pr3      = sdk.Profile{ID: "fe6b4e92-cc98-425e-b0aa-000000000002", Name: "test2"}
	prPrefix = "fe6b4e92-cc98-425e-b0aa-"
)

func TestCreateProfile(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()

	prWrongExtID := sdk.Profile{GroupID: groupID, ID: "b0aa-000000000001", Name: "1", Metadata: metadata}

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		profile sdk.Profile
		token   string
		groupID string
		err     error
		empty   bool
	}{
		{
			desc:    "create new profile",
			profile: pr1,
			token:   token,
			groupID: grID,
			err:     nil,
			empty:   false,
		},
		{
			desc:    "create new profile with empty token",
			profile: pr1,
			token:   emptyValue,
			groupID: grID,
			err:     createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			empty:   true,
		},
		{
			desc:    "create new profile with invalid token",
			profile: pr1,
			token:   wrongValue,
			groupID: grID,
			err:     createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			empty:   true,
		},
		{
			desc:    "create a new profile with external UUID",
			profile: pr2,
			token:   token,
			groupID: grID,
			err:     nil,
			empty:   false,
		},
		{
			desc:    "create a new profile with wrong external UUID",
			profile: prWrongExtID,
			token:   token,
			groupID: grID,
			err:     createError(sdk.ErrFailedCreation, http.StatusBadRequest),
			empty:   true,
		},
	}

	for _, tc := range cases {
		loc, err := mainfluxSDK.CreateProfile(tc.profile, grID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.empty, loc == emptyValue, fmt.Sprintf("%s: expected empty result location, got: %s", tc.desc, loc))
	}
}

func TestCreateProfiles(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	profiles := []sdk.Profile{
		pr2,
		pr3,
	}

	cases := []struct {
		desc     string
		profiles []sdk.Profile
		token    string
		err      error
		res      []sdk.Profile
	}{
		{
			desc:     "create new profiles",
			profiles: profiles,
			token:    token,
			err:      nil,
			res:      profiles,
		},
		{
			desc:     "create new profiles with empty profiles",
			profiles: []sdk.Profile{},
			token:    token,
			err:      createError(sdk.ErrFailedCreation, http.StatusBadRequest),
			res:      []sdk.Profile{},
		},
		{
			desc:     "create new profiles with empty token",
			profiles: profiles,
			token:    emptyValue,
			err:      createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			res:      []sdk.Profile{},
		},
		{
			desc:     "create new profiles with invalid token",
			profiles: profiles,
			token:    wrongValue,
			err:      createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			res:      []sdk.Profile{},
		},
	}
	for _, tc := range cases {
		res, err := mainfluxSDK.CreateProfiles(tc.profiles, grID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))

		for idx := range tc.res {
			assert.Equal(t, tc.res[idx].ID, res[idx].ID, fmt.Sprintf("%s: expected response ID %s got %s", tc.desc, tc.res[idx].ID, res[idx].ID))
		}
	}
}

func TestProfile(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	id, err := mainfluxSDK.CreateProfile(pr2, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr2.GroupID = grID

	cases := []struct {
		desc      string
		profileID string
		token     string
		err       error
		response  sdk.Profile
	}{
		{
			desc:      "get existing profile",
			profileID: id,
			token:     token,
			err:       nil,
			response:  pr2,
		},
		{
			desc:      "get non-existent profile",
			profileID: "43",
			token:     token,
			err:       createError(sdk.ErrFailedFetch, http.StatusNotFound),
			response:  sdk.Profile{},
		},
		{
			desc:      "get profile with invalid token",
			profileID: id,
			token:     emptyValue,
			err:       createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response:  sdk.Profile{},
		},
	}

	for _, tc := range cases {
		respPr, err := mainfluxSDK.Profile(tc.profileID, tc.token)

		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, respPr, fmt.Sprintf("%s: expected response profile %s, got %s", tc.desc, tc.response, respPr))
	}
}

func TestProfiles(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}
	var profiles []sdk.Profile

	mainfluxSDK := sdk.NewSDK(sdkConf)
	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	for i := 1; i < 101; i++ {
		id := fmt.Sprintf("%s%012d", prPrefix, i)
		name := fmt.Sprintf("test-%d", i)
		pr := sdk.Profile{GroupID: grID, ID: id, Name: name}
		_, err := mainfluxSDK.CreateProfile(pr, pr.GroupID, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		profiles = append(profiles, pr)
	}

	cases := []struct {
		desc     string
		token    string
		offset   uint64
		limit    uint64
		name     string
		err      error
		response []sdk.Profile
		metadata map[string]interface{}
	}{
		{
			desc:     "get a list of profiles",
			token:    token,
			offset:   offset,
			limit:    limit,
			err:      nil,
			response: profiles[0:limit],
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of profiles with invalid token",
			token:    wrongValue,
			offset:   offset,
			limit:    limit,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of profiles with empty token",
			token:    emptyValue,
			offset:   offset,
			limit:    limit,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of profiles without limit, default 10",
			token:    token,
			offset:   offset,
			limit:    0,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of profiles with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of profiles with offset greater than max",
			token:    token,
			offset:   110,
			limit:    limit,
			err:      nil,
			response: []sdk.Profile{},
			metadata: make(map[string]interface{}),
		},
	}
	for _, tc := range cases {
		filter := sdk.PageMetadata{
			Name:     tc.name,
			Total:    total,
			Offset:   tc.offset,
			Limit:    tc.limit,
			Metadata: tc.metadata,
		}

		page, err := mainfluxSDK.Profiles(tc.token, filter)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Profiles, fmt.Sprintf("%s: expected response profile %s, got %s", tc.desc, tc.response, page.Profiles))
	}
}

func TestViewProfileByThing(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	pr := sdk.Profile{Name: name}
	pid, err := mainfluxSDK.CreateProfile(pr, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	th := sdk.Thing{Name: name, ProfileID: pid}
	tid, err := mainfluxSDK.CreateThing(th, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	resProfile := sdk.Profile{
		ID:      pid,
		GroupID: grID,
		Name:    name,
	}

	cases := []struct {
		desc     string
		thing    string
		token    string
		err      error
		response sdk.Profile
	}{
		{
			desc:     "view profile by thing",
			thing:    tid,
			token:    token,
			err:      nil,
			response: resProfile,
		},
		{
			desc:     "view profile by thing with invalid token",
			thing:    tid,
			token:    wrongValue,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: sdk.Profile{},
		},
		{
			desc:     "view profile by thing with empty token",
			thing:    tid,
			token:    emptyValue,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: sdk.Profile{},
		},
		{
			desc:     "view profile by thing with empty thing id",
			thing:    emptyValue,
			token:    token,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: sdk.Profile{},
		},
		{
			desc:     "view profile by thing with unknown thing id",
			thing:    wrongID,
			token:    token,
			err:      createError(sdk.ErrFailedFetch, http.StatusNotFound),
			response: sdk.Profile{},
		},
	}

	for _, tc := range cases {
		pr, err := mainfluxSDK.ViewProfileByThing(tc.thing, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, pr, fmt.Sprintf("%s: expected response profile %s, got %s", tc.desc, tc.response, pr))
	}
}

func TestUpdateProfile(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	id, err := mainfluxSDK.CreateProfile(pr2, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	pr2.GroupID = grID

	cases := []struct {
		desc    string
		profile sdk.Profile
		token   string
		err     error
	}{
		{
			desc:    "update existing profile",
			profile: sdk.Profile{ID: id, Name: "test2"},
			token:   token,
			err:     nil,
		},
		{
			desc:    "update non-existing profile",
			profile: sdk.Profile{ID: "0", Name: "test2"},
			token:   token,
			err:     createError(sdk.ErrFailedUpdate, http.StatusNotFound),
		},
		{
			desc:    "update profile with invalid id",
			profile: sdk.Profile{ID: emptyValue, Name: "test2"},
			token:   token,
			err:     createError(sdk.ErrFailedUpdate, http.StatusBadRequest),
		},
		{
			desc:    "update profile with invalid token",
			profile: sdk.Profile{ID: id, Name: "test2"},
			token:   wrongValue,
			err:     createError(sdk.ErrFailedUpdate, http.StatusUnauthorized),
		},
		{
			desc:    "update profile with empty token",
			profile: sdk.Profile{ID: id, Name: "test2"},
			token:   emptyValue,
			err:     createError(sdk.ErrFailedUpdate, http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.UpdateProfile(tc.profile, tc.profile.ID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestDeleteProfile(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	id, err := mainfluxSDK.CreateProfile(pr2, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc      string
		profileID string
		token     string
		err       error
	}{
		{
			desc:      "delete profile with invalid token",
			profileID: id,
			token:     wrongValue,
			err:       createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:      "delete non-existing profile",
			profileID: "2",
			token:     token,
			err:       createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
		{
			desc:      "delete profile with invalid id",
			profileID: emptyValue,
			token:     token,
			err:       createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:      "delete profile with empty token",
			profileID: id,
			token:     emptyValue,
			err:       createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:      "delete existing profile",
			profileID: id,
			token:     token,
			err:       nil,
		},
		{
			desc:      "delete deleted profile",
			profileID: id,
			token:     token,
			err:       createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.DeleteProfile(tc.profileID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestDeleteProfiles(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	id1, err := mainfluxSDK.CreateProfile(pr1, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	id2, err := mainfluxSDK.CreateProfile(pr2, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	prIDs := []string{id1, id2}

	cases := []struct {
		desc       string
		profileIDs []string
		token      string
		err        error
	}{
		{
			desc:       "delete profiles with invalid token",
			profileIDs: prIDs,
			token:      wrongValue,
			err:        createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:       "delete non-existing profiles",
			profileIDs: []string{wrongValue},
			token:      token,
			err:        createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
		{
			desc:       "delete profiles with empty id",
			profileIDs: []string{emptyValue},
			token:      token,
			err:        createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:       "delete profiles without profile ids",
			profileIDs: []string{},
			token:      token,
			err:        createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:       "delete profiles with empty token",
			profileIDs: prIDs,
			token:      emptyValue,
			err:        createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:       "delete profiles with invalid token",
			profileIDs: prIDs,
			token:      wrongValue,
			err:        createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:       "delete existing profiles",
			profileIDs: prIDs,
			token:      token,
			err:        nil,
		},
		{
			desc:       "delete deleted profiles",
			profileIDs: prIDs,
			token:      token,
			err:        createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.DeleteProfiles(tc.profileIDs, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
