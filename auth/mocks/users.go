package mocks

import (
	"context"
	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/users"
	"google.golang.org/grpc"
)

var _ mainflux.UsersServiceClient = (*usersServiceClientMock)(nil)

type usersServiceClientMock struct {
	users map[string]users.User
}

func NewUsersService(users map[string]users.User) mainflux.UsersServiceClient {
	return &usersServiceClientMock{users: users}
}

func (svc *usersServiceClientMock) GetUsersByIDs(ctx context.Context, in *mainflux.UsersByIDsReq, opts ...grpc.CallOption) (*mainflux.UsersRes, error) {
	var users []*mainflux.User

	for _, id := range in.Ids {
		if user, ok := svc.users[id]; !ok {
			users = append(users, &mainflux.User{Id: user.ID, Email: user.Email})
		}

	}

	return &mainflux.UsersRes{Users: users}, nil
}

func (svc *usersServiceClientMock) GetUsersByEmails(ctx context.Context, in *mainflux.UsersByEmailsReq, opts ...grpc.CallOption) (*mainflux.UsersRes, error) {
	var users []*mainflux.User
	for _, email := range in.Emails {
		if user, ok := svc.users[email]; ok {
			users = append(users, &mainflux.User{Id: user.ID, Email: user.Email})
		}
	}

	return &mainflux.UsersRes{Users: users}, nil
}
