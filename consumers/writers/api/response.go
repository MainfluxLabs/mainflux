package api

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux"
)

var _ mainflux.Response = (*restoreMessagesRes)(nil)

type restoreMessagesRes struct{}

func (res restoreMessagesRes) Code() int {
	return http.StatusCreated
}

func (res restoreMessagesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res restoreMessagesRes) Empty() bool {
	return true
}
