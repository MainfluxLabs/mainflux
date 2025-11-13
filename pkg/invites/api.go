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
