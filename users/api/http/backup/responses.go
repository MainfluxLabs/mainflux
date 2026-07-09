package backup

import "net/http"

type backupUserRes struct {
	ID       string         `json:"id"`
	Email    string         `json:"email"`
	Password string         `json:"password"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Status   string         `json:"status"`
}

type backupIdentityRes struct {
	UserID         string `json:"user_id"`
	Provider       string `json:"provider"`
	ProviderUserID string `json:"provider_user_id"`
}

type backupRes struct {
	Users      []backupUserRes     `json:"users"`
	Admin      backupUserRes       `json:"admin"`
	Identities []backupIdentityRes `json:"identities"`
}

func (res backupRes) Code() int {
	return http.StatusOK
}

func (res backupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res backupRes) Empty() bool {
	return false
}
