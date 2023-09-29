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

var _ users.UserRepository = (*userRepositoryMock)(nil)

type userRepositoryMock struct {
	mu           sync.Mutex
	usersByID    map[string]users.User
	usersByEmail map[string]users.User
}

// NewUserRepository creates in-memory user repository
func NewUserRepository(us []users.User) users.UserRepository {
	usersByEmail := make(map[string]users.User)
	usersByID := make(map[string]users.User)

	for _, u := range us {
		usersByEmail[u.Email] = u
		usersByID[u.ID] = u
	}

	return &userRepositoryMock{
		usersByEmail: usersByEmail,
		usersByID:    usersByID,
	}
}

func (urm *userRepositoryMock) Save(ctx context.Context, u users.User) (string, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := urm.usersByEmail[u.Email]; ok {
		return "", errors.ErrConflict
	}

	urm.usersByEmail[u.Email] = u
	urm.usersByID[u.ID] = u
	return u.ID, nil
}

func (urm *userRepositoryMock) Update(ctx context.Context, u users.User) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := urm.usersByEmail[u.Email]; !ok {
		return errors.ErrNotFound
	}

	urm.usersByEmail[u.Email] = u
	urm.usersByID[u.ID] = u
	return nil
}

func (urm *userRepositoryMock) UpdateUser(ctx context.Context, u users.User) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := urm.usersByEmail[u.Email]; !ok {
		return errors.ErrNotFound
	}

	urm.usersByEmail[u.Email] = u
	urm.usersByID[u.ID] = u
	return nil
}

func (urm *userRepositoryMock) RetrieveByEmail(ctx context.Context, email string) (users.User, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	val, ok := urm.usersByEmail[email]
	if !ok {
		return users.User{}, errors.ErrNotFound
	}

	return val, nil
}

func (urm *userRepositoryMock) RetrieveByID(ctx context.Context, id string) (users.User, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	val, ok := urm.usersByID[id]
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
		val, ok := urm.usersByEmail[pm.Email]
		if !ok {
			return users.UserPage{}, errors.ErrNotFound
		}
		up.Offset = pm.Offset
		up.Limit = pm.Limit
		up.Total = uint64(i)
		up.Users = []users.User{val}
		return up, nil
	}

	sortedUsers := sortUsers(urm.usersByEmail)
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
	for _, u := range urm.usersByEmail {
		users = append(users, u)
	}

	return users, nil
}

func (urm *userRepositoryMock) UpdatePassword(_ context.Context, token, password string) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := urm.usersByEmail[token]; !ok {
		return errors.ErrNotFound
	}
	return nil
}

func (urm *userRepositoryMock) ChangeStatus(ctx context.Context, id, status string) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	u, ok := urm.usersByID[id]
	if !ok {
		return errors.ErrNotFound
	}
	u.Status = status
	urm.usersByID[id] = u
	urm.usersByEmail[u.Email] = u
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
