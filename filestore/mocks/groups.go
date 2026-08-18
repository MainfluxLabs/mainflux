package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/filestore"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
)

var (
	_ filestore.GroupsRepository = (*GroupsRepository)(nil)
)

// GroupsRepository is an in-memory filestore.GroupsRepository for tests.
type GroupsRepository struct {
	mu     sync.Mutex
	byKey  map[string]filestore.FileInfo
	FailOn string // name that triggers a synthetic Save failure
}

func NewGroupsRepository() *GroupsRepository {
	return &GroupsRepository{byKey: map[string]filestore.FileInfo{}}
}

func groupKey(groupID string, fi filestore.FileInfo) string {
	return groupID + "|" + fi.Class + "|" + fi.Format + "|" + fi.Name
}

func (r *GroupsRepository) Save(_ context.Context, groupID string, fi filestore.FileInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.FailOn != "" && fi.Name == r.FailOn {
		return dbutil.ErrCreateEntity
	}
	k := groupKey(groupID, fi)
	// Mirrors the primary key on (group_id, file_name, file_class, file_format).
	if _, ok := r.byKey[k]; ok {
		return dbutil.ErrConflict
	}
	r.byKey[k] = fi
	return nil
}

func (r *GroupsRepository) UpdateChecksum(_ context.Context, groupID string, fi filestore.FileInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := groupKey(groupID, fi)
	got, ok := r.byKey[k]
	if !ok {
		return dbutil.ErrNotFound
	}
	got.Checksum = fi.Checksum
	r.byKey[k] = got
	return nil
}

func (r *GroupsRepository) Update(_ context.Context, groupID string, fi filestore.FileInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := groupKey(groupID, fi)
	if _, ok := r.byKey[k]; !ok {
		return dbutil.ErrNotFound
	}
	r.byKey[k] = fi
	return nil
}

func (r *GroupsRepository) Retrieve(_ context.Context, groupID string, fi filestore.FileInfo) (filestore.FileInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	got, ok := r.byKey[groupKey(groupID, fi)]
	if !ok {
		return filestore.FileInfo{}, dbutil.ErrNotFound
	}
	return got, nil
}

func (r *GroupsRepository) RetrieveByGroup(_ context.Context, _ string, _ filestore.FileInfo, _ filestore.PageMetadata) (filestore.FileGroupsPage, error) {
	return filestore.FileGroupsPage{}, nil
}

func (r *GroupsRepository) Remove(_ context.Context, groupID string, fi filestore.FileInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := groupKey(groupID, fi)
	if _, ok := r.byKey[k]; !ok {
		return dbutil.ErrNotFound
	}
	delete(r.byKey, k)
	return nil
}

func (r *GroupsRepository) RemoveByGroup(_ context.Context, groupID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k := range r.byKey {
		if len(k) > len(groupID) && k[:len(groupID)+1] == groupID+"|" {
			delete(r.byKey, k)
		}
	}
	return nil
}

func (r *GroupsRepository) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byKey)
}
