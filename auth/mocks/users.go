package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/users"
	"google.golang.org/grpc"
)

var _ mainflux.UsersServiceClient = (*usersServiceClientMock)(nil)

type usersServiceClientMock struct {
	usersByID     map[string]users.User
	usersByEmails map[string]users.User
}

func NewUsersService(usersByID map[string]users.User, usersByEmails map[string]users.User) mainflux.UsersServiceClient {
	return &usersServiceClientMock{usersByID: usersByID, usersByEmails: usersByEmails}
}

func (svc *usersServiceClientMock) GetUsersByIDs(ctx context.Context, in *mainflux.UsersByIDsReq, opts ...grpc.CallOption) (*mainflux.UsersRes, error) {
	var users []*mainflux.User
	for _, id := range in.Ids {
		if user, ok := svc.usersByID[id]; ok {
			users = append(users, &mainflux.User{Id: user.ID, Email: user.Email})
		}
	}

	return &mainflux.UsersRes{Users: users}, nil
}

func (svc *usersServiceClientMock) GetUsersByEmails(ctx context.Context, in *mainflux.UsersByEmailsReq, opts ...grpc.CallOption) (*mainflux.UsersRes, error) {
	var users []*mainflux.User
	for _, email := range in.Emails {
		if user, ok := svc.usersByEmails[email]; ok {
			users = append(users, &mainflux.User{Id: user.ID, Email: user.Email})
		}
	}

	return &mainflux.UsersRes{Users: users}, nil
}
