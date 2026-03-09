// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
)

const (
	// CTJSON represents JSON content type.
	CTJSON ContentType = "application/json"

	// CTJSONSenML represents JSON SenML content type.
	CTJSONSenML ContentType = "application/senml+json"

	// CTBinary represents binary content type.
	CTBinary ContentType = "application/octet-stream"
)

var (
	// ErrFailedCreation indicates that entity creation failed.
	ErrFailedCreation = errors.New("failed to create entity")

	// ErrFailedUpdate indicates that entity update failed.
	ErrFailedUpdate = errors.New("failed to update entity")

	// ErrFailedFetch indicates that fetching of entity data failed.
	ErrFailedFetch = errors.New("failed to fetch entity")

	// ErrFailedRemoval indicates that entity removal failed.
	ErrFailedRemoval = errors.New("failed to remove entity")

	// ErrFailedPublish indicates that publishing message failed.
	ErrFailedPublish = errors.New("failed to publish message")

	// ErrFailedRead indicates that read messages failed.
	ErrFailedRead = errors.New("failed to read messages")

	// ErrInvalidContentType indicates that non-existent message content type
	// was passed.
	ErrInvalidContentType = errors.New("unknown content-type")

	// ErrFetchHealth indicates that fetching of health check failed.
	ErrFetchHealth = errors.New("failed to fetch health check")

	// ErrCerts indicates error fetching certificates.
	ErrCerts = errors.New("failed to fetch certs data")

	// ErrCertsRemove indicates failure while cleaning up from the Certs service.
	ErrCertsRemove = errors.New("failed to remove certificate")
)

// ContentType represents all possible content types.
type ContentType string

var _ SDK = (*mfSDK)(nil)

