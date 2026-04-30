// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/filestore"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType            = "application/json"
	multiPartContentType   = "multipart/form-data"
	octetStreamContentType = "application/octet-stream"
	maxMemory              = 32 << 20
	metadataKey            = "metadata"
	fileKey                = "file"
	nameKey                = "name"
	imagesClass            = "images"
	documentsClass         = "documents"
	bimClass               = "BIM"
	pointcloudsClass       = "pointclouds"
	binariesClass          = "binaries"
	jpgFormat              = "jpg"  // JPG image
	jpegFormat             = "jpeg" // JPEG image
	svgFormat              = "svg"  // Scalable Vector Graphics
	pngFormat              = "png"  // Portable Network Graphics
	pdfFormat              = "pdf"  // Portable Document Format
	csvFormat              = "csv"  // Comma-separated values
	txtFormat              = "txt"  // Text
	docFormat              = "doc"  // Microsoft Word
	docxFormat             = "docx" // Microsoft Word
	xlsFormat              = "xls"  // Microsoft Excel
	xlsxFormat             = "xlsx" // Microsoft Excel
	pptFormat              = "ppt"  // Microsoft PowerPoint
	pptxFormat             = "pptx" // Microsoft PowerPoint
	odtFormat              = "odt"  // OpenDocument Text
	odfFormat              = "odf"  // OpenDocument Formula
	odpFormat              = "odp"  // OpenDocument Presentation
	odsFormat              = "ods"  // OpenDocument Spreadsheet
	xpsFormat              = "xps"  // XML Paper Specification
	ifcFormat              = "ifc"  // Building Information Modeling
	e57Format              = "e57"  // Point Clouds
	binFormat              = "bin"  // Point Clouds
	limitKey               = "limit"
	formatKey              = "format"
	offsetKey              = "offset"
	classKey               = "class"
	orderKey               = "order"
	timeKey                = "time"
	directionKey           = "dir"
	idKey                  = "id"
	defDirection           = "desc"
	defOrder               = "time"
	defOffset              = 0
	defLimit               = 10
	maxFileNameLen         = 255
)

