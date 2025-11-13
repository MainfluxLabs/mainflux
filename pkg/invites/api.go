package invites

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const StateKey = "state"

func BuildPageMetadataInvites(r *http.Request) (PageMetadataInvites, error) {
	pm := PageMetadataInvites{}

	apm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return PageMetadataInvites{}, err
	}

	pm.PageMetadata = apm

	state, err := apiutil.ReadStringQuery(r, StateKey, "")
	if err != nil {
		return PageMetadataInvites{}, err
	}

	pm.State = state

	return pm, nil
}

type PageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
	Ord    string `json:"order,omitempty"`
	Dir    string `json:"direction,omitempty"`
	State  string `json:"state,omitempty"`
}
