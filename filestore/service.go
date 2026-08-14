// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package filestore

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/MainfluxLabs/mainflux/filestore/store"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	groupsPath = "groups"
	thingsPath = "things"
)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// SaveFile stores files in filestore
	SaveFile(ctx context.Context, file io.Reader, key string, fi FileInfo) error
	// UpdateFile updates file from filestore
	UpdateFile(ctx context.Context, key string, fi FileInfo) error
	// ViewFile streams file from filestore. Caller must Close.
	ViewFile(ctx context.Context, key string, fi FileInfo) (io.ReadCloser, error)
	// ListFiles retrieves files from filestore by thing
	ListFiles(ctx context.Context, key string, fi FileInfo, pm PageMetadata) (FileThingsPage, error)
	// RemoveFile removes file from filestore
	RemoveFile(ctx context.Context, key string, fi FileInfo) error
	// RemoveFiles removes files from filestore by thing ID
	RemoveFiles(ctx context.Context, thingID string) error

	// SaveGroupFile stores group files in filestore
	SaveGroupFile(ctx context.Context, file io.Reader, token, groupID string, fi FileInfo) error
	// UpdateGroupFile updates group file from filestore
	UpdateGroupFile(ctx context.Context, token, groupID string, fi FileInfo) error
	// ViewGroupFile streams group file from filestore. Caller must Close.
	ViewGroupFile(ctx context.Context, token, groupID string, fi FileInfo) (io.ReadCloser, error)
	// ListGroupFiles retrieves files from filestore by group
	ListGroupFiles(ctx context.Context, token, groupID string, fi FileInfo, pm PageMetadata) (FileGroupsPage, error)
	// RemoveGroupFile removes group file from filestore
	RemoveGroupFile(ctx context.Context, token, groupID string, fi FileInfo) error

	// RemoveAllFilesByGroup removes group files and
	// all files belonging to things related to the given group
	RemoveAllFilesByGroup(ctx context.Context, groupID string) error

	// ViewGroupFileByKey streams group file using Thing Key. Caller must Close.
	ViewGroupFileByKey(ctx context.Context, thingKey string, fi FileInfo) (io.ReadCloser, error)
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total    uint64
	Offset   uint64         `json:"offset,omitempty"`
	Limit    uint64         `json:"limit,omitempty"`
	Name     string         `json:"name,omitempty"`
	Order    string         `json:"order,omitempty"`
	Dir      string         `json:"dir,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// FileInfo contains information about the file
type FileInfo struct {
	Name     string         `json:"name"`
	Class    string         `json:"class"`
	Format   string         `json:"format"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Time     float64        `json:"time,omitempty"`
	Checksum string         `json:"checksum,omitempty"`
}

type filestoreService struct {
	things     domain.ThingsClient
	thingsRepo ThingsRepository
	groupsRepo GroupsRepository
	store      store.FileStore
	logger     logger.Logger
}

var _ Service = (*filestoreService)(nil)

func New(tc domain.ThingsClient, thingsRepo ThingsRepository, groupsRepo GroupsRepository, fs store.FileStore, log logger.Logger) Service {
	return &filestoreService{
		things:     tc,
		thingsRepo: thingsRepo,
		groupsRepo: groupsRepo,
		store:      fs,
		logger:     log,
	}
}

func groupFileKey(groupID, name string) string {
	return filepath.Join(groupsPath, groupID, name)
}

func thingFileDirKey(thingID string) string {
	return filepath.Join(thingsPath, thingID)
}

func thingFileKey(thingID, name string) string {
	return filepath.Join(thingsPath, thingID, name)
}

func (fs *filestoreService) SaveFile(ctx context.Context, file io.Reader, key string, fi FileInfo) error {
	thID, err := fs.identify(ctx, key)
	if err != nil {
		return err
	}

	grID, err := fs.things.GetGroupIDByThing(ctx, thID)
	if err != nil {
		return err
	}

	fi.Checksum = ""
	if err := fs.thingsRepo.Save(ctx, thID, grID, fi); err != nil {
		return err
	}

	objKey := thingFileKey(thID, fi.Name)
	checksum, err := fs.store.Put(ctx, objKey, file)
	if err != nil {
		fs.releaseFile(ctx, objKey, func(ctx context.Context) error {
			return fs.thingsRepo.Remove(ctx, thID, fi)
		})
		return err
	}

	fi.Checksum = checksum

	if err := fs.thingsRepo.UpdateChecksum(ctx, thID, fi); err != nil {
		fs.releaseFile(ctx, objKey, func(ctx context.Context) error {
			return fs.thingsRepo.Remove(ctx, thID, fi)
		})
		return err
	}

	return nil
}

