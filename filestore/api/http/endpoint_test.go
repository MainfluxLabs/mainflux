// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
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
	largeFileName = "large.bin"
	largeFileSize = 64 << 10
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
	// The server sets Content-Length itself for bodies that fit its write buffer.
	large := filestore.FileInfo{Name: largeFileName, Class: binariesClass, Format: binFormat}
	require.Nil(t, svc.SaveGroupFile(context.Background(), bytes.NewReader(make([]byte, largeFileSize)), token, groupID, large))

	h := httpapi.MakeHandler(mocktracer.New(), svc, mocks.NewAuthService("", nil, nil), logger.NewMock())
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	return srv
}

func TestRangeRequest(t *testing.T) {
	srv := newServer(t)
	size := len(fileContent)
	largeContent := string(make([]byte, largeFileSize))

	cases := []struct {
		desc         string
		file         string
		rangeHeader  string
		status       int
		body         string
		contentRange string
	}{
		{
			desc:        "view file without range",
			file:        fileName,
			rangeHeader: "",
			status:      http.StatusOK,
			body:        fileContent,
		},
		{
			desc:   "view large file without range",
			file:   largeFileName,
			status: http.StatusOK,
			body:   largeContent,
		},
		{
			desc:         "view large file with open-ended range",
			file:         largeFileName,
			rangeHeader:  "bytes=1024-",
			status:       http.StatusPartialContent,
			body:         largeContent[1024:],
			contentRange: fmt.Sprintf("bytes 1024-%d/%d", largeFileSize-1, largeFileSize),
		},
		{
			desc:         "view file with bounded range",
			file:         fileName,
			rangeHeader:  "bytes=6-9",
			status:       http.StatusPartialContent,
			body:         "main",
			contentRange: fmt.Sprintf("bytes 6-9/%d", size),
		},
		{
			desc:         "view file with open-ended range",
			file:         fileName,
			rangeHeader:  "bytes=6-",
			status:       http.StatusPartialContent,
			body:         "mainflux",
			contentRange: fmt.Sprintf("bytes 6-13/%d", size),
		},
		{
			desc:         "view file with suffix range",
			file:         fileName,
			rangeHeader:  "bytes=-3",
			status:       http.StatusPartialContent,
			body:         "lux",
			contentRange: fmt.Sprintf("bytes 11-13/%d", size),
		},
		{
			desc:        "view file with multiple ranges",
			file:        fileName,
			rangeHeader: "bytes=0-1,4-5",
			status:      http.StatusOK,
			body:        fileContent,
		},
		{
			desc:        "view file with unsupported range unit",
			file:        fileName,
			rangeHeader: "items=0-1",
			status:      http.StatusOK,
			body:        fileContent,
		},
		{
			desc:        "view file with malformed range",
			file:        fileName,
			rangeHeader: "bytes=5-2",
			status:      http.StatusOK,
			body:        fileContent,
		},
		{
			desc:        "view file with range starting past file size",
			file:        fileName,
			rangeHeader: fmt.Sprintf("bytes=%d-", size),
			status:      http.StatusRequestedRangeNotSatisfiable,
		},
		{
			desc:        "view file with empty suffix range",
			file:        fileName,
			rangeHeader: "bytes=-0",
			status:      http.StatusRequestedRangeNotSatisfiable,
		},
	}

	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/groupfiles/%s", srv.URL, tc.file), nil)
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
		assert.Equal(t, int64(len(tc.body)), res.ContentLength, fmt.Sprintf("%s: Content-Length mismatch", tc.desc))
		assert.Empty(t, res.TransferEncoding, fmt.Sprintf("%s: unexpected transfer encoding", tc.desc))
	}
}

func TestSaveFileFormats(t *testing.T) {
	srv := newServer(t)

	cases := []struct {
		desc    string
		name    string
		content []byte
		status  int
	}{
		{
			desc:    "upload bin firmware",
			name:    "app.bin",
			content: []byte{0x00, 0xff, 0x10, 0x20},
			status:  http.StatusCreated,
		},
		{
			desc:    "upload hex firmware",
			name:    "app.hex",
			content: []byte(":10010000214601360121470136007EFE09D2190140\n:00000001FF\n"),
			status:  http.StatusCreated,
		},
		{
			desc:    "upload hex with binary content",
			name:    "bad.hex",
			content: []byte{0x00, 0xff, 0x10, 0x20},
			status:  http.StatusUnsupportedMediaType,
		},
		{
			desc:    "upload zip archive",
			name:    "app.zip",
			content: zipArchive(t),
			status:  http.StatusCreated,
		},
		{
			desc:    "upload zip with non-zip content",
			name:    "bad.zip",
			content: []byte(fileContent),
			status:  http.StatusUnsupportedMediaType,
		},
		{
			desc:    "upload unsupported format",
			name:    "app.elf",
			content: []byte(fileContent),
			status:  http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		body := &bytes.Buffer{}
		mw := multipart.NewWriter(body)
		fw, err := mw.CreateFormFile("file", tc.name)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		_, err = fw.Write(tc.content)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		require.Nil(t, mw.Close(), fmt.Sprintf("%s: closing multipart writer failed", tc.desc))

		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/files", srv.URL), body)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		req.Header.Set("Authorization", apiutil.ThingKeyPrefixInternal+thingKey)
		req.Header.Set("Content-Type", mw.FormDataContentType())

		res, err := srv.Client().Do(req)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		res.Body.Close()

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func zipArchive(t *testing.T) []byte {
	t.Helper()

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	w, err := zw.Create("app.bin")
	require.Nil(t, err, fmt.Sprintf("creating zip entry failed: %s", err))
	_, err = w.Write([]byte(fileContent))
	require.Nil(t, err, fmt.Sprintf("writing zip entry failed: %s", err))
	require.Nil(t, zw.Close(), "closing zip writer failed")

	return buf.Bytes()
}