// User represents mainflux user its credentials.
type User struct {
	ID       string         `json:"id,omitempty"`
	Email    string         `json:"email,omitempty"`
	Groups   []string       `json:"groups,omitempty"`
	Password string         `json:"password,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}
type PageMetadata struct {
	Total    uint64         `json:"total"`
	Offset   uint64         `json:"offset"`
	Limit    uint64         `json:"limit"`
	Order    string         `json:"order"`
	Dir      string         `json:"dir"`
	Subtopic string         `json:"subtopic,omitempty"`
	Format   string         `json:"format,omitempty"`
	Email    string         `json:"email,omitempty"`
	Name     string         `json:"name,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// JSONPageMetadata represents query parameters for JSON message endpoints.
type JSONPageMetadata struct {
	Offset      uint64   `json:"offset"`
	Limit       uint64   `json:"limit"`
	Subtopic    string   `json:"subtopic,omitempty"`
	Publisher   string   `json:"publisher,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
	From        int64    `json:"from,omitempty"`
	To          int64    `json:"to,omitempty"`
	Filter      string   `json:"filter,omitempty"`
	AggInterval string   `json:"agg_interval,omitempty"`
	AggValue    uint64   `json:"agg_value,omitempty"`
	AggType     string   `json:"agg_type,omitempty"`
	AggFields   []string `json:"agg_fields,omitempty"`
	Dir         string   `json:"dir,omitempty"`
}

// SenMLPageMetadata represents query parameters for SenML message endpoints.
type SenMLPageMetadata struct {
	Offset      uint64   `json:"offset"`
	Limit       uint64   `json:"limit"`
	Subtopic    string   `json:"subtopic,omitempty"`
	Publisher   string   `json:"publisher,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
	Name        string   `json:"name,omitempty"`
	Value       float64  `json:"v,omitempty"`
	Comparator  string   `json:"comparator,omitempty"`
	BoolValue   bool     `json:"vb,omitempty"`
	StringValue string   `json:"vs,omitempty"`
	DataValue   string   `json:"vd,omitempty"`
	From        int64    `json:"from,omitempty"`
	To          int64    `json:"to,omitempty"`
	AggInterval string   `json:"agg_interval,omitempty"`
	AggValue    uint64   `json:"agg_value,omitempty"`
	AggType     string   `json:"agg_type,omitempty"`
	AggFields   []string `json:"agg_fields,omitempty"`
	Dir         string   `json:"dir,omitempty"`
}

// Group represents mainflux users group.
type Group struct {
	ID          string         `json:"id,omitempty"`
	Name        string         `json:"name,omitempty"`
	OwnerID     string         `json:"owner_id,omitempty"`
	OrgID       string         `json:"org_id,omitempty"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at,omitempty"`
	UpdatedAt   time.Time      `json:"updated_at,omitempty"`
}

// Thing represents mainflux thing.
type Thing struct {
	ID        string         `json:"id,omitempty"`
	GroupID   string         `json:"group_id,omitempty"`
	ProfileID string         `json:"profile_id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Key       string         `json:"key,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Profile represents mainflux profile.
type Profile struct {
	ID       string         `json:"id,omitempty"`
	GroupID  string         `json:"group_id,omitempty"`
	Name     string         `json:"name,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Org represents mainflux org.
type Org struct {
	ID          string         `json:"id,omitempty"`
	OwnerID     string         `json:"owner_id,omitempty"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at,omitempty"`
	UpdatedAt   time.Time      `json:"updated_at,omitempty"`
}

// OrgMembership represents mainflux Org Membership.
type OrgMembership struct {
	MemberID  string    `json:"member_id,omitempty"`
	OrgID     string    `json:"org_id,omitempty"`
	Role      string    `json:"role,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	Email     string    `json:"email,omitempty"`
}

// GroupMembership represents mainflux Group Membership.
type GroupMembership struct {
	MemberID string `json:"member_id,omitempty"`
	Role     string `json:"role,omitempty"`
	Email    string `json:"email,omitempty"`
}

type Key struct {
	ID        string
	Type      uint32
	IssuerID  string
	Subject   string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type Invite struct {
	ID           string    `json:"id"`
	InviteeID    string    `json:"invitee_id"`
	InviteeEmail string    `json:"invitee_email"`
	InviterID    string    `json:"inviter_id"`
	OrgID        string    `json:"org_id"`
	InviteeRole  string    `json:"invitee_role"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type Webhook struct {
	ID      string            `json:"id"`
	GroupID string            `json:"group_id"`
	Name    string            `json:"name"`
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type Metadata map[string]any

// SDK contains Mainflux API.
type SDK interface {
	// CreateUser creates mainflux user.
	CreateUser(user User, token string) (string, error)

	// GetUser returns user object by id.
	GetUser(id, token string) (User, error)

	// ListUsers returns list of users.
	ListUsers(pm PageMetadata, token string) (UsersPage, error)

	// CreateToken receives credentials and returns user token.
	CreateToken(user User) (string, error)

	// RegisterUser registers mainflux user.
	RegisterUser(user User) (string, error)

	// UpdateUser updates existing user.
	UpdateUser(user User, token string) error

	// UpdatePassword updates user password.
	UpdatePassword(oldPass, newPass, token string) error

	// CreateThing registers new thing and returns its id.
	CreateThing(thing Thing, profileID, token string) (string, error)

	// CreateThings registers new things and returns their ids.
	CreateThings(things []Thing, profileID, token string) ([]Thing, error)

	// ListThings returns page of things.
	ListThings(pm PageMetadata, token string) (ThingsPage, error)

	// ListThingsByProfile returns page of things assigned to the specified profile.
	ListThingsByProfile(profileID string, pm PageMetadata, token string) (ThingsPage, error)

	// GetThing returns thing object by id.
	GetThing(id, token string) (Thing, error)

	// GetThingMetadataByKey retrieves metadata about the thing identified by the given key.
	GetThingMetadataByKey(key things.ThingKey) (Metadata, error)

	// UpdateThing updates existing thing.
	UpdateThing(thing Thing, thingID, token string) error

	// DeleteThing removes existing thing.
	DeleteThing(id, token string) error

	// DeleteThings removes existing things.
	DeleteThings(ids []string, token string) error

	// IdentifyThing validates thing's key and returns its ID
	IdentifyThing(key things.ThingKey) (string, error)

	// UpdateExternalThingKey sets the external key of the Thing identified by `thingID`.`
	UpdateExternalThingKey(key, thingID, token string) error

	// CreateGroup creates new group and returns its id.
	CreateGroup(group Group, orgID, token string) (string, error)

	// DeleteGroup deletes users group.
	DeleteGroup(id, token string) error

	// DeleteGroups delete users groups.
	DeleteGroups(ids []string, token string) error

	// ListGroups returns page of groups.
	ListGroups(meta PageMetadata, token string) (GroupsPage, error)

	// ListGroupsByOrg returns a page of all Groups belonging to the spcified Org.
	ListGroupsByOrg(orgID string, meta PageMetadata, token string) (GroupsPage, error)

	// GetGroup returns users group object by id.
	GetGroup(id, token string) (Group, error)

	// ListThingsByGroup lists things that are members of specified group.
	ListThingsByGroup(groupID string, meta PageMetadata, token string) (ThingsPage, error)

	// GetGroupByThing retrieves a group that the specified thing is a member of.
	GetGroupByThing(thingID, token string) (Group, error)

	// UpdateGroup updates existing group.
	UpdateGroup(group Group, groupID, token string) error

	// CreateProfile creates new profile and returns its id.
	CreateProfile(profile Profile, groupID, token string) (string, error)

	// CreateProfiles registers new profiles and returns their ids.
	CreateProfiles(profiles []Profile, groupID, token string) ([]Profile, error)

	// ListProfiles returns page of profiles.
	ListProfiles(pm PageMetadata, token string) (ProfilesPage, error)

	// GetProfileByThing returns profile that are assigned to specified thing.
	GetProfileByThing(thingID, token string) (Profile, error)

	// GetProfile returns profile data by id.
	GetProfile(id, token string) (Profile, error)

	// UpdateProfile updates existing profile.
	UpdateProfile(profile Profile, profileID, token string) error

	// DeleteProfile removes existing profile.
	DeleteProfile(id, token string) error

	// DeleteProfiles removes existing profile.
	DeleteProfiles(ids []string, token string) error

	// ListProfilesByGroup lists profiles that are members of specified group.
	ListProfilesByGroup(groupID string, pm PageMetadata, token string) (ProfilesPage, error)

	// GetGroupByProfile retrieves a group that the specified profile is a member of.
	GetGroupByProfile(profileID, token string) (Group, error)

	// CreateGroupMemberships creates memberships to the specified group.
	CreateGroupMemberships(gms []GroupMembership, groupID, token string) error

	// UpdateGroupMemberships updates existing memberships.
	UpdateGroupMemberships(gms []GroupMembership, groupID, token string) error

	// RemoveGroupMemberships removes memberships from the specified group.
	RemoveGroupMemberships(ids []string, groupID, token string) error

	// ListGroupMemberships lists memberships created for a specified group.
	ListGroupMemberships(groupID string, pm PageMetadata, token string) (GroupMembershipsPage, error)

	// CreateOrg registers a new Org and returns its ID.
	CreateOrg(org Org, token string) (string, error)

	// GetOrg returns org data by id.
	GetOrg(id, token string) (Org, error)

	// UpdateOrg updates existing org.
	UpdateOrg(o Org, orgID, token string) error

	// DeleteOrg removes existing org.
	DeleteOrg(id, token string) error

	// ListOrgs returns page of orgs.
	ListOrgs(meta PageMetadata, token string) (OrgsPage, error)

	// CreateOrgMemberships creates memberships to the specified org.
	CreateOrgMemberships(oms []OrgMembership, orgID, token string) error

	// GetOrgMembership retrieves a membership for the specified org and member.
	GetOrgMembership(memberID, orgID, token string) (OrgMembership, error)

	// ListOrgMemberships lists memberships created for a specified org.
	ListOrgMemberships(orgID string, meta PageMetadata, token string) (OrgMembershipsPage, error)

	// UpdateOrgMemberships updates existing memberships.
	UpdateOrgMemberships(oms []OrgMembership, orgID, token string) error

	// RemoveOrgMemberships removes memberships from the specified org.
	RemoveOrgMemberships(memberIDs []string, orgID, token string) error

	// SendMessage send message.
	SendMessage(subtopic, msg string, key things.ThingKey) error

	// ReadMessages read messages.
	ReadMessages(isAdmin bool, pm PageMetadata, keyType, token string) (map[string]any, error)

	// ListJSONMessages lists JSON messages with filtering.
	ListJSONMessages(pm JSONPageMetadata, token string, key things.ThingKey) (map[string]any, error)

	// ListSenMLMessages lists SenML messages with filtering.
	ListSenMLMessages(pm SenMLPageMetadata, token string, key things.ThingKey) (map[string]any, error)

	// DeleteJSONMessages deletes JSON messages by publisher.
	DeleteJSONMessages(publisherID, token string, pm JSONPageMetadata) error

	// DeleteSenMLMessages deletes SenML messages by publisher.
	DeleteSenMLMessages(publisherID, token string, pm SenMLPageMetadata) error

	// DeleteAllJSONMessages deletes all JSON messages (admin only).
	DeleteAllJSONMessages(token string, pm JSONPageMetadata) error

	// DeleteAllSenMLMessages deletes all SenML messages (admin only).
	DeleteAllSenMLMessages(token string, pm SenMLPageMetadata) error

	// ExportJSONMessages exports JSON messages.
	ExportJSONMessages(token string, pm JSONPageMetadata, convert, timeFormat string) ([]byte, error)

	// ExportSenMLMessages exports SenML messages.
	ExportSenMLMessages(token string, pm SenMLPageMetadata, convert, timeFormat string) ([]byte, error)

	// BackupMessages backs up all messages (admin only).
	BackupMessages(token string) ([]byte, error)

	// RestoreMessages restores messages from backup data (admin only).
	RestoreMessages(token string, data []byte) error

	// ValidateContentType sets message content type.
	ValidateContentType(ct ContentType) error

	// Health returns things service health check.
	Health() (mainflux.HealthInfo, error)

	// IssueCert issues a certificate for a thing required for mtls.
	IssueCert(thingID string, keyBits int, keyType, valid, token string) (Cert, error)

	// RemoveCert removes a certificate
	RemoveCert(id, token string) error

	// RevokeCert revokes certificate with certID for thing with thingID
	RevokeCert(thingID, certID, token string) error

	// Issue issues a new key, returning its token value alongside.
	Issue(duration time.Duration, token string) (KeyRes, error)

	// Revoke removes the key with the provided ID that is issued by the user identified by the provided key.
	Revoke(id, token string) error

	// RetrieveKey retrieves data for the key identified by the provided ID, that is issued by the user identified by the provided key.
	RetrieveKey(id, token string) (retrieveKeyRes, error)

	// CreateInvite creates and sends a new Invite.
	CreateInvite(orgID string, om OrgMembership, token string) (Invite, error)

	// RevokeInvite revokes a specific Invite.
	RevokeInvite(inviteID string, token string) error

	// InviteRespond responds to a specific Invite either accepting or declining it.
	InviteRespond(inviteID string, accept bool, token string) error

	// GetInvite retrieves a specific Invite.
	GetInvite(inviteID string, token string) (Invite, error)

	// ListInvitesByUser retrieves a list of Invites either sent out by, or sent to the user identifed by
	// the specific userID.
	ListInvitesByUser(userID string, userType string, pm PageMetadata, token string) (InvitesPage, error)

	// CreateWebhooks creates new webhooks.
	CreateWebhooks(whs []Webhook, groupID, token string) ([]Webhook, error)

	// ListWebhooksByGroup lists webhooks who belong to a specified group.
	ListWebhooksByGroup(groupID, token string) (WebhooksPage, error)

	// ListWebhooksByThing lists webhooks who belong to a specified thing.
	ListWebhooksByThing(thingID, token string) (WebhooksPage, error)

	// GetWebhook returns webhook data by id.
	GetWebhook(webhookID, token string) (Webhook, error)

	// UpdateWebhook updates existing webhook.
	UpdateWebhook(wh Webhook, webhookID, token string) error

	// DeleteWebhooks removes existing webhooks.
	DeleteWebhooks(ids []string, token string) error
}

type mfSDK struct {
	authURL        string
	certsURL       string
	httpAdapterURL string
	readerURL      string
	thingsURL      string
	usersURL       string
	webhooksURL    string

	msgContentType ContentType
	client         *http.Client
}

// Config contains sdk configuration parameters.
type Config struct {
	AuthURL        string
	CertsURL       string
	HTTPAdapterURL string
	ReaderURL      string
	ThingsURL      string
	UsersURL       string
	WebhooksURL    string

	MsgContentType  ContentType
	TLSVerification bool
}

// NewSDK returns new mainflux SDK instance.
func NewSDK(conf Config) SDK {
	return &mfSDK{
		authURL:        conf.AuthURL,
		certsURL:       conf.CertsURL,
		httpAdapterURL: conf.HTTPAdapterURL,
		readerURL:      conf.ReaderURL,
		thingsURL:      conf.ThingsURL,
		usersURL:       conf.UsersURL,
		webhooksURL:    conf.WebhooksURL,

		msgContentType: conf.MsgContentType,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: !conf.TLSVerification,
				},
			},
		},
	}
}

func (sdk mfSDK) sendRequest(req *http.Request, token, contentType string) (*http.Response, error) {
	if token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+token)
	}

	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}

	return sdk.client.Do(req)
}