// MakeHandler returns a HTTP handler for API endpoints. maxUploadBytes bounds
// the size of any multipart upload body; requests exceeding it are rejected.
func MakeHandler(tracer opentracing.Tracer, svc filestore.Service, logger logger.Logger, maxUploadBytes int64) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/files", kithttp.NewServer(
		kitot.TraceServer(tracer, "save_file")(saveFileEndpoint(svc)),
		decodeSaveFile(maxUploadBytes),
		encodeResponse,
		opts...,
	))
	r.Put("/files/:name", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_file")(updateFileEndpoint(svc)),
		decodeUpdateFile,
		encodeResponse,
		opts...,
	))
	r.Get("/files", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_files")(listFilesEndpoint(svc)),
		decodeListFiles,
		encodeResponse,
		opts...,
	))
	r.Get("/files/:name", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_file")(viewFileEndpoint(svc)),
		decodeFile,
		encodeViewFileResponse,
		opts...,
	))
	r.Delete("/files/:name", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_file")(removeFileEndpoint(svc)),
		decodeFile,
		encodeResponse,
		opts...,
	))
	r.Post("/groups/:id/files", kithttp.NewServer(
		kitot.TraceServer(tracer, "save_group_file")(saveGroupFileEndpoint(svc)),
		decodeSaveGroupFile(maxUploadBytes),
		encodeResponse,
		opts...,
	))
	r.Put("/groups/:id/files/:name", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_group_file")(updateGroupFileEndpoint(svc)),
		decodeUpdateGroupFile,
		encodeResponse,
		opts...,
	))
	r.Get("/groups/:id/files", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_group_files")(listGroupFilesEndpoint(svc)),
		decodeListGroupFiles,
		encodeResponse,
		opts...,
	))
	r.Get("/groups/:id/files/:name", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_group_file")(viewGroupFileEndpoint(svc)),
		decodeGroupFile,
		encodeViewFileResponse,
		opts...,
	))
	r.Get("/groupfiles/:name", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_group_file_by_thing")(viewGroupFileByKeyEndpoint(svc)),
		decodeGroupFileByKey,
		encodeViewFileResponse,
		opts...,
	))
	r.Delete("/groups/:id/files/:name", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_group_file")(removeGroupFileEndpoint(svc)),
		decodeGroupFile,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("things"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeSaveFile(maxUploadBytes int64) kithttp.DecodeRequestFunc {
	return func(_ context.Context, r *http.Request) (any, error) {
		if !strings.Contains(r.Header.Get("Content-Type"), multiPartContentType) {
			return nil, apiutil.ErrUnsupportedContentType
		}

		if r.ContentLength > maxUploadBytes {
			return nil, apiutil.ErrLimitSize
		}
		r.Body = http.MaxBytesReader(nil, r.Body, maxUploadBytes)

		fip, err := getFileInfoParams(r)
		if err != nil {
			return nil, err
		}

		req := saveFileReq{
			key:      apiutil.ExtractThingKey(r),
			fileInfo: fip.fileInfo,
			file:     fip.file,
		}

		return req, nil
	}
}

func decodeUpdateFile(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	name := bone.GetValue(r, nameKey)

	class, format, err := parseFileName(name)
	if err != nil {
		return nil, err
	}

	req := updateFileReq{
		key: apiutil.ExtractThingKey(r),
		fileInfo: filestore.FileInfo{
			Name:   name,
			Format: format,
			Class:  class,
		},
	}

	if err := json.NewDecoder(r.Body).Decode(&req.fileInfo); err != nil {
		return nil, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListFiles(_ context.Context, r *http.Request) (any, error) {
	lfp, err := getListFilesParams(r)
	if err != nil {
		return nil, err
	}

	req := listFilesReq{
		key: apiutil.ExtractThingKey(r),
		info: info{
			name:   lfp.name,
			format: lfp.format,
			class:  lfp.class,
		},
		pageMetadata: lfp.pageMetadata,
	}

	return req, nil
}

func decodeFile(_ context.Context, r *http.Request) (any, error) {
	name := bone.GetValue(r, nameKey)

	class, format, err := parseFileName(name)
	if err != nil {
		return nil, err
	}

	req := fileReq{
		key: apiutil.ExtractThingKey(r),
		info: info{
			name:   name,
			format: format,
			class:  class,
		},
	}

	return req, nil
}

func decodeSaveGroupFile(maxUploadBytes int64) kithttp.DecodeRequestFunc {
	return func(_ context.Context, r *http.Request) (any, error) {
		if !strings.Contains(r.Header.Get("Content-Type"), multiPartContentType) {
			return nil, apiutil.ErrUnsupportedContentType
		}

		if r.ContentLength > maxUploadBytes {
			return nil, apiutil.ErrLimitSize
		}
		r.Body = http.MaxBytesReader(nil, r.Body, maxUploadBytes)

		fip, err := getFileInfoParams(r)
		if err != nil {
			return nil, err
		}

		req := saveGroupFileReq{
			token:    apiutil.ExtractBearerToken(r),
			groupID:  bone.GetValue(r, idKey),
			fileInfo: fip.fileInfo,
			file:     fip.file,
		}

		return req, nil
	}
}

func decodeUpdateGroupFile(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	name := bone.GetValue(r, nameKey)

	class, format, err := parseFileName(name)
	if err != nil {
		return nil, err
	}

	req := updateGroupFileReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, idKey),
		fileInfo: filestore.FileInfo{
			Name:   name,
			Format: format,
			Class:  class,
		},
	}

	if err := json.NewDecoder(r.Body).Decode(&req.fileInfo); err != nil {
		return nil, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListGroupFiles(_ context.Context, r *http.Request) (any, error) {
	lfp, err := getListFilesParams(r)
	if err != nil {
		return nil, err
	}

	req := listGroupFilesReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, idKey),
		info: info{
			name:   lfp.name,
			format: lfp.format,
			class:  lfp.class,
		},
		pageMetadata: lfp.pageMetadata,
	}

	return req, nil
}

func decodeGroupFile(_ context.Context, r *http.Request) (any, error) {
	name := bone.GetValue(r, nameKey)

	class, format, err := parseFileName(name)
	if err != nil {
		return nil, err
	}

	req := groupFileReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, idKey),
		info: info{
			name:   name,
			format: format,
			class:  class,
		},
	}

	return req, nil
}

func decodeGroupFileByKey(_ context.Context, r *http.Request) (any, error) {
	name := bone.GetValue(r, nameKey)

	class, format, err := parseFileName(name)
	if err != nil {
		return nil, err
	}

	req := groupFileByKeyReq{
		key: apiutil.ExtractThingKey(r),
		info: info{
			name:   name,
			format: format,
			class:  class,
		},
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response any) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(apiutil.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}

		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeViewFileResponse(_ context.Context, w http.ResponseWriter, response any) (err error) {
	w.Header().Set("Content-Type", octetStreamContentType)

	switch fr := response.(type) {
	case viewFileRes:
		for k, v := range fr.Headers() {
			w.Header().Set(k, v)
		}
		w.WriteHeader(fr.Code())
		if fr.Empty() {
			return nil
		}
		w.Write(fr.file)
	case streamFileRes:
		for k, v := range fr.Headers() {
			w.Header().Set(k, v)
		}
		w.WriteHeader(fr.Code())
		defer fr.reader.Close()
		_, err = io.Copy(w, fr.reader)
	default:
		return fmt.Errorf("unsupported view response type: %T", response)
	}

	return err
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	apiutil.EncodeError(err, w)
	apiutil.WriteErrorResponse(err, w)
}

func getFileInfoParams(r *http.Request) (fileInfoParams, error) {
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		return fileInfoParams{}, err
	}

	f, h, err := r.FormFile(fileKey)
	if err != nil {
		return fileInfoParams{}, err
	}
	// Closed by the endpoint after the service finishes reading, so
	// multipart tempfiles are released regardless of payload size.

	class, format, err := parseFileName(h.Filename)
	if err != nil {
		return fileInfoParams{}, err
	}

	if err := verifyMagicBytes(f, class, format); err != nil {
		return fileInfoParams{}, err
	}

	metaDataReq := r.FormValue(metadataKey)
	var metadata map[string]any

	if len(metaDataReq) > 0 {
		err = json.Unmarshal([]byte(metaDataReq), &metadata)
		if err != nil {
			return fileInfoParams{}, err
		}
	}

	t := r.FormValue(timeKey)

	var timeStamp float64
	switch t {
	case "":
		timeStamp = float64(time.Now().UnixNano()) / float64(1e9)
	default:
		timeStamp, err = strconv.ParseFloat(t, 64)
		if err != nil {
			return fileInfoParams{}, err
		}
	}

	res := fileInfoParams{
		fileInfo: filestore.FileInfo{
			Name:     h.Filename,
			Format:   format,
			Class:    class,
			Time:     timeStamp,
			Metadata: metadata,
		},
		file: f,
	}

	return res, nil
}

