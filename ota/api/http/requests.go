// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/hex"

	"github.com/MainfluxLabs/mainflux/ota"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	maxLimitSize    = 200
	maxThingIDs     = 1000
	maxFileNameSize = 255
	maxVersionSize  = 254
	checksumSize    = 64
	maxTimeout      = 7 * 24 * 60 * 60
)

var (
	// ErrMissingUpdateID indicates a missing update ID.
	ErrMissingUpdateID = errors.New("missing update id")

	// ErrInvalidFileName indicates a missing or too long firmware file name.
	ErrInvalidFileName = errors.New("missing or invalid file name")

	// ErrInvalidVersion indicates a missing or too long firmware version.
	ErrInvalidVersion = errors.New("missing or invalid version")

	// ErrInvalidChecksum indicates a firmware checksum that is not a SHA-256 hex digest.
	ErrInvalidChecksum = errors.New("missing or invalid checksum")

	// ErrInvalidFileSize indicates a negative firmware file size.
	ErrInvalidFileSize = errors.New("invalid file size")

	// ErrInvalidTimeout indicates a negative or too long timeout.
	ErrInvalidTimeout = errors.New("invalid timeout")

	// ErrDuplicateThingIDs indicates a thing ID listed more than once.
	ErrDuplicateThingIDs = errors.New("duplicate thing ids")
)

func validatePageMetadata(pm ota.PageMetadata, orderFields map[string]string) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}

	return common.Validate(maxLimitSize, orderFields)
}

type createUpdateReq struct {
	token       string
	groupID     string
	FileName    string   `json:"file_name"`
	Version     string   `json:"version"`
	FileSize    int64    `json:"file_size"`
	Checksum    string   `json:"checksum"`
	Description string   `json:"description,omitempty"`
	Timeout     int64    `json:"timeout,omitempty"`
	ThingIDs    []string `json:"thing_ids"`
}

func (req createUpdateReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.FileName == "" || len(req.FileName) > maxFileNameSize {
		return ErrInvalidFileName
	}

	if req.Version == "" || len(req.Version) > maxVersionSize {
		return ErrInvalidVersion
	}

	if !isSHA256Hex(req.Checksum) {
		return ErrInvalidChecksum
	}

	if req.FileSize < 0 {
		return ErrInvalidFileSize
	}

	if req.Timeout < 0 || req.Timeout > maxTimeout {
		return ErrInvalidTimeout
	}

	if len(req.ThingIDs) == 0 || len(req.ThingIDs) > maxThingIDs {
		return apiutil.ErrThingIDsSize
	}

	ids := make(map[string]struct{}, len(req.ThingIDs))
	for _, thingID := range req.ThingIDs {
		if thingID == "" {
			return apiutil.ErrMissingThingID
		}
		if _, ok := ids[thingID]; ok {
			return ErrDuplicateThingIDs
		}
		ids[thingID] = struct{}{}
	}

	return nil
}

func isSHA256Hex(checksum string) bool {
	if len(checksum) != checksumSize {
		return false
	}
	_, err := hex.DecodeString(checksum)
	return err == nil
}

type updateReq struct {
	token string
	id    string
}

func (req updateReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return ErrMissingUpdateID
	}

	return nil
}

type listUpdatesReq struct {
	token        string
	groupID      string
	pageMetadata ota.PageMetadata
}

func (req listUpdatesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return validatePageMetadata(req.pageMetadata, ota.UpdateOrderFields)
}

type listThingUpdatesReq struct {
	token        string
	updateID     string
	pageMetadata ota.PageMetadata
}

func (req listThingUpdatesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.updateID == "" {
		return ErrMissingUpdateID
	}

	switch req.pageMetadata.Status {
	case "", ota.StatusPending, ota.StatusDownloading, ota.StatusInstalling, ota.StatusSucceeded, ota.StatusFailed:
	default:
		return ota.ErrInvalidStatus
	}

	return validatePageMetadata(req.pageMetadata, ota.ThingUpdateOrderFields)
}

type viewPendingReq struct {
	key domain.ThingKey
}

func (req viewPendingReq) validate() error {
	return apiutil.ValidateThingKey(req.key)
}

type reportStatusReq struct {
	key      domain.ThingKey
	UpdateID string `json:"update_id"`
	Status   string `json:"status"`
	Version  string `json:"version,omitempty"`
	Error    string `json:"error,omitempty"`
}

func (req reportStatusReq) validate() error {
	if err := apiutil.ValidateThingKey(req.key); err != nil {
		return err
	}

	if req.UpdateID == "" {
		return ErrMissingUpdateID
	}

	return nil
}
