package api

import (
	"time"

	"github.com/MainfluxLabs/mainflux/internal/apiutil"
)

type message struct {
	ID           string    `json:"id"`
	Channel      string    `json:"channel"`
	Subtopic     string    `json:"subtopic"`
	Publisher    string    `json:"publisher"`
	Protocol     string    `json:"protocol"`
	Name         string    `json:"name"`
	Unit         string    `json:"unit"`
	Value        float64   `json:"value"`
	String_value string    `json:"string_value"`
	Bool_value   bool      `json:"bool_value"`
	Data_value   []byte    `json:"data_value"`
	Sum          float64   `json:"sum"`
	Time         time.Time `json:"time"`
	Update_time  time.Time `json:"update_time"`
}

type restoreMessagesReq struct {
	token    string
	Messages []message `json:"messages"`
}

func (req restoreMessagesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Messages) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}
