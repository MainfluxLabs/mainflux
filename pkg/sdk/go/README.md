# Mainflux Go SDK

Go SDK, a Go driver for Mainflux HTTP API.

Does both system administration (provisioning) and messaging.

## Installation
Import `"github.com/MainfluxLabs/mainflux/sdk/go"` in your Go package.

```
import "github.com/MainfluxLabs/mainflux/pkg/sdk/go"```

Then call SDK Go functions to interact with the system.

## API Reference

```go
FUNCTIONS

func NewSDK(conf Config) mfSDK

func (sdk mfSDK) CreateUser(token string, user User) (string, error)
    CreateUser - create user

func (sdk mfSDK) User(token, id string) (User, error)
    User - gets user

func (sdk mfSDK) Users(token string, pm PageMetadata) (UsersPage, error)
    Users - gets list of users.
    
func (sdk mfSDK) CreateToken(user User) (string, error)
    CreateToken - create user token

func (sdk mfSDK) RegisterUser(user User) (string, error)
    RegisterUser - registers mainflux user
    
func (sdk mfSDK) UpdateUser(user User, token string) error
    UpdateUser - update user

func (sdk mfSDK) UpdatePassword(oldPass, newPass, token string) error
    UpdatePassword - update user password

func (sdk mfSDK) CreateThing(thing Thing, groupID, token string) (string, error)
    CreateThing - creates new thing and generates thing UUID

func (sdk mfSDK) CreateThings(things []Thing, groupID, token string) ([]Thing, error)
    CreateThings - registers new things 
	
func (sdk mfSDK) Things(token string, pm PageMetadata) (ThingsPage, error)
     Things - gets all things
	
func (sdk mfSDK) ThingsByChannel(token, chanID string, offset, limit uint64) (ThingsPage, error)
    ThingsByChannel - gets things by channel
	
func (sdk mfSDK) Thing(id, token string) (Thing, error)
    Thing - gets thing by id.

func (sdk mfSDK) UpdateThing(thing Thing, token string) error
    UpdateThing - updates existing thing
    
func (sdk mfSDK) DeleteThing(id, token string) error
    DeleteThing - removes thing

func (sdk mfSDK) DeleteThings(ids []string, token string) error
    DeleteThings - removes existing things
	
func (sdk mfSDK) IdentifyThing(key string) (string, error)
     IdentifyThing - validates thing's key and returns its ID
	
func (sdk mfSDK) CreateGroup(group Group, orgID, token string) (string, error)
    CreateGroup - creates new group
	
func (sdk mfSDK) DeleteGroup(id, token string) error
    DeleteGroup - deletes users group

func (sdk mfSDK) DeleteGroups(ids []string, token string) error
	DeleteGroups - delete users groups
	
func (sdk mfSDK) Groups(meta PageMetadata, token string) (GroupsPage, error)
    Groups - returns page of groups
	
func (sdk mfSDK) Group(id, token string) (Group, error)
    Group - returns users group by id
	 
func (sdk mfSDK) ListThingsByGroup(groupID, token string, offset, limit uint64) (ThingsPage, error)
    ListThingsByGroup - lists things by group

func (sdk mfSDK) ViewGroupByThing(thingID, token string) (Group, error)
    ViewGroupByThing - retrieves a group by thing
    
func (sdk mfSDK) UpdateGroup(group Group, token string) error
    UpdateGroup - updates existing group
    
func (sdk mfSDK) Connect(conns ConnectionIDs, token string) error
    Connect - connect things to channels

func (sdk mfSDK) Disconnect(conns ConnectionIDs, token string) error
    Disconnect - disconnect things from a channel

func (sdk mfSDK) CreateChannel(channel Channel, groupID, token string) (string, error)
    CreateChannel - creates new channel and generates UUID

func (sdk mfSDK) CreateChannels(channels []Channel, groupID, token string) ([]Channel, error)
    CreateChannels - registers new channels
    
func (sdk mfSDK) Channels(token string) ([]things.Channel, error)
    Channels - gets all channels

func (sdk mfSDK) ViewChannelByThing(token, thingID string) (Channel, error)
    ViewChannelByThing - returns channel by thing
    
func (sdk mfSDK) Channel(id, token string) (things.Channel, error)
    Channel - gets channel by ID

func (sdk mfSDK) UpdateChannel(channel Channel, token string) error
    UpdateChannel - updates existing channel
    
func (sdk mfSDK) DeleteChannel(id, token string) error
    DeleteChannel - removes channel

func (sdk mfSDK) DeleteChannels(ids []string, token string) error
    DeleteChannels - removes existing channels
    
func (sdk mfSDK) ListChannelsByGroup(groupID, token string, offset, limit uint64) (ChannelsPage, error)
    ListChannelsByGroup - lists channels by group
    
func (sdk mfSDK) ViewGroupByChannel(channelID, token string) (Group, error)
    ViewGroupByChannel retrieves a group by channel
    
func (sdk mfSDK) CreateRolesByGroup(roles []GroupMember, groupID, token string) error
    CreateRolesByGroup - creates new roles by group
    
func (sdk mfSDK) UpdateRolesByGroup(roles []GroupMember, groupID, token string) error
    UpdateRolesByGroup - updates existing group roles.
	
func (sdk mfSDK) RemoveRolesByGroup(ids []string, groupID, token string) error
    RemoveRolesByGroup - removes existing group roles
	
func (sdk mfSDK) ListRolesByGroup(groupID, token string, offset, limit uint64) (GroupRolesPage, error)
    ListRolesByGroup - lists roles by group
 
func (sdk mfSDK) CreateOrg(org Org, token string) error
    CreateOrg - registers new org
	
func (sdk mfSDK) Org(id, token string) (Org, error)
    Org - returns org data by id
	
func (sdk mfSDK) UpdateOrg(o Org, token string) error
    UpdateOrg - updates existing org

func (sdk mfSDK) DeleteOrg(id, token string) error
    DeleteOrg - removes existing org

func (sdk mfSDK) Orgs(meta PageMetadata, token string) (OrgsPage, error)
    Orgs - returns page of orgs

func (sdk mfSDK) ViewMember(orgID, memberID, token string) (Member, error)
    ViewMember - retrieves a member belonging to the specified org
	
func (sdk mfSDK) AssignMembers(om []OrgMember, orgID, token string) error
    AssignMembers - assigns a members to the org
	
func (sdk mfSDK) UnassignMembers(token, orgID string, memberIDs ...string) error
    UnassignMembers - unassigns a members from the specified org
    
func (sdk mfSDK) UpdateMember(member OrgMember, token string) error
    UpdateMember - updates existing member

func (sdk mfSDK) ListMembersByOrg(orgID, token string, offset, limit uint64) (MembersPage, error)
    ListMembersByOrg - lists members by org
	
func (sdk mfSDK) ListOrgsByMember(memberID, token string, offset, limit uint64) (OrgsPage, error)
    ListOrgsByMember - lists orgs by member
	
func (sdk mfSDK) CreateWebhooks(whs []Webhook, groupID, token string) ([]Webhook, error)
    CreateWebhooks - creates new webhooks
	
func (sdk mfSDK) ListWebhooksByGroup(groupID, token string) (Webhooks, error)
    ListWebhooksByGroup - lists webhooks by group
	
func (sdk mfSDK) Webhook(webhookID, token string) (Webhook, error)
    Webhook - returns webhook by id
	
func (sdk mfSDK) UpdateWebhook(wh Webhook, token string) error
    UpdateWebhook - updates existing webhook
	
func (sdk mfSDK) DeleteWebhooks(ids []string, groupID, token string) error
    DeleteWebhooks - removes existing webhooks
    
func (sdk mfSDK) SendMessage(chanID, msg, token string) error
    SendMessage - send message on Mainflux channel

func (sdk mfSDK) ReadMessages(chanID, token string) (MessagesPage, error)
    ReadMessages - read messages of specified channel

func (sdk mfSDK) SetContentType(ct ContentType) error
    SetContentType - set message content type. Available options are SenML JSON, custom JSON and custom binary (octet-stream).

func (sdk mfSDK) Health() (mainflux.HealthInfo, error)
    Health - things service health check

func (sdk mfSDK) IssueCert(thingID string, keyBits int, keyType, valid, token string) (Cert, error)
    IssueCert - issues a certificate for a thing required for mtls

func (sdk mfSDK) RemoveCert(id, token string) error
    RemoveCert - removes a certificate

func (sdk mfSDK) RevokeCert(thingID, certID, token string) error
    RevokeCert - revokes certificate with certID for thing with thingID

func (sdk mfSDK) Issue(token string, duration time.Duration) (KeyRes, error)
    Issue - issues a new key, returning its token value alongside
	
func (sdk mfSDK) Revoke(token, id string) error
    Revoke - removes the key with the provided ID 
    
func (sdk mfSDK) RetrieveKey(token, id string) (retrieveKeyRes, error)
	RetrieveKey - retrieves data for the key identified by the provided ID

```
