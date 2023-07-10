// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sort"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
)

var (
	_             users.UserRepository = (*userRepositoryMock)(nil)
	mockUsers     map[string]users.User
	mockUsersByID map[string]users.User
)

type userRepositoryMock struct {
	mu sync.Mutex
}

// NewUserRepository creates in-memory user repository
func NewUserRepository() users.UserRepository {
	mockUsers = make(map[string]users.User)
	mockUsersByID = make(map[string]users.User)
	return &userRepositoryMock{}
}

func (urm *userRepositoryMock) Save(ctx context.Context, user users.User) (string, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := mockUsers[user.Email]; ok {
		return "", errors.ErrConflict
	}

	mockUsers[user.Email] = user
	mockUsersByID[user.ID] = user
	return user.ID, nil
}

func (urm *userRepositoryMock) Update(ctx context.Context, user users.User) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := mockUsers[user.Email]; !ok {
		return errors.ErrNotFound
	}

	mockUsers[user.Email] = user
	mockUsersByID[user.ID] = user
	return nil
}

func (urm *userRepositoryMock) UpdateUser(ctx context.Context, user users.User) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := mockUsers[user.Email]; !ok {
		return errors.ErrNotFound
	}

	mockUsers[user.Email] = user
	mockUsersByID[user.ID] = user
	return nil
}

func (urm *userRepositoryMock) RetrieveByEmail(ctx context.Context, email string) (users.User, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	val, ok := mockUsers[email]
	if !ok {
		return users.User{}, errors.ErrNotFound
	}

	return val, nil
}

func (urm *userRepositoryMock) RetrieveByID(ctx context.Context, id string) (users.User, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	val, ok := mockUsersByID[id]
	if !ok {
		return users.User{}, errors.ErrNotFound
	}

	return val, nil
}

func (urm *userRepositoryMock) RetrieveByIDs(ctx context.Context, ids []string, pm users.PageMetadata) (users.UserPage, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	up := users.UserPage{}
	i := uint64(0)

	if pm.Email != "" {
		val, ok := mockUsers[pm.Email]
		if !ok {
			return users.UserPage{}, errors.ErrNotFound
		}
		up.Offset = pm.Offset
		up.Limit = pm.Limit
		up.Total = uint64(i)
		up.Users = []users.User{val}
		return up, nil
	}

	sortedUsers := sortUsers(mockUsers)
	for _, u := range sortedUsers {
		if i >= pm.Offset && i < pm.Offset+pm.Limit || pm.Limit == 0 {
			switch pm.Status {
			case users.DisabledStatusKey,
				users.EnabledStatusKey:
				if pm.Status == u.Status {
					up.Users = append(up.Users, u)
				}
			default:
				up.Users = append(up.Users, u)
			}
		}
		i++
	}

	up.Offset = pm.Offset
	up.Limit = pm.Limit
	up.Total = uint64(i)

	return up, nil
}

func (urm *userRepositoryMock) RetrieveAll(ctx context.Context) ([]users.User, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	var users []users.User
	for _, user := range mockUsers {
		users = append(users, user)
	}

	return users, nil
}

func (urm *userRepositoryMock) UpdatePassword(_ context.Context, token, password string) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := mockUsers[token]; !ok {
		return errors.ErrNotFound
	}
	return nil
}

func (urm *userRepositoryMock) ChangeStatus(ctx context.Context, id, status string) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	user, ok := mockUsersByID[id]
	if !ok {
		return errors.ErrNotFound
	}
	user.Status = status
	mockUsersByID[id] = user
	mockUsers[user.Email] = user
	return nil
}

func sortUsers(us map[string]users.User) []users.User {
	users := []users.User{}
	ids := make([]string, 0, len(us))
	for k := range us {
		ids = append(ids, k)
	}

	sort.Strings(ids)
	for _, id := range ids {
		users = append(users, us[id])
	}

	return users
}

func (urm *userRepositoryMock) SaveRole(ctx context.Context, id, role string) error {
	panic("not implemented")
}

func (urm *userRepositoryMock) RetrieveRole(ctx context.Context, id string) (string, error) {
	panic("not implemented")
}

func (urm *userRepositoryMock) UpdateRole(ctx context.Context, id, role string) error {
	panic("not implemented")
}

func (urm *userRepositoryMock) RemoveRole(ctx context.Context, id string) error {
	panic("not implemented")
}