func getListFilesParams(r *http.Request) (listFilesParams, error) {
	l, err := apiutil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return listFilesParams{}, apiutil.ErrInvalidQueryParams
	}

	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return listFilesParams{}, apiutil.ErrInvalidQueryParams
	}

	or, err := apiutil.ReadStringQuery(r, orderKey, defOrder)
	if err != nil {
		return listFilesParams{}, apiutil.ErrInvalidQueryParams
	}

	d, err := apiutil.ReadStringQuery(r, directionKey, defDirection)
	if err != nil {
		return listFilesParams{}, apiutil.ErrInvalidQueryParams
	}

	n, err := apiutil.ReadStringQuery(r, nameKey, "")
	if err != nil {
		return listFilesParams{}, apiutil.ErrInvalidQueryParams
	}

	f, err := apiutil.ReadStringQuery(r, formatKey, "")
	if err != nil {
		return listFilesParams{}, apiutil.ErrInvalidQueryParams
	}

	c, err := apiutil.ReadStringQuery(r, classKey, "")
	if err != nil {
		return listFilesParams{}, apiutil.ErrInvalidQueryParams
	}

	res := listFilesParams{
		info: info{
			name:   n,
			format: f,
			class:  c,
		},
		pageMetadata: filestore.PageMetadata{
			Limit:  l,
			Offset: o,
			Order:  or,
			Dir:    d,
		},
	}

	return res, nil
}

// verifyMagicBytes sniffs the first 512 bytes of f and ensures the detected
// MIME type is plausible for the declared class/format. Binaries and pointclouds
// get no magic-byte enforcement (arbitrary payloads).
func verifyMagicBytes(f multipart.File, class, format string) error {
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if n == 0 {
		return nil
	}

	mime := http.DetectContentType(buf[:n])
	if !mimeMatches(class, format, mime) {
		return apiutil.ErrUnsupportedContentType
	}
	return nil
}

func mimeMatches(class, format, mime string) bool {
	switch class {
	case binariesClass, pointcloudsClass, bimClass:
		return true
	case imagesClass:
		return strings.HasPrefix(mime, "image/")
	case documentsClass:
		switch format {
		case pdfFormat:
			return strings.HasPrefix(mime, "application/pdf")
		case csvFormat, txtFormat:
			return strings.HasPrefix(mime, "text/")
		}
		return strings.HasPrefix(mime, "application/") || strings.HasPrefix(mime, "text/")
	}
	return true
}

// ParseFileName returns file class and format based on file name.
// Rejects names with path separators, traversal sequences, control
// characters, invalid UTF-8, or lengths beyond maxFileNameLen so that
// downstream URL building, header emission, and DB writes are safe.
func parseFileName(name string) (string, string, error) {
	if name == "" || len(name) > maxFileNameLen {
		return "", "", apiutil.ErrInvalidQueryParams
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") || strings.HasPrefix(name, ".") {
		return "", "", apiutil.ErrInvalidQueryParams
	}
	if !utf8.ValidString(name) {
		return "", "", apiutil.ErrInvalidQueryParams
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return "", "", apiutil.ErrInvalidQueryParams
		}
	}

	lastDotIndex := strings.LastIndex(name, ".")
	if lastDotIndex == -1 || lastDotIndex == len(name)-1 {
		return "", "", apiutil.ErrInvalidQueryParams
	}

	format := name[lastDotIndex+1:]

	var class string
	switch format {
	case jpgFormat, pngFormat, svgFormat, jpegFormat:
		class = imagesClass
	case pdfFormat, csvFormat, txtFormat, docFormat,
		docxFormat, odtFormat, odfFormat, odpFormat,
		odsFormat, xlsFormat, xlsxFormat, pptFormat,
		pptxFormat, xpsFormat:
		class = documentsClass
	case ifcFormat:
		class = bimClass
	case e57Format:
		class = pointcloudsClass
	case binFormat:
		class = binariesClass
	default:
		return "", "", apiutil.ErrUnsupportedContentType
	}
	return class, format, nil
}
