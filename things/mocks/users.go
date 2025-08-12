package mocks

import (
	"context"
	"strings"

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

func (svc *usersServiceClientMock) GetUsersByIDs(_ context.Context, req *protomfx.UsersByIDsReq, _ ...grpc.CallOption) (*protomfx.UsersRes, error) {
	if req.PageMetadata == nil {
		req.PageMetadata = &protomfx.PageMetadata{}
	}

	if req.PageMetadata.Limit == 0 {
		req.PageMetadata.Limit = uint64(len(req.Ids))
	}

	res := &protomfx.UsersRes{
		PageMetadata: &protomfx.PageMetadata{
			Limit:  req.PageMetadata.Limit,
			Offset: req.PageMetadata.Offset,
			Total:  req.PageMetadata.Total,
			Email:  req.PageMetadata.Email,
			Order:  req.PageMetadata.Order,
			Dir:    req.PageMetadata.Dir,
		},
	}

	i := uint64(0)
	for _, id := range req.Ids {
		if user, ok := svc.usersByID[id]; ok {
			if req.PageMetadata.Email != "" && !strings.Contains(user.Email, req.PageMetadata.Email) {
				continue
			}
			if i >= req.PageMetadata.Offset && i < req.PageMetadata.Offset+req.PageMetadata.Limit {
				res.Users = append(res.Users, &protomfx.User{
					Id:    user.ID,
					Email: user.Email,
				})
			}
			i++
		}
	}

	return res, nil
}

func (svc *usersServiceClientMock) GetUsersByEmails(_ context.Context, req *protomfx.UsersByEmailsReq, _ ...grpc.CallOption) (*protomfx.UsersRes, error) {
	var users []*protomfx.User
	for _, email := range req.Emails {
		if user, ok := svc.usersByEmails[email]; ok {
			users = append(users, &protomfx.User{Id: user.ID, Email: user.Email})
		}
	}

	return &protomfx.UsersRes{Users: users}, nil
}