// releaseFile rolls back a reserved row and any bytes written under objKey, so
// a failed upload leaves neither a row pointing at a missing or unverifiable
// object nor an orphaned object with no row. Cleanup runs on a cancellation-free
// context: the most common way an upload fails is the client disconnecting
// mid-stream, and inheriting that cancellation would strand the reservation and
// permanently block re-uploading the same name.
func (fs *filestoreService) releaseFile(ctx context.Context, objKey string, removeRow func(context.Context) error) {
	ctx = context.WithoutCancel(ctx)

	if err := removeRow(ctx); err != nil {
		fs.logger.Error(fmt.Sprintf("failed to release reserved file row after upload error: key=%s err=%s", objKey, err))
	}

	if err := fs.store.Delete(ctx, objKey); err != nil {
		fs.logger.Error(fmt.Sprintf("orphaned object after upload error: key=%s err=%s", objKey, err))
	}
}

func (fs *filestoreService) UpdateFile(ctx context.Context, key string, fi FileInfo) error {
	thID, err := fs.identify(ctx, key)
	if err != nil {
		return err
	}

	err = fs.thingsRepo.Update(ctx, thID, fi)
	if err != nil {
		return err
	}

	return nil
}

func (fs *filestoreService) ListFiles(ctx context.Context, key string, fi FileInfo, pm PageMetadata) (FileThingsPage, error) {
	thID, err := fs.identify(ctx, key)
	if err != nil {
		return FileThingsPage{}, err
	}

	ftp, err := fs.thingsRepo.RetrieveByThing(ctx, thID, fi, pm)
	if err != nil {
		return FileThingsPage{}, err
	}

	return ftp, nil
}

func (fs *filestoreService) RemoveFile(ctx context.Context, key string, fi FileInfo) error {
	thID, err := fs.identify(ctx, key)
	if err != nil {
		return err
	}

	if err := fs.store.Delete(ctx, thingFileKey(thID, fi.Name)); err != nil {
		return err
	}

	if err := fs.thingsRepo.Remove(ctx, thID, fi); err != nil {
		return err
	}

	return nil
}

func (fs *filestoreService) RemoveFiles(ctx context.Context, thingID string) error {
	if err := fs.store.DeletePrefix(ctx, thingFileDirKey(thingID)); err != nil {
		return err
	}

	return fs.thingsRepo.RemoveByThing(ctx, thingID)
}

func (fs *filestoreService) ViewFile(ctx context.Context, key string, fi FileInfo) (io.ReadCloser, error) {
	thID, err := fs.identify(ctx, key)
	if err != nil {
		return nil, err
	}

	f, err := fs.thingsRepo.Retrieve(ctx, thID, fi)
	if err != nil {
		return nil, err
	}

	objKey := thingFileKey(thID, f.Name)
	rc, err := fs.store.Get(ctx, objKey, f.Checksum)
	if err != nil {
		return nil, fs.translateGetErr(objKey, err)
	}

	return rc, nil
}

func (fs *filestoreService) SaveGroupFile(ctx context.Context, file io.Reader, token, groupID string, fi FileInfo) error {
	if err := fs.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	// See SaveFile: the row is reserved first so concurrent uploads of the same
	// name cannot corrupt each other's object.
	fi.Checksum = ""
	if err := fs.groupsRepo.Save(ctx, groupID, fi); err != nil {
		return err
	}

	objKey := groupFileKey(groupID, fi.Name)
	checksum, err := fs.store.Put(ctx, objKey, file)
	if err != nil {
		fs.releaseFile(ctx, objKey, func(ctx context.Context) error {
			return fs.groupsRepo.Remove(ctx, groupID, fi)
		})
		return err
	}

	fi.Checksum = checksum

	if err := fs.groupsRepo.UpdateChecksum(ctx, groupID, fi); err != nil {
		fs.releaseFile(ctx, objKey, func(ctx context.Context) error {
			return fs.groupsRepo.Remove(ctx, groupID, fi)
		})
		return err
	}

	return nil
}

func (fs *filestoreService) UpdateGroupFile(ctx context.Context, token, groupID string, fi FileInfo) error {
	if err := fs.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	return fs.groupsRepo.Update(ctx, groupID, fi)
}