func (sdk mfSDK) sendThingRequest(req *http.Request, key things.ThingKey, contentType string) (*http.Response, error) {
	if key.Value != "" {
		switch key.Type {
		case things.KeyTypeInternal:
			req.Header.Set("Authorization", apiutil.ThingKeyPrefixInternal+key.Value)
		case things.KeyTypeExternal:
			req.Header.Set("Authorization", apiutil.ThingKeyPrefixExternal+key.Value)
		}
	}

	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}

	return sdk.client.Do(req)
}

func (sdk mfSDK) withQueryParams(baseURL, endpoint string, pm PageMetadata) (string, error) {
	q, err := pm.query()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s?%s", baseURL, endpoint, q), nil
}

func (pm PageMetadata) query() (string, error) {
	q := url.Values{}
	q.Add("total", strconv.FormatUint(pm.Total, 10))
	q.Add("offset", strconv.FormatUint(pm.Offset, 10))
	q.Add("limit", strconv.FormatUint(pm.Limit, 10))
	if pm.Email != "" {
		q.Add("email", pm.Email)
	}
	if pm.Name != "" {
		q.Add("name", pm.Name)
	}
	if pm.Subtopic != "" {
		q.Add("subtopic", pm.Subtopic)
	}
	if pm.Format != "" {
		q.Add("format", pm.Format)
	}
	if pm.Order != "" {
		q.Add("order", pm.Order)
	}
	if pm.Dir != "" {
		q.Add("dir", pm.Dir)
	}
	if pm.Metadata != nil {
		md, err := json.Marshal(pm.Metadata)
		if err != nil {
			return "", err
		}
		q.Add("metadata", string(md))
	}
	return q.Encode(), nil
}
