// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"mime/multipart"

	"github.com/MainfluxLabs/mainflux/filestore"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
)

const (
	maxLimitSize = 200
)

var ErrMissingName = errors.New("missing file name")

type info struct {
	name   string
	class  string
	format string
}

type fileInfoParams struct {
	fileInfo filestore.FileInfo
	file     multipart.File
}

type listFilesParams struct {
	info
	pageMetadata filestore.PageMetadata
}

type saveFileReq struct {
	key      things.ThingKey
	fileInfo filestore.FileInfo
	file     multipart.File
}

func (req saveFileReq) validate() error {
	if req.key.Value == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type updateFileReq struct {
	key      things.ThingKey
	fileInfo filestore.FileInfo
}

func (req updateFileReq) validate() error {
	if req.key.Value == "" {
		return apiutil.ErrBearerKey
	}

	if req.fileInfo.Name == "" {
		return ErrMissingName
	}

	return nil
}

type listFilesReq struct {
	key things.ThingKey
	info
	pageMetadata filestore.PageMetadata
}

func (req listFilesReq) validate() error {
	if req.key.Value == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}

type fileReq struct {
	key things.ThingKey
	info
}

func (req fileReq) validate() error {
	if req.key.Value == "" {
		return apiutil.ErrBearerToken
	}

	if req.name == "" {
		return ErrMissingName
	}

	return nil
}

type saveGroupFileReq struct {
	token    string
	groupID  string
	fileInfo filestore.FileInfo
	file     multipart.File
}

func (req saveGroupFileReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return nil
}

type updateGroupFileReq struct {
	token    string
	groupID  string
	fileInfo filestore.FileInfo
}

func (req updateGroupFileReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.fileInfo.Name == "" {
		return ErrMissingName
	}

	return nil
}

type listGroupFilesReq struct {
	token   string
	groupID string
	info
	pageMetadata filestore.PageMetadata
}

func (req listGroupFilesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}

type groupFileReq struct {
	token   string
	groupID string
	info
}

func (req groupFileReq) validate() error {
	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.name == "" {
		return ErrMissingName
	}

	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type groupFileByKeyReq struct {
	key things.ThingKey
	info
}

func (req groupFileByKeyReq) validate() error {
	if req.name == "" {
		return ErrMissingName
	}

	if req.key.Value == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}
