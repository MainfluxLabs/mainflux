// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package filestore

import (
	"context"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
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
	// ViewGroupFile views group file from filestore
	ViewGroupFile(ctx context.Context, token, groupID string, fi FileInfo) ([]byte, error)
	// ListGroupFiles retrieves files from filestore by group
	ListGroupFiles(ctx context.Context, token, groupID string, fi FileInfo, pm PageMetadata) (FileGroupsPage, error)
	// RemoveGroupFile removes group file from filestore
	RemoveGroupFile(ctx context.Context, token, groupID string, fi FileInfo) error

	// RemoveAllFilesByGroup removes group files and
	// all files belonging to things related to the given group
	RemoveAllFilesByGroup(ctx context.Context, groupID string) error

	// ViewGroupFileByKey views group file from filestore using Thing Key
	ViewGroupFileByKey(ctx context.Context, thingKey string, fi FileInfo) ([]byte, error)
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
}

type filestoreService struct {
	things     protomfx.ThingsServiceClient
	thingsRepo ThingsRepository
	groupsRepo GroupsRepository
}

var _ Service = (*filestoreService)(nil)

// New instantiates the filestore service implementation.
func New(tc protomfx.ThingsServiceClient, thingsRepo ThingsRepository, groupsRepo GroupsRepository) Service {
	return &filestoreService{
		things:     tc,
		thingsRepo: thingsRepo,
		groupsRepo: groupsRepo,
	}
}

func (fs *filestoreService) SaveFile(ctx context.Context, file io.Reader, key string, fi FileInfo) error {
	thID, err := fs.identify(ctx, key)
	if err != nil {
		return err
	}

	grID, err := fs.things.GetGroupIDByThing(ctx, &protomfx.ThingID{Value: thID})
	if err != nil {
		return err
	}

	path := filepath.Join(filesPath, thingsPath, thID)
	if err := createFile(path, fi.Name, file); err != nil {
		return err
	}

	if err = fs.thingsRepo.Save(ctx, thID, grID.GetValue(), fi); err != nil {
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
	if _, err := fs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Editor}); err != nil {
		return err
	}

	path := filepath.Join(filesPath, groupsPath, groupID)
	if err := createFile(path, fi.Name, file); err != nil {
		return err
	}

	if err := fs.groupsRepo.Save(ctx, groupID, fi); err != nil {
		return err
	}

	return nil
}

func (fs *filestoreService) UpdateGroupFile(ctx context.Context, token, groupID string, fi FileInfo) error {
	if _, err := fs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Editor}); err != nil {
		return err
	}

	return fs.groupsRepo.Update(ctx, groupID, fi)
}

func (fs *filestoreService) ListGroupFiles(ctx context.Context, token, groupID string, fi FileInfo, pm PageMetadata) (FileGroupsPage, error) {
	if _, err := fs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Viewer}); err != nil {
		return FileGroupsPage{}, err
	}

	fgp, err := fs.groupsRepo.RetrieveByGroup(ctx, groupID, fi, pm)
	if err != nil {
		return FileGroupsPage{}, err
	}

	return fgp, nil
}

func (fs *filestoreService) RemoveGroupFile(ctx context.Context, token, groupID string, fi FileInfo) error {
	if _, err := fs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Editor}); err != nil {
		return err
	}

	path := filepath.Join(filesPath, groupsPath, groupID, fi.Name)
	if err := os.Remove(path); err != nil {
		return err
	}

	if err := fs.groupsRepo.Remove(ctx, groupID, fi); err != nil {
		return err
	}

	directories := []string{groupsPath, groupID}
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

func (fs *filestoreService) RemoveAllFilesByGroup(ctx context.Context, groupID string) error {
	// Remove group files
	if err := fs.groupsRepo.RemoveByGroup(ctx, groupID); err != nil {
		return err
	}

	gp := filepath.Join(filesPath, groupsPath, groupID)
	if err := os.RemoveAll(gp); err != nil {
		return err
	}

	// Remove all files belonging to things related to the group
	thingIDs, err := fs.thingsRepo.RetrieveThingIDsByGroup(ctx, groupID)
	if err != nil {
		return err
	}

	if err := fs.thingsRepo.RemoveByGroup(ctx, groupID); err != nil {
		return err
	}

	// File removal is done sequentially to keep the operation simple
	// Parallel deletion can be added later if needed
	for _, thingID := range thingIDs {
		tp := filepath.Join(filesPath, thingsPath, thingID)
		if err := os.RemoveAll(tp); err != nil {
			return err
		}
	}

	return nil
}

func (fs *filestoreService) ViewGroupFile(ctx context.Context, token, groupID string, fi FileInfo) ([]byte, error) {
	if _, err := fs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Viewer}); err != nil {
		return nil, err
	}

	f, err := fs.groupsRepo.Retrieve(ctx, groupID, fi)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(filesPath, groupsPath, groupID, f.Name)
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}

func (fs *filestoreService) ViewGroupFileByKey(ctx context.Context, thingKey string, fi FileInfo) ([]byte, error) {
	thID, err := fs.identify(ctx, thingKey)
	if err != nil {
		return nil, err
	}
	grID, err := fs.things.GetGroupIDByThing(ctx, &protomfx.ThingID{Value: thID})
	if err != nil {
		return nil, err
	}

	f, err := fs.groupsRepo.Retrieve(ctx, grID.GetValue(), fi)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(filesPath, groupsPath, grID.GetValue(), f.Name)
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
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
	thingID, err := fs.things.Identify(ctx, &protomfx.ThingKey{Type: things.KeyTypeInternal, Value: thingKey})
	if err != nil {
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}

	return thingID.GetValue(), nil
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
