// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/ota"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*updateRes)(nil)
	_ apiutil.Response = (*updatesPageRes)(nil)
	_ apiutil.Response = (*thingUpdatesPageRes)(nil)
	_ apiutil.Response = (*pendingRes)(nil)
)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Order  string `json:"order,omitempty"`
	Dir    string `json:"dir,omitempty"`
}

type statsRes struct {
	Targets     uint64 `json:"targets"`
	Pending     uint64 `json:"pending"`
	Downloading uint64 `json:"downloading"`
	Installing  uint64 `json:"installing"`
	Succeeded   uint64 `json:"succeeded"`
	Failed      uint64 `json:"failed"`
}

type updateRes struct {
	ID          string    `json:"id"`
	GroupID     string    `json:"group_id"`
	FileName    string    `json:"file_name"`
	Version     string    `json:"version"`
	FileSize    int64     `json:"file_size"`
	Checksum    string    `json:"checksum"`
	Description string    `json:"description,omitempty"`
	Timeout     int64     `json:"timeout"`
	CreatedAt   time.Time `json:"created_at"`
	Stats       statsRes  `json:"stats"`
	created     bool
}

func (res updateRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res updateRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateRes) Empty() bool {
	return false
}

type updatesPageRes struct {
	pageRes
	Updates []updateRes `json:"updates"`
}

func (res updatesPageRes) Code() int {
	return http.StatusOK
}

func (res updatesPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updatesPageRes) Empty() bool {
	return false
}

type thingUpdateRes struct {
	ThingID         string    `json:"thing_id"`
	Status          string    `json:"status"`
	Error           string    `json:"error,omitempty"`
	ReportedVersion string    `json:"reported_version,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type thingUpdatesPageRes struct {
	pageRes
	Status       string           `json:"status,omitempty"`
	ThingUpdates []thingUpdateRes `json:"thing_updates"`
}

func (res thingUpdatesPageRes) Code() int {
	return http.StatusOK
}

func (res thingUpdatesPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res thingUpdatesPageRes) Empty() bool {
	return false
}

type pendingRes struct {
	UpdateID string `json:"update_id"`
	Version  string `json:"version"`
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
}

func (res pendingRes) Code() int {
	return http.StatusOK
}

func (res pendingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res pendingRes) Empty() bool {
	return false
}

func buildUpdateRes(u ota.Update, created bool) updateRes {
	return updateRes{
		ID:          u.ID,
		GroupID:     u.GroupID,
		FileName:    u.FileName,
		Version:     u.Version,
		FileSize:    u.FileSize,
		Checksum:    u.Checksum,
		Description: u.Description,
		Timeout:     int64(u.Timeout / time.Second),
		CreatedAt:   u.CreatedAt,
		Stats: statsRes{
			Targets:     u.Stats.Targets,
			Pending:     u.Stats.Pending,
			Downloading: u.Stats.Downloading,
			Installing:  u.Stats.Installing,
			Succeeded:   u.Stats.Succeeded,
			Failed:      u.Stats.Failed,
		},
		created: created,
	}
}

func buildUpdatesPageRes(page ota.UpdatesPage, pm ota.PageMetadata) updatesPageRes {
	res := updatesPageRes{
		pageRes: pageRes{
			Total:  page.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
		Updates: []updateRes{},
	}

	for _, u := range page.Updates {
		res.Updates = append(res.Updates, buildUpdateRes(u, false))
	}

	return res
}

func buildThingUpdatesPageRes(page ota.ThingUpdatesPage, pm ota.PageMetadata) thingUpdatesPageRes {
	res := thingUpdatesPageRes{
		pageRes: pageRes{
			Total:  page.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
		Status:       pm.Status,
		ThingUpdates: []thingUpdateRes{},
	}

	for _, tu := range page.ThingUpdates {
		res.ThingUpdates = append(res.ThingUpdates, thingUpdateRes{
			ThingID:         tu.ThingID,
			Status:          tu.Status,
			Error:           tu.Error,
			ReportedVersion: tu.ReportedVersion,
			UpdatedAt:       tu.UpdatedAt,
		})
	}

	return res
}

func buildPendingRes(u ota.Update) pendingRes {
	return pendingRes{
		UpdateID: u.ID,
		Version:  u.Version,
		FileName: u.FileName,
		Size:     u.FileSize,
		Checksum: u.Checksum,
	}
}
