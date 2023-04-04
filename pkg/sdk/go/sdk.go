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
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
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

	// ErrFailedCertUpdate failed to update certs in bootstrap config
	ErrFailedCertUpdate = errors.New("failed to update certs in bootstrap config")

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
	Description string                 `json:"description,omitempty"`
	ParentID    string                 `json:"parent_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Thing represents mainflux thing.
type Thing struct {
	ID       string                 `json:"id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Channel represents mainflux channel.
type Channel struct {
	ID       string                 `json:"id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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
	CreateThing(thing Thing, token string) (string, error)

	// CreateThings registers new things and returns their ids.
	CreateThings(things []Thing, token string) ([]Thing, error)

	// Things returns page of things.
	Things(token string, pm PageMetadata) (ThingsPage, error)

	// ThingsByChannel returns page of things that are connected or not connected
	// to specified channel.
	ThingsByChannel(token, chanID string, offset, limit uint64, disconnected bool) (ThingsPage, error)

	// Thing returns thing object by id.
	Thing(id, token string) (Thing, error)

	// UpdateThing updates existing thing.
	UpdateThing(thing Thing, token string) error

	// DeleteThing removes existing thing.
	DeleteThing(id, token string) error

	// IdentifyThing validates thing's key and returns its ID
	IdentifyThing(key string) (string, error)

	// CreateGroup creates new group and returns its id.
	CreateGroup(group Group, token string) (string, error)

	// DeleteGroup deletes users group.
	DeleteGroup(id, token string) error

	// Groups returns page of groups.
	Groups(meta PageMetadata, token string) (GroupsPage, error)

	// Group returns users group object by id.
	Group(id, token string) (Group, error)

	// Assign assigns member of member type (thing or user) to a group.
	Assign(memberIDs []string, memberType, groupID string, token string) error

	// Unassign removes member from a group.
	Unassign(token, groupID string, memberIDs ...string) error

	// Members lists members of a group.
	Members(groupID, token string, offset, limit uint64) (MembersPage, error)

	// Memberships lists groups for user.
	Memberships(userID, token string, offset, limit uint64) (GroupsPage, error)

	// UpdateGroup updates existing group.
	UpdateGroup(group Group, token string) error

	// Connect bulk connects things to channels specified by id.
	Connect(conns ConnectionIDs, token string) error

	// DisconnectThing disconnect thing from specified channel by id.
	DisconnectThing(thingID, chanID, token string) error

	// CreateChannel creates new channel and returns its id.
	CreateChannel(channel Channel, token string) (string, error)

	// CreateChannels registers new channels and returns their ids.
	CreateChannels(channels []Channel, token string) ([]Channel, error)

	// Channels returns page of channels.
	Channels(token string, pm PageMetadata) (ChannelsPage, error)

	// ChannelsByThing returns page of channels that are connected or not connected
	// to specified thing.
	ChannelsByThing(token, thingID string, offset, limit uint64, connected bool) (ChannelsPage, error)

	// Channel returns channel data by id.
	Channel(id, token string) (Channel, error)

	// UpdateChannel updates existing channel.
	UpdateChannel(channel Channel, token string) error

	// DeleteChannel removes existing channel.
	DeleteChannel(id, token string) error

	// SendMessage send message to specified channel.
	SendMessage(chanID, msg, token string) error

	// ReadMessages read messages of specified channel.
	ReadMessages(chanID, token string) (MessagesPage, error)

	// SetContentType sets message content type.
	SetContentType(ct ContentType) error

	// Health returns things service health check.
	Health() (mainflux.HealthInfo, error)

	// AddBootstrap add bootstrap configuration
	AddBootstrap(token string, cfg BootstrapConfig) (string, error)

	// View returns Thing Config with given ID belonging to the user identified by the given token.
	ViewBootstrap(token, id string) (BootstrapConfig, error)

	// Update updates editable fields of the provided Config.
	UpdateBootstrap(token string, cfg BootstrapConfig) error

	// Update boostrap config certificates
	UpdateBootstrapCerts(token string, id string, clientCert, clientKey, ca string) error

	// Remove removes Config with specified token that belongs to the user identified by the given token.
	RemoveBootstrap(token, id string) error

	// Bootstrap returns Config to the Thing with provided external ID using external key.
	Bootstrap(externalKey, externalID string) (BootstrapConfig, error)

	// Whitelist updates Thing state Config with given ID belonging to the user identified by the given token.
	Whitelist(token string, cfg BootstrapConfig) error

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
