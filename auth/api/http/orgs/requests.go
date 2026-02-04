package orgs

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	maxLimitSize = 200
	maxNameSize  = 254
)

type createOrgsReq struct {
	token       string
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func (req createOrgsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if len(req.Name) > maxNameSize || req.Name == "" {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateOrgReq struct {
	token       string
	id          string
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func (req updateOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingOrgID
	}

	return nil
}

type listOrgsReq struct {
	token        string
	pageMetadata apiutil.PageMetadata
}

func (req listOrgsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if len(req.pageMetadata.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if req.pageMetadata.Order != "" &&
		req.pageMetadata.Order != apiutil.NameOrder && req.pageMetadata.Order != apiutil.IDOrder {
		return apiutil.ErrInvalidOrder
	}

	if req.pageMetadata.Dir != "" &&
		req.pageMetadata.Dir != apiutil.AscDir && req.pageMetadata.Dir != apiutil.DescDir {
		return apiutil.ErrInvalidDirection
	}

	return nil
}

type orgReq struct {
	token string
	id    string
}

func (req orgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingOrgID
	}

	return nil
}

type deleteOrgsReq struct {
	token  string
	OrgIDs []string `json:"org_ids,omitempty"`
}

func (req deleteOrgsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.OrgIDs) < 1 {
		return apiutil.ErrEmptyList
	}

	for _, orgID := range req.OrgIDs {
		if orgID == "" {
			return apiutil.ErrMissingOrgID
		}
	}

	return nil
}
