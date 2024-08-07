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

	// ErrFailedConnect indicates that connecting thing to channel failed.
	ErrFailedConnect = errors.New("failed to connect thing to channel")

	// ErrFailedDisconnect indicates that disconnecting thing from a channel failed.
	ErrFailedDisconnect = errors.New("failed to disconnect thing from channel")

	// ErrFailedPublish indicates that publishing message failed.
	ErrFailedPublish = errors.New("failed to publish message")

	// ErrFailedRead indicates that read messages failed.
	ErrFailedRead = errors.New("failed to read messages")

	// ErrInvalidContentType indicates that non-existent message content type
	// was passed.
	ErrInvalidContentType = errors.New("Unknown Content Type")

	// ErrFetchHealth indicates that fetching of health check failed.
	ErrFetchHealth = errors.New("failed to fetch health check")

	// ErrFailedWhitelist failed to whitelist configs
	ErrFailedWhitelist = errors.New("failed to whitelist")

	// ErrCerts indicates error fetching certificates.
	ErrCerts = errors.New("failed to fetch certs data")

	// ErrCertsRemove indicates failure while cleaning up from the Certs service.
	ErrCertsRemove = errors.New("failed to remove certificate")

	// ErrMemberAdd failed to add member to a group.
	ErrMemberAdd = errors.New("failed to add member to group")
)

// ContentType represents all possible content types.
type ContentType string

var _ SDK = (*mfSDK)(nil)

// User represents mainflux user its credentials.
type User struct {
	ID       string                 `json:"id,omitempty"`
	Email    string                 `json:"email,omitempty"`
	Groups   []string               `json:"groups,omitempty"`
	Password string                 `json:"password,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
type PageMetadata struct {
	Total    uint64                 `json:"total"`
	Offset   uint64                 `json:"offset"`
	Limit    uint64                 `json:"limit"`
	Level    uint64                 `json:"level,omitempty"`
	Email    string                 `json:"email,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Type     string                 `json:"type,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Group represents mainflux users group.
type Group struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	OwnerID     string                 `json:"owner_id,omitempty"`
	OrgID       string                 `json:"org_id,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty"`
}

