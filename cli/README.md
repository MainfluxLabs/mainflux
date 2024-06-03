# Mainflux CLI
## Build
From the project root:
```bash
make cli
```

## Usage
### Service
#### Get Mainflux Things services Health Check
```bash
mainfluxlabs-cli health
```

### Users management
#### Create User
```bash
mainfluxlabs-cli users create <user_email> <user_password>
```

#### Login User
```bash
mainfluxlabs-cli users token <user_email> <user_password>
```

#### Retrieve User
```bash
mainfluxlabs-cli users get <user_token>
```

#### Update User Metadata
```bash
mainfluxlabs-cli users update '{"key1":"value1", "key2":"value2"}' <user_token>
```

#### Update User Password
```bash
mainfluxlabs-cli users password <old_password> <password> <user_token>
```

### System Provisioning
#### Create Group
```bash
mainfluxlabs-cli groups create '{"name":"<group_name>"}' <org_id> <user_token>
```

#### Create Thing
```bash
mainfluxlabs-cli things create '{"name":"<thing_name>"}' <group_id> <user_token>
```

#### Create Thing with metadata
```bash
mainfluxlabs-cli things create '{"name":"<thing_name>", "metadata": {\"key1\":\"value1\"}}' <group_id> <user_token>
```

#### Bulk Provision Things
```bash
mainfluxlabs-cli provision things <file> <user_token>
```

* `file` - A CSV or JSON file containing things
* `user_token` - A valid user auth token for the current system

#### Update Thing
```bash
mainfluxlabs-cli things update '{"name":"<new_name>"}' <thing_id> <user_token>
```

#### Remove Thing
```bash
mainfluxlabs-cli things delete <thing_id> <user_token>
```

#### Retrieve a subset list of provisioned Things
```bash
mainfluxlabs-cli things get all --offset=1 --limit=5 <user_token>
```

#### Retrieve Thing By ID
```bash
mainfluxlabs-cli things get <thing_id> <user_token>
```

#### Create Channel
```bash
mainfluxlabs-cli channels create '{"name":"<channel_name>"}' <group_id> <user_token>
```

#### Bulk Provision Channels
```bash
mainfluxlabs-cli provision channels <file> <user_token>
```

* `file` - A CSV or JSON file containing channels
* `user_token` - A valid user auth token for the current system

#### Update Channel
```bash
mainfluxlabs-cli channels update '{"name":"<new_name>"}' <channel_id> <user_token>
```

#### Remove Channel
```bash
mainfluxlabs-cli channels delete <channel_id> <user_token>
```

#### Retrieve a subset list of provisioned Channels
```bash
mainfluxlabs-cli channels get all --offset=1 --limit=5 <user_token>
```

#### Retrieve Channel By ID
```bash
mainfluxlabs-cli channels get <channel_id> <user_token>
```

### Access control
#### Connect Thing to Channel
```bash
mainfluxlabs-cli things connect <thing_id> <channel_id> <user_token>
```

#### Bulk Connect Things to Channels
```bash
mainfluxlabs-cli provision connect <file> <user_token>
```

* `file` - A CSV or JSON file containing thing and channel ids
* `user_token` - A valid user auth token for the current system

An example CSV file might be

```csv
<thing_id>,<channel_id>
<thing_id>,<channel_id>
```

in which the first column is thing IDs and the second column is channel IDs. A connection will be created for each thing to each channel. This example would result in 4 connections being created.

A comparable JSON file would be

```json
{
    "thing_ids": [
        "<thing_id>",
        "<thing_id>"
    ],
    "channel_ids": [
        "<channel_id>",
        "<channel_id>"
    ]
}
```

#### Disconnect Thing from Channel
```bash
mainfluxlabs-cli things disconnect <thing_id> <channel_id> <user_token>

```

#### Retrieve a Channel by Thing
```bash
mainfluxlabs-cli things connections <thing_id> <user_token>
```

#### Retrieve a subset list of Things connected to Channel
```bash
mainfluxlabs-cli channels connections <channel_id> <user_token>
```

