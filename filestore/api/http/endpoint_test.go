// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/filestore"
	httpapi "github.com/MainfluxLabs/mainflux/filestore/api/http"
	fsmocks "github.com/MainfluxLabs/mainflux/filestore/mocks"
	"github.com/MainfluxLabs/mainflux/filestore/store"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token         = "tok"
	groupID       = "group-1"
	thingID       = "thing-1"
	thingKey      = "thingkey-1"
	fileName      = "fw.bin"
	fileContent   = "hello mainflux"
	binariesClass = "binaries"
	binFormat     = "bin"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()

	things := map[string]domain.Thing{
		thingKey: {ID: thingID, GroupID: groupID, Key: thingKey},
		thingID:  {ID: thingID, GroupID: groupID, Key: thingKey},
	}
	groups := map[string]domain.Group{
		token: {ID: groupID},
	}
	tc := mocks.NewThingsServiceClient(nil, things, groups)

	svc := filestore.New(tc, fsmocks.NewThingsRepository(), fsmocks.NewGroupsRepository(), store.NewLocal(t.TempDir()), logger.NewMock())

	fi := filestore.FileInfo{Name: fileName, Class: binariesClass, Format: binFormat}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader(fileContent), token, groupID, fi))

	h := httpapi.MakeHandler(mocktracer.New(), svc, mocks.NewAuthService("", nil, nil), logger.NewMock())
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	return srv
}

func TestRangeRequest(t *testing.T) {
	srv := newServer(t)
	size := len(fileContent)

	cases := []struct {
		desc         string
		rangeHeader  string
		status       int
		body         string
		contentRange string
	}{
		{
			desc:        "view file without range",
			rangeHeader: "",
			status:      http.StatusOK,
			body:        fileContent,
		},
		{
			desc:         "view file with bounded range",
			rangeHeader:  "bytes=6-9",
			status:       http.StatusPartialContent,
			body:         "main",
			contentRange: fmt.Sprintf("bytes 6-9/%d", size),
		},
		{
			desc:         "view file with open-ended range",
			rangeHeader:  "bytes=6-",
			status:       http.StatusPartialContent,
			body:         "mainflux",
			contentRange: fmt.Sprintf("bytes 6-13/%d", size),
		},
		{
			desc:         "view file with suffix range",
			rangeHeader:  "bytes=-3",
			status:       http.StatusPartialContent,
			body:         "lux",
			contentRange: fmt.Sprintf("bytes 11-13/%d", size),
		},
		{
			desc:        "view file with multiple ranges",
			rangeHeader: "bytes=0-1,4-5",
			status:      http.StatusOK,
			body:        fileContent,
		},
		{
			desc:        "view file with unsupported range unit",
			rangeHeader: "items=0-1",
			status:      http.StatusOK,
			body:        fileContent,
		},
		{
			desc:        "view file with malformed range",
			rangeHeader: "bytes=5-2",
			status:      http.StatusOK,
			body:        fileContent,
		},
		{
			desc:        "view file with range starting past file size",
			rangeHeader: fmt.Sprintf("bytes=%d-", size),
			status:      http.StatusRequestedRangeNotSatisfiable,
		},
		{
			desc:        "view file with empty suffix range",
			rangeHeader: "bytes=-0",
			status:      http.StatusRequestedRangeNotSatisfiable,
		},
	}

	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/groupfiles/%s", srv.URL, fileName), nil)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		req.Header.Set("Authorization", apiutil.ThingKeyPrefixInternal+thingKey)
		if tc.rangeHeader != "" {
			req.Header.Set("Range", tc.rangeHeader)
		}

		res, err := srv.Client().Do(req)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		require.Nil(t, err, fmt.Sprintf("%s: read failed: %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
		if tc.status == http.StatusRequestedRangeNotSatisfiable {
			continue
		}

		assert.Equal(t, tc.body, string(body), fmt.Sprintf("%s: body mismatch", tc.desc))
		assert.Equal(t, "bytes", res.Header.Get("Accept-Ranges"), fmt.Sprintf("%s: Accept-Ranges mismatch", tc.desc))
		assert.Equal(t, tc.contentRange, res.Header.Get("Content-Range"), fmt.Sprintf("%s: Content-Range mismatch", tc.desc))
		if tc.status == http.StatusPartialContent {
			assert.Equal(t, int64(len(tc.body)), res.ContentLength, fmt.Sprintf("%s: Content-Length mismatch", tc.desc))
		}
	}
}