// Thing represents mainflux thing.
type Thing struct {
	ID       string                 `json:"id,omitempty"`
	GroupID  string                 `json:"group_id,omitempty"`
	OwnerID  string                 `json:"owner_id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Channel represents mainflux channel.
type Channel struct {
	ID       string                 `json:"id,omitempty"`
	GroupID  string                 `json:"group_id,omitempty"`
	OwnerID  string                 `json:"owner_id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Profile  map[string]interface{} `json:"profile,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Org represents mainflux org.
type Org struct {
	ID          string                 `json:"id,omitempty"`
	OwnerID     string                 `json:"owner_id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty"`
}

// OrgMember represents mainflux Org Member.
type OrgMember struct {
	MemberID  string    `json:"member_id,omitempty"`
	OrgID     string    `json:"org_id,omitempty"`
	Role      string    `json:"role,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	Email     string    `json:"email,omitempty"`
}

// GroupMember represents mainflux Group Member.
type GroupMember struct {
	ID    string `json:"id,omitempty"`
	Role  string `json:"role,omitempty"`
	Email string `json:"email,omitempty"`
}

// Webhook represents mainflux Webhook.
type Webhook struct {
	ID      string            `json:"id"`
	GroupID string            `json:"group_id"`
	Name    string            `json:"name"`
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type Key struct {
	ID        string
	Type      uint32
	IssuerID  string
	Subject   string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// SDK contains Mainflux API.
type SDK interface {
	// CreateUser creates mainflux user.
	CreateUser(token string, user User) (string, error)

	// User returns user object by id.
	User(token, id string) (User, error)

	// Users returns list of users.
	Users(token string, pm PageMetadata) (UsersPage, error)

	// CreateToken receives credentials and returns user token.
	CreateToken(user User) (string, error)

	// RegisterUser registers mainflux user.
	RegisterUser(user User) (string, error)

	// UpdateUser updates existing user.
	UpdateUser(user User, token string) error

	// UpdatePassword updates user password.
	UpdatePassword(oldPass, newPass, token string) error

	// CreateThing registers new thing and returns its id.
	CreateThing(thing Thing, groupID, token string) (string, error)

	// CreateThings registers new things and returns their ids.
	CreateThings(things []Thing, groupID, token string) ([]Thing, error)

	// Things returns page of things.
	Things(token string, pm PageMetadata) (ThingsPage, error)

	// ThingsByChannel returns page of things that are connected to specified channel.
	ThingsByChannel(token, chanID string, offset, limit uint64) (ThingsPage, error)

	// Thing returns thing object by id.
	Thing(id, token string) (Thing, error)

	// UpdateThing updates existing thing.
	UpdateThing(thing Thing, thingID, token string) error

	// DeleteThing removes existing thing.
	DeleteThing(id, token string) error

	// DeleteThings removes existing things.
	DeleteThings(ids []string, token string) error

	// IdentifyThing validates thing's key and returns its ID
	IdentifyThing(key string) (string, error)

	// CreateGroup creates new group and returns its id.
	CreateGroup(group Group, orgID, token string) (string, error)

	// DeleteGroup deletes users group.
	DeleteGroup(id, token string) error

	// DeleteGroups delete users groups.
	DeleteGroups(ids []string, token string) error

	// Groups returns page of groups.
	Groups(meta PageMetadata, token string) (GroupsPage, error)

	// Group returns users group object by id.
	Group(id, token string) (Group, error)

	// ListThingsByGroup lists things that are members of specified group.
	ListThingsByGroup(groupID, token string, offset, limit uint64) (ThingsPage, error)

	// ViewGroupByThing retrieves a group that the specified thing is a member of.
	ViewGroupByThing(thingID, token string) (Group, error)

	// UpdateGroup updates existing group.
	UpdateGroup(group Group, groupID, token string) error

	// Connect connects a list of things to a channel.
	Connect(conns ConnectionIDs, token string) error

	// Disconnect disconnects a list of things from a channel.
	Disconnect(conns ConnectionIDs, token string) error

	// CreateChannel creates new channel and returns its id.
	CreateChannel(channel Channel, groupID, token string) (string, error)

	// CreateChannels registers new channels and returns their ids.
	CreateChannels(channels []Channel, groupID, token string) ([]Channel, error)

	// Channels returns page of channels.
	Channels(token string, pm PageMetadata) (ChannelsPage, error)

	// ViewChannelByThing returns channel that are connected to specified thing.
	ViewChannelByThing(token, thingID string) (Channel, error)

	// Channel returns channel data by id.
	Channel(id, token string) (Channel, error)

	// UpdateChannel updates existing channel.
	UpdateChannel(channel Channel, channelID, token string) error

	// DeleteChannel removes existing channel.
	DeleteChannel(id, token string) error

	// DeleteChannels removes existing channel.
	DeleteChannels(ids []string, token string) error

	// ListChannelsByGroup lists channels that are members of specified group.
	ListChannelsByGroup(groupID, token string, offset, limit uint64) (ChannelsPage, error)

	// ViewGroupByChannel retrieves a group that the specified channel is a member of.
	ViewGroupByChannel(channelID, token string) (Group, error)

	// CreateRolesByGroup creates new roles by group.
	CreateRolesByGroup(roles []GroupMember, groupID, token string) error

	// UpdateRolesByGroup updates existing group roles.
	UpdateRolesByGroup(roles []GroupMember, groupID, token string) error

	// RemoveRolesByGroup removes existing group roles.
	RemoveRolesByGroup(ids []string, groupID, token string) error

	// ListRolesByGroup lists roles that are specified for a certain group.
	ListRolesByGroup(groupID, token string, offset, limit uint64) (GroupRolesPage, error)

	// CreateOrg registers new org.
	CreateOrg(org Org, token string) error

	// Org returns org data by id.
	Org(id, token string) (Org, error)

	// UpdateOrg updates existing org.
	UpdateOrg(o Org, orgID, token string) error

	// DeleteOrg removes existing org.
	DeleteOrg(id, token string) error

	// Orgs returns page of orgs.
	Orgs(meta PageMetadata, token string) (OrgsPage, error)

	// ViewMember retrieves a member belonging to the specified org.
	ViewMember(orgID, memberID, token string) (Member, error)

	// AssignMembers assigns a members to the specified org.
	AssignMembers(om []OrgMember, orgID, token string) error

	// UnassignMembers unassigns a members from the specified org.
	UnassignMembers(token, orgID string, memberIDs ...string) error

	// UpdateMembers updates existing member.
	UpdateMembers(members []OrgMember, orgID, token string) error

	// ListMembersByOrg lists members who belong to a specified org.
	ListMembersByOrg(orgID, token string, offset, limit uint64) (MembersPage, error)

	// ListOrgsByMember lists orgs to which the specified member belongs.
	ListOrgsByMember(memberID, token string, offset, limit uint64) (OrgsPage, error)

	// CreateWebhooks creates new webhooks.
	CreateWebhooks(whs []Webhook, groupID, token string) ([]Webhook, error)

	// ListWebhooksByGroup lists webhooks who belong to a specified group.
	ListWebhooksByGroup(groupID, token string) (Webhooks, error)

	// Webhook returns webhook data by id.
	Webhook(webhookID, token string) (Webhook, error)

	// UpdateWebhook updates existing webhook.
	UpdateWebhook(wh Webhook, webhookID, token string) error

	// DeleteWebhooks removes existing webhooks.
	DeleteWebhooks(ids []string, groupID, token string) error

	// SendMessage send message to specified channel.
	SendMessage(chanID, msg, token string) error

	// ReadMessages read messages of specified channel.
	ReadMessages(chanID, token string) (MessagesPage, error)

	// SetContentType sets message content type.
	SetContentType(ct ContentType) error

	// Health returns things service health check.
	Health() (mainflux.HealthInfo, error)

	// IssueCert issues a certificate for a thing required for mtls.
	IssueCert(thingID string, keyBits int, keyType, valid, token string) (Cert, error)

	// RemoveCert removes a certificate
	RemoveCert(id, token string) error

	// RevokeCert revokes certificate with certID for thing with thingID
	RevokeCert(thingID, certID, token string) error

	// Issue issues a new key, returning its token value alongside.
	Issue(token string, duration time.Duration) (KeyRes, error)

	// Revoke removes the key with the provided ID that is issued by the user identified by the provided key.
	Revoke(token, id string) error

	// RetrieveKey retrieves data for the key identified by the provided ID, that is issued by the user identified by the provided key.
	RetrieveKey(token, id string) (retrieveKeyRes, error)
}

type mfSDK struct {
	authURL        string
	bootstrapURL   string
	certsURL       string
	httpAdapterURL string
	readerURL      string
	thingsURL      string
	webhooksURL    string
	usersURL       string

	msgContentType ContentType
	client         *http.Client
}

// Config contains sdk configuration parameters.
type Config struct {
	AuthURL        string
	BootstrapURL   string
	CertsURL       string
	HTTPAdapterURL string
	ReaderURL      string
	ThingsURL      string
	WebhooksURL    string
	UsersURL       string

	MsgContentType  ContentType
	TLSVerification bool
}

// NewSDK returns new mainflux SDK instance.
func NewSDK(conf Config) SDK {
	return &mfSDK{
		authURL:        conf.AuthURL,
		bootstrapURL:   conf.BootstrapURL,
		certsURL:       conf.CertsURL,
		httpAdapterURL: conf.HTTPAdapterURL,
		readerURL:      conf.ReaderURL,
		thingsURL:      conf.ThingsURL,
		webhooksURL:    conf.WebhooksURL,
		usersURL:       conf.UsersURL,

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

func (sdk mfSDK) sendThingRequest(req *http.Request, key, contentType string) (*http.Response, error) {
	if key != "" {
		req.Header.Set("Authorization", apiutil.ThingPrefix+key)
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
	if pm.Level != 0 {
		q.Add("level", strconv.FormatUint(pm.Level, 10))
	}
	if pm.Email != "" {
		q.Add("email", pm.Email)
	}
	if pm.Name != "" {
		q.Add("name", pm.Name)
	}
	if pm.Type != "" {
		q.Add("type", pm.Type)
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
