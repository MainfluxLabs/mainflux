package orgs

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux"
)

var (
	_ mainflux.Response = (*memberPageRes)(nil)
	_ mainflux.Response = (*orgRes)(nil)
	_ mainflux.Response = (*deleteRes)(nil)
	_ mainflux.Response = (*assignRes)(nil)
	_ mainflux.Response = (*unassignRes)(nil)
	_ mainflux.Response = (*backupRes)(nil)
	_ mainflux.Response = (*restoreRes)(nil)
	_ mainflux.Response = (*listGroupPoliciesRes)(nil)
	_ mainflux.Response = (*updateGroupPoliciesRes)(nil)
	_ mainflux.Response = (*createGroupPoliciesRes)(nil)
)

type viewMemberRes struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (res viewMemberRes) Code() int {
	return http.StatusOK
}

func (res viewMemberRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewMemberRes) Empty() bool {
	return false
}

type memberPageRes struct {
	pageRes
	Members []viewMemberRes `json:"members"`
}

func (res memberPageRes) Code() int {
	return http.StatusOK
}

func (res memberPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res memberPageRes) Empty() bool {
	return false
}

type viewGroupRes struct {
	ID          string `json:"id"`
	OwnerID     string `json:"owner_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type groupsPageRes struct {
	pageRes
	Groups []viewGroupRes `json:"groups"`
}

func (res groupsPageRes) Code() int {
	return http.StatusOK
}

func (res groupsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res groupsPageRes) Empty() bool {
	return false
}

type viewOrgRes struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	OwnerID     string                 `json:"owner_id"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

func (res viewOrgRes) Code() int {
	return http.StatusOK
}

func (res viewOrgRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewOrgRes) Empty() bool {
	return false
}

type orgRes struct {
	id      string
	created bool
}

func (res orgRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res orgRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/orgs/%s", res.id),
		}
	}

	return map[string]string{}
}

func (res orgRes) Empty() bool {
	return true
}

type orgsPageRes struct {
	pageRes
	Orgs []viewOrgRes `json:"orgs"`
}

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
	Name   string `json:"name"`
}

func (res orgsPageRes) Code() int {
	return http.StatusOK
}

func (res orgsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res orgsPageRes) Empty() bool {
	return false
}

type deleteRes struct{}

func (res deleteRes) Code() int {
	return http.StatusNoContent
}

func (res deleteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteRes) Empty() bool {
	return true
}

type assignRes struct{}

func (res assignRes) Code() int {
	return http.StatusOK
}

func (res assignRes) Headers() map[string]string {
	return map[string]string{}
}

func (res assignRes) Empty() bool {
	return true
}

type unassignRes struct{}

func (res unassignRes) Code() int {
	return http.StatusNoContent
}

func (res unassignRes) Headers() map[string]string {
	return map[string]string{}
}

func (res unassignRes) Empty() bool {
	return true
}

type viewOrgMembers struct {
	MemberID  string    `json:"member_id"`
	OrgID     string    `json:"org_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type viewOrgGroups struct {
	GroupID   string    `json:"group_id"`
	OrgID     string    `json:"org_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type viewGroupPolicies struct {
	GroupID  string `json:"group_id"`
	MemberID string `json:"member_id"`
	Policy   string `json:"policy"`
}

type backupRes struct {
	Orgs          []viewOrgRes        `json:"orgs"`
	OrgMembers    []viewOrgMembers    `json:"org_members"`
	OrgGroups     []viewOrgGroups     `json:"org_groups"`
	GroupPolicies []viewGroupPolicies `json:"group_policies"`
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

type groupPolicy struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Policy string `json:"policy"`
}

type listGroupPoliciesRes struct {
	pageRes
	GroupPolicies []groupPolicy `json:"group_policies"`
}

func (res listGroupPoliciesRes) Code() int {
	return http.StatusOK
}

func (res listGroupPoliciesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listGroupPoliciesRes) Empty() bool {
	return false
}

type restoreRes struct{}

func (res restoreRes) Code() int {
	return http.StatusCreated
}

func (res restoreRes) Headers() map[string]string {
	return map[string]string{}
}

func (res restoreRes) Empty() bool {
	return true
}

type createGroupPoliciesRes struct{}

func (res createGroupPoliciesRes) Code() int {
	return http.StatusCreated
}

func (res createGroupPoliciesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res createGroupPoliciesRes) Empty() bool {
	return true
}

type updateGroupPoliciesRes struct{}

func (res updateGroupPoliciesRes) Code() int {
	return http.StatusOK
}

func (res updateGroupPoliciesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateGroupPoliciesRes) Empty() bool {
	return true
}
