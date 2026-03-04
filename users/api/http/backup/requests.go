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

type restoreIdentityReq struct {
	UserID         string `json:"user_id"`
	Provider       string `json:"provider"`
	ProviderUserID string `json:"provider_user_id"`
}

type restoreReq struct {
	token      string
	Users      []restoreUserReq     `json:"users"`
	Admin      restoreUserReq       `json:"admin"`
	Identities []restoreIdentityReq `json:"identities"`
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
