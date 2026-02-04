package backup

import "github.com/MainfluxLabs/mainflux/pkg/apiutil"

type backupReq struct {
	token string
}

func (req backupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}

type restoreUserReq struct {
	ID       string         `json:"id"`
	Email    string         `json:"email"`
	Password string         `json:"password"`
	Metadata map[string]any `json:"metadata"`
	Status   string
}
type restoreReq struct {
	token string
	Users []restoreUserReq `json:"users"`
	Admin restoreUserReq   `json:"admin"`
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Users) == 0 {
		return apiutil.ErrEmptyList
	}

	if req.Admin.ID == "" {
		return apiutil.ErrMissingUserID
	}

	return nil
}
