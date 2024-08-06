package mocks

import (
	"context"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/users"
	"google.golang.org/grpc"
)

var _ protomfx.UsersServiceClient = (*usersServiceClientMock)(nil)

type usersServiceClientMock struct {
	usersByID     map[string]users.User
	usersByEmails map[string]users.User
}

func NewUsersService(usersByID map[string]users.User, usersByEmails map[string]users.User) protomfx.UsersServiceClient {
	return &usersServiceClientMock{usersByID: usersByID, usersByEmails: usersByEmails}
}

func (svc *usersServiceClientMock) GetUsersByIDs(ctx context.Context, in *protomfx.UsersByIDsReq, opts ...grpc.CallOption) (*protomfx.UsersRes, error) {
	var users []*protomfx.User
	for _, id := range in.Ids {
		if user, ok := svc.usersByID[id]; ok {
			users = append(users, &protomfx.User{Id: user.ID, Email: user.Email})
		}
	}

	return &protomfx.UsersRes{Users: users}, nil
}

func (svc *usersServiceClientMock) GetUsersByEmails(ctx context.Context, in *protomfx.UsersByEmailsReq, opts ...grpc.CallOption) (*protomfx.UsersRes, error) {
	var users []*protomfx.User
	for _, email := range in.Emails {
		if user, ok := svc.usersByEmails[email]; ok {
			users = append(users, &protomfx.User{Id: user.ID, Email: user.Email})
		}
	}

	return &protomfx.UsersRes{Users: users}, nil
}
