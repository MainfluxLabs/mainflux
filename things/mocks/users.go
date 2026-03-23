package mocks

import (
	"context"
	"strings"

	domainusers "github.com/MainfluxLabs/mainflux/pkg/domain/users"
	"github.com/MainfluxLabs/mainflux/users"
)

var _ domainusers.Client = (*usersServiceClientMock)(nil)

type usersServiceClientMock struct {
	usersByID     map[string]users.User
	usersByEmails map[string]users.User
}

func NewUsersService(usersByID map[string]users.User, usersByEmails map[string]users.User) domainusers.Client {
	return &usersServiceClientMock{usersByID: usersByID, usersByEmails: usersByEmails}
}

func (svc *usersServiceClientMock) GetUsersByIDs(_ context.Context, ids []string, pm domainusers.PageMetadata) (domainusers.UsersPage, error) {
	if pm.Limit == 0 {
		pm.Limit = uint64(len(ids))
	}

	page := domainusers.UsersPage{
		Total: 0,
		Users: []domainusers.User{},
	}

	i := uint64(0)
	for _, id := range ids {
		if user, ok := svc.usersByID[id]; ok {
			if pm.Email != "" && !strings.Contains(user.Email, pm.Email) {
				continue
			}

			if i >= pm.Offset && i < pm.Offset+pm.Limit {
				page.Users = append(page.Users, domainusers.User{ID: user.ID, Email: user.Email, Status: user.Status})
			}
			i++
		}
	}
	page.Total = i

	return page, nil
}

func (svc *usersServiceClientMock) GetUsersByEmails(_ context.Context, emails []string) ([]domainusers.User, error) {
	var result []domainusers.User
	for _, email := range emails {
		if user, ok := svc.usersByEmails[email]; ok {
			result = append(result, domainusers.User{ID: user.ID, Email: user.Email, Status: user.Status})
		}
	}

	return result, nil
}
