package mocks

import (
	"context"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/users"
)

var _ domain.UsersClient = (*usersServiceClientMock)(nil)

type usersServiceClientMock struct {
	usersByID     map[string]users.User
	usersByEmails map[string]users.User
}

func NewUsersService(usersByID map[string]users.User, usersByEmails map[string]users.User) domain.UsersClient {
	return &usersServiceClientMock{usersByID: usersByID, usersByEmails: usersByEmails}
}

func (svc *usersServiceClientMock) GetUsersByIDs(_ context.Context, ids []string, pm domain.UsersPageMetadata) (domain.UsersPage, error) {
	if pm.Limit == 0 {
		pm.Limit = uint64(len(ids))
	}

	page := domain.UsersPage{
		Total: 0,
		Users: []domain.User{},
	}

	i := uint64(0)
	for _, id := range ids {
		if user, ok := svc.usersByID[id]; ok {
			if pm.Email != "" && !strings.Contains(user.Email, pm.Email) {
				continue
			}

			if i >= pm.Offset && i < pm.Offset+pm.Limit {
				page.Users = append(page.Users, domain.User{ID: user.ID, Email: user.Email, Status: user.Status})
			}
			i++
		}
	}
	page.Total = i

	return page, nil
}

func (svc *usersServiceClientMock) GetUsersByEmails(_ context.Context, emails []string) ([]domain.User, error) {
	var result []domain.User
	for _, email := range emails {
		if user, ok := svc.usersByEmails[email]; ok {
			result = append(result, domain.User{ID: user.ID, Email: user.Email, Status: user.Status})
		}
	}

	return result, nil
}
