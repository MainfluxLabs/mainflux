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
mainfluxlabs-cli things create '{"name":"<thing_name>","profile_id":"<profile_id>"}' <group_id> <user_token>
```

#### Create Thing with metadata
```bash
mainfluxlabs-cli things create '{"name":"<thing_name>","profile_id":"<profile_id>","metadata": {\"key1\":\"value1\"}}' <group_id> <user_token>
```

#### Bulk Provision Things
```bash
mainfluxlabs-cli provision things <file> <user_token>
```

* `file` - A CSV or JSON file containing things
* `user_token` - A valid user auth token for the current system

#### Update Thing
```bash
mainfluxlabs-cli things update '{"name":"<new_name>","profile_id":"<profile_id>"}' <thing_id> <user_token>
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

#### Retrieve Metadata By Key
```bash
mainfluxlabs-cli things metadata <thing_key>
```

#### Create Profile
```bash
mainfluxlabs-cli profiles create '{"name":"<profile_name>"}' <group_id> <user_token>
```

#### Bulk Provision Profiles
```bash
mainfluxlabs-cli provision profiles <file> <user_token>
```

* `file` - A CSV or JSON file containing profiles
* `user_token` - A valid user auth token for the current system

#### Update Profile
```bash
mainfluxlabs-cli profiles update '{"name":"<new_name>"}' <profile_id> <user_token>
```

#### Remove Profile
```bash
mainfluxlabs-cli profiles delete <profile_id> <user_token>
```

#### Retrieve a subset list of provisioned Profiles
```bash
mainfluxlabs-cli profiles get all --offset=1 --limit=5 <user_token>
```

#### Retrieve Profile By ID
```bash
mainfluxlabs-cli profiles get <profile_id> <user_token>
```

#### Retrieve a Profile by Thing
```bash
mainfluxlabs-cli profiles thing <thing_id> <user_token>
```

#### Retrieve a subset list of Things by Profile
```bash
mainfluxlabs-cli things profile <profile_id> <user_token>
```

### Messaging
#### Send a message over HTTP
```bash
mainfluxlabs-cli messages send <profile_id> '[{"bn":"Dev1","n":"temp","v":20}, {"n":"hum","v":40}, {"bn":"Dev2", "n":"temp","v":20}, {"n":"hum","v":40}]' <thing_auth_token>
```

#### Read messages over HTTP
```bash
mainfluxlabs-cli messages read <profile_id> <thing_auth_token>
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

#### List profiles by group
```bash
mainfluxlabs-cli groups profiles <group_id> <user_token>
```

#### View group by profile
```bash
mainfluxlabs-cli groups profile <profile_id> <user_token>
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