### Messaging
#### Send a message over HTTP
```bash
mainfluxlabs-cli messages send <channel_id> '[{"bn":"Dev1","n":"temp","v":20}, {"n":"hum","v":40}, {"bn":"Dev2", "n":"temp","v":20}, {"n":"hum","v":40}]' <thing_auth_token>
```

#### Read messages over HTTP
```bash
mainfluxlabs-cli messages read <channel_id> <thing_auth_token>
```

### Groups
#### Create new group
```bash
mainfluxlabs-cli groups create '{"name":"<group_name>","description":"<description>","metadata":{"key":"value",...}}' <org_id> <user_token>
```

#### Delete group
```bash
mainfluxlabs-cli groups delete <group_id> <user_token>
```

#### Get group by id
```bash
mainfluxlabs-cli groups get <group_id> <user_token>
```

#### List all groups
```bash
mainfluxlabs-cli groups get all <user_token>
```

#### Update group
```bash
mainfluxlabs-cli groups update '{"name":"<new_name>"}' <group_id> <user_token>
```

#### List things by group
```bash
mainfluxlabs-cli groups things <group_id> <user_token>
```

#### View group by thing
```bash
mainfluxlabs-cli groups thing <thing_id> <user_token>
```

#### List channels by group
```bash
mainfluxlabs-cli groups channels <group_id> <user_token>
```

#### View group by channel
```bash
mainfluxlabs-cli groups channel <channel_id> <user_token>
```

### Orgs
#### Create new org
```bash
mainfluxlabs-cli orgs create '{"name":"<org_name>","description":"<description>","metadata":{"key":"value",...}}' <user_token>
```

#### Get org by id
```bash
mainfluxlabs-cli orgs get <org_id> <user_token>
```

#### List all orgs
```bash
mainfluxlabs-cli orgs get all <user_token>
```

#### Update org
```bash
mainfluxlabs-cli orgs update '{"name":"<new_name>"}' <org_id> <user_token>
```

#### Delete org
```bash
mainfluxlabs-cli orgs delete <org_id> <user_token>
```

#### Assign user to an org
```bash
mainfluxlabs-cli orgs assign '[{"member_id":"<member_id>","email":"<email>","role":"<role>"}]' <org_id> <user_token>
```

#### Unassign user from org
```bash
mainfluxlabs-cli orgs unassign '["<member_id>"]' <org_id> <user_token>
```

#### Update members
```bash
mainfluxlabs-cli orgs update-members '[{"member_id":"<member_id>","role":"<new_role>"}]' <org_id> <user_token>
```

#### List users by org
```bash
mainfluxlabs-cli orgs members <org_id> <user_token>
```

#### List orgs that user belongs to
```bash
mainfluxlabs-cli orgs memberships <member_id> <user_token>
```

### Webhooks
#### Create new webhooks
```bash
mainfluxlabs-cli webhooks create '[{"name":"<webhook_name>","url":"<http://webhook-url.com>","headers":{"key":"value",...}}]' <group_id> <user_token>
```

#### Get webhook by id
```bash
mainfluxlabs-cli webhooks get by-id <id> <user_token>
```

#### Get webhooks by group
```bash
mainfluxlabs-cli webhooks get group <group_id> <user_token>
```

#### Update webhook
```bash
mainfluxlabs-cli webhooks update '{"name":"<new_name>","url":"<http://webhook-url.com>"}' <webhook_id> <user_token>
```

#### Delete webhooks
```bash
mainfluxlabs-cli webhooks delete '["<webhook_id>"]' <group_id> <user_token>
```

### Keys management
#### Issue a new Key
```bash
mainfluxlabs-cli keys issue <duration> <user_token>
```

#### Remove API key from database
```bash
mainfluxlabs-cli keys revoke <key_id> <user_token>
```

#### Retrieve API key with given id
```bash
mainfluxlabs-cli keys retrieve <key_id> <user_token>
```
