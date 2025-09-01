package mocks

import (
	"context"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/users"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ protomfx.UsersServiceClient = (*usersServiceClientMock)(nil)

type usersServiceClientMock struct {
	usersByID     map[string]users.User
	usersByEmails map[string]users.User
}

func NewUsersService(usersByID map[string]users.User, usersByEmails map[string]users.User) protomfx.UsersServiceClient {
	return &usersServiceClientMock{usersByID: usersByID, usersByEmails: usersByEmails}
}

func (svc *usersServiceClientMock) GetUsersByIDs(_ context.Context, in *protomfx.UsersByIDsReq, _ ...grpc.CallOption) (*protomfx.UsersRes, error) {
	if in.PageMetadata == nil {
		in.PageMetadata = &protomfx.PageMetadata{}
	}

	if in.PageMetadata.Limit == 0 {
		in.PageMetadata.Limit = uint64(len(in.Ids))
	}

	res := &protomfx.UsersRes{
		PageMetadata: &protomfx.PageMetadata{
			Limit:  in.PageMetadata.Limit,
			Offset: in.PageMetadata.Offset,
			Total:  in.PageMetadata.Total,
			Email:  in.PageMetadata.Email,
			Order:  in.PageMetadata.Order,
			Dir:    in.PageMetadata.Dir,
		},
	}

	i := uint64(0)
	for _, id := range in.Ids {
		if user, ok := svc.usersByID[id]; ok {
			if in.PageMetadata.Email != "" && !strings.Contains(user.Email, in.PageMetadata.Email) {
				continue
			}

			if i >= in.PageMetadata.Offset && i < in.PageMetadata.Offset+in.PageMetadata.Limit {
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

func (svc *usersServiceClientMock) GetUsersByEmails(_ context.Context, in *protomfx.UsersByEmailsReq, _ ...grpc.CallOption) (*protomfx.UsersRes, error) {
	var users []*protomfx.User
	for _, email := range in.Emails {
		if user, ok := svc.usersByEmails[email]; ok {
			users = append(users, &protomfx.User{Id: user.ID, Email: user.Email})
		} else {
			return nil, status.Error(codes.NotFound, dbutil.ErrNotFound.Error())
		}
	}

	return &protomfx.UsersRes{Users: users}, nil
}