func (fs *filestoreService) ListGroupFiles(ctx context.Context, token, groupID string, fi FileInfo, pm PageMetadata) (FileGroupsPage, error) {
	if err := fs.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupViewer}); err != nil {
		return FileGroupsPage{}, err
	}

	fgp, err := fs.groupsRepo.RetrieveByGroup(ctx, groupID, fi, pm)
	if err != nil {
		return FileGroupsPage{}, err
	}

	return fgp, nil
}

func (fs *filestoreService) RemoveGroupFile(ctx context.Context, token, groupID string, fi FileInfo) error {
	if err := fs.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	key := groupFileKey(groupID, fi.Name)
	if err := fs.store.Delete(ctx, key); err != nil {
		return err
	}

	if err := fs.groupsRepo.Remove(ctx, groupID, fi); err != nil {
		return err
	}

	return nil
}

func (fs *filestoreService) RemoveAllFilesByGroup(ctx context.Context, groupID string) error {
	thingIDs, err := fs.thingsRepo.RetrieveThingIDsByGroup(ctx, groupID)
	if err != nil {
		return err
	}

	if err := fs.store.DeletePrefix(ctx, filepath.Join(groupsPath, groupID)); err != nil {
		return err
	}

	const workers = 8
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	errCh := make(chan error, len(thingIDs))
	for _, thingID := range thingIDs {
		wg.Add(1)
		sem <- struct{}{}
		go func(id string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fs.store.DeletePrefix(ctx, thingFileDirKey(id)); err != nil {
				errCh <- fmt.Errorf("thing %s: %w", id, err)
			}
		}(thingID)
	}
	wg.Wait()
	close(errCh)
	var errs []error
	for e := range errCh {
		errs = append(errs, e)
	}
	if err := stderrors.Join(errs...); err != nil {
		return err
	}

	if err := fs.groupsRepo.RemoveByGroup(ctx, groupID); err != nil {
		return err
	}

	return fs.thingsRepo.RemoveByGroup(ctx, groupID)
}

func (fs *filestoreService) ViewGroupFile(ctx context.Context, token, groupID string, fi FileInfo) (io.ReadCloser, error) {
	if err := fs.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupViewer}); err != nil {
		return nil, err
	}

	f, err := fs.groupsRepo.Retrieve(ctx, groupID, fi)
	if err != nil {
		return nil, err
	}

	key := groupFileKey(groupID, f.Name)
	rc, err := fs.store.Get(ctx, key, f.Checksum)
	if err != nil {
		return nil, fs.translateGetErr(key, err)
	}

	return rc, nil
}

func (fs *filestoreService) ViewGroupFileByKey(ctx context.Context, thingKey string, fi FileInfo) (io.ReadCloser, error) {
	thID, err := fs.identify(ctx, thingKey)
	if err != nil {
		return nil, err
	}
	grID, err := fs.things.GetGroupIDByThing(ctx, thID)
	if err != nil {
		return nil, err
	}
	if err := fs.things.CanThingAccessGroup(ctx, domain.ThingAccessReq{ThingKey: domain.ThingKey{Type: domain.KeyTypeInternal, Value: thingKey}, ID: grID}); err != nil {
		return nil, err
	}

	f, err := fs.groupsRepo.Retrieve(ctx, grID, fi)
	if err != nil {
		return nil, err
	}

	key := groupFileKey(grID, f.Name)
	rc, err := fs.store.Get(ctx, key, f.Checksum)
	if err != nil {
		return nil, fs.translateGetErr(key, err)
	}

	return rc, nil
}

// translateGetErr maps a store miss to dbutil.ErrNotFound so the API returns
// 404 instead of 500. A missing object for an existing index row is a
// storage/DB inconsistency, so it is logged before being flattened for the
// caller.
func (fs *filestoreService) translateGetErr(key string, err error) error {
	if errors.Contains(err, store.ErrNotFound) {
		fs.logger.Warn(fmt.Sprintf("file index row exists but object is missing: key=%s", key))
		return errors.Wrap(dbutil.ErrNotFound, err)
	}
	return err
}

func (fs *filestoreService) identify(ctx context.Context, thingKey string) (string, error) {
	thingID, err := fs.things.Identify(ctx, domain.ThingKey{Type: domain.KeyTypeInternal, Value: thingKey})
	if err != nil {
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}

	return thingID, nil
}
