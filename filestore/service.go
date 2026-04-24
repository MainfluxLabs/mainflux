// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package filestore

import (
	"context"
	stderrors "errors"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/MainfluxLabs/mainflux/filestore/store"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	filesPath  = "files"
	groupsPath = "groups"
	thingsPath = "things"
	permission = 0755
)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// SaveFile stores files in filestore
	SaveFile(ctx context.Context, file io.Reader, key string, fi FileInfo) error
	// UpdateFile updates file from filestore
	UpdateFile(ctx context.Context, key string, fi FileInfo) error
	// ViewFile views file from filestore
	ViewFile(ctx context.Context, key string, fi FileInfo) ([]byte, error)
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

func New(tc domain.ThingsClient, thingsRepo ThingsRepository, groupsRepo GroupsRepository, fs store.FileStore) Service {
	return &filestoreService{
		things:     tc,
		thingsRepo: thingsRepo,
		groupsRepo: groupsRepo,
		store:      fs,
	}
}

func groupFileKey(groupID, name string) string {
	return filepath.Join(groupsPath, groupID, name)
}

func thingFileDirKey(thingID string) string {
	return filepath.Join(thingsPath, thingID)
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

	path := filepath.Join(filesPath, thingsPath, thID)
	if err := createFile(path, fi.Name, file); err != nil {
		return err
	}

	if err = fs.thingsRepo.Save(ctx, thID, grID, fi); err != nil {
		return err
	}

	return nil
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

	path := filepath.Join(filesPath, thingsPath, thID, fi.Name)
	if err := os.Remove(path); err != nil {
		return err
	}

	if err := fs.thingsRepo.Remove(ctx, thID, fi); err != nil {
		return err
	}

	directories := []string{thingsPath, thID}
	for i := len(directories) - 1; i >= 0; i-- {
		path := filepath.Join(directories[:i+1]...)
		if isDirEmpty(path) {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		}
	}

	return nil
}

func (fs *filestoreService) RemoveFiles(ctx context.Context, thingID string) error {
	if err := fs.thingsRepo.RemoveByThing(ctx, thingID); err != nil {
		return err
	}

	dirPath := filepath.Join(filesPath, thingsPath, thingID)
	if err := os.RemoveAll(dirPath); err != nil {
		return err
	}

	return nil
}

func (fs *filestoreService) ViewFile(ctx context.Context, key string, fi FileInfo) ([]byte, error) {
	thID, err := fs.identify(ctx, key)
	if err != nil {
		return nil, err
	}

	f, err := fs.thingsRepo.Retrieve(ctx, thID, fi)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(filesPath, thingsPath, thID, f.Name)
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}

func (fs *filestoreService) SaveGroupFile(ctx context.Context, file io.Reader, token, groupID string, fi FileInfo) error {
	if err := fs.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	checksum, err := fs.store.Put(ctx, groupFileKey(groupID, fi.Name), file)
	if err != nil {
		return err
	}
	fi.Checksum = checksum

	if err := fs.groupsRepo.Save(ctx, groupID, fi); err != nil {
		key := groupFileKey(groupID, fi.Name)
		if delErr := fs.store.Delete(ctx, key); delErr != nil {
			fs.logger.Error(fmt.Sprintf("orphaned object after failed DB save: key=%s err=%s", key, delErr))
		}
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

	if err := fs.groupsRepo.Remove(ctx, groupID, fi); err != nil {
		return err
	}

	key := groupFileKey(groupID, fi.Name)
	if err := fs.store.Delete(ctx, key); err != nil {
		fs.logger.Error(fmt.Sprintf("orphaned object after DB remove: key=%s err=%s", key, err))
		return err
	}

	return nil
}

func (fs *filestoreService) RemoveAllFilesByGroup(ctx context.Context, groupID string) error {
	if err := fs.groupsRepo.RemoveByGroup(ctx, groupID); err != nil {
		return err
	}

	if err := fs.store.DeletePrefix(ctx, filepath.Join(groupsPath, groupID)); err != nil {
		return err
	}

	thingIDs, err := fs.thingsRepo.RetrieveThingIDsByGroup(ctx, groupID)
	if err != nil {
		return err
	}

	if err := fs.thingsRepo.RemoveByGroup(ctx, groupID); err != nil {
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
	return stderrors.Join(errs...)
}

func (fs *filestoreService) ViewGroupFile(ctx context.Context, token, groupID string, fi FileInfo) (io.ReadCloser, error) {
	if err := fs.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupViewer}); err != nil {
		return nil, err
	}

	f, err := fs.groupsRepo.Retrieve(ctx, groupID, fi)
	if err != nil {
		return nil, err
	}

	return fs.store.Get(ctx, groupFileKey(groupID, f.Name), f.Checksum)
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

	f, err := fs.groupsRepo.Retrieve(ctx, grID, fi)
	if err != nil {
		return nil, err
	}

	return fs.store.Get(ctx, groupFileKey(grID, f.Name), f.Checksum)
}

// createDirectory creates directory for storing files
func createFile(path, name string, file io.Reader) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, permission)
		if err != nil {
			return err
		}
	}

	tmpfile, err := os.Create(filepath.Join(path, name))
	if err != nil {
		return err
	}
	defer tmpfile.Close()

	if _, err := io.Copy(tmpfile, file); err != nil {
		return err
	}

	return nil
}

func (fs *filestoreService) identify(ctx context.Context, thingKey string) (string, error) {
	thingID, err := fs.things.Identify(ctx, domain.ThingKey{Type: domain.KeyTypeInternal, Value: thingKey})
	if err != nil {
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}

	return thingID, nil
}

// isDirEmpty checks if directory is empty
func isDirEmpty(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	_, err = f.Readdirnames(1)

	return err == io.EOF
}
