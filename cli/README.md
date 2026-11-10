# Mainflux CLI
## Build
From the project root:
```bash
make cli
```

## Usage
### Remote deployment
The `--url` flag derives every service URL from the deployment base URL, and an explicitly set service URL flag takes precedence.
```bash
mainfluxlabs-cli --url https://example.com things get all <user_token>
```

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
mainfluxlabs-cli provision things <file> <group_id> <user_token>
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
mainfluxlabs-cli things get all <user_token>
```

#### Retrieve Thing by ID
```bash
mainfluxlabs-cli things get by-id <thing_id> <user_token>
```

#### Retrieve Things by Profile
```bash
mainfluxlabs-cli things get by-profile <profile_id> <user_token>
```

#### Retrieve Metadata by Key
```bash
mainfluxlabs-cli things metadata <thing_key>
```

#### Retrieve Thing ID by Key
```bash
mainfluxlabs-cli things identify <thing_key>
```

#### Create Profile
```bash
mainfluxlabs-cli profiles create '{"name":"<profile_name>"}' <group_id> <user_token>
```

#### Bulk Provision Profiles
```bash
mainfluxlabs-cli provision profiles <file> <group_id> <user_token>
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
mainfluxlabs-cli profiles get all <user_token>
```

#### Retrieve Profile by ID
```bash
mainfluxlabs-cli profiles get by-id <profile_id> <user_token>
```

#### Retrieve a Profile by Thing
```bash
mainfluxlabs-cli profiles get by-thing <thing_id> <user_token>
```

### Messaging
#### Send a message over HTTP
```bash
mainfluxlabs-cli messages send [subtopic] '[{"bn":"Dev1","n":"temp","v":20}, {"n":"hum","v":40}, {"bn":"Dev2", "n":"temp","v":20}, {"n":"hum","v":40}]' <thing_key>
```

#### Read messages over HTTP
* Read messages from a specific subtopic by adding a flag `-s=<subtopic>`
* Reading SenML messages is the default. To read JSON messages add the flag `-f=json`
* Setting the `by-admin` flag allows the admin to list all messages of a certain format, from all things
* If `by-admin` is set, `auth_token` will be `admin_token`, otherwise `auth_token` is `thing_key`
```bash
mainfluxlabs-cli messages read [by-admin] <auth_token>
```

### Groups
#### Create Group
```bash
mainfluxlabs-cli groups create '{"name":"<group_name>","description":"<description>","metadata":{"key":"value",...}}' <org_id> <user_token>
```

#### Delete Group
```bash
mainfluxlabs-cli groups delete <group_id> <user_token>
```

#### Get Group by ID
```bash
mainfluxlabs-cli groups get <group_id> <user_token>
```

#### List all Groups
```bash
mainfluxlabs-cli groups get all <user_token>
```

#### Update Group
```bash
mainfluxlabs-cli groups update '{"name":"<new_name>"}' <group_id> <user_token>
```

#### List Things by Group
```bash
mainfluxlabs-cli groups things <group_id> <user_token>
```

#### View Group by Thing
```bash
mainfluxlabs-cli groups thing <thing_id> <user_token>
```

#### List Profiles by Group
```bash
mainfluxlabs-cli groups profiles <group_id> <user_token>
```

#### View Group by Profile
```bash
mainfluxlabs-cli groups profile <profile_id> <user_token>
```

#### Create Group Membership
```bash
mainfluxlabs-cli group_memberships create '[{"member_id":"<member_id>","email":"<email>","role":"<role>"}]' <group_id> <user_token>
```

#### Remove Group Membership
```bash
mainfluxlabs-cli group_memberships delete '["<member_id>"]' <group_id> <user_token>
```

#### Update Group Membership
```bash
mainfluxlabs-cli group_memberships update '[{"member_id":"<member_id>","role":"<new_role>"}]' <group_id> <user_token>
```

#### List Group Memberships
```bash
mainfluxlabs-cli group_memberships get <group_id> <user_token>
```

### Orgs
#### Create Org
```bash
mainfluxlabs-cli orgs create '{"name":"<org_name>","description":"<description>","metadata":{"key":"value",...}}' <user_token>
```

#### Get Org by ID
```bash
mainfluxlabs-cli orgs get <org_id> <user_token>
```

#### List all Orgs
```bash
mainfluxlabs-cli orgs get all <user_token>
```

#### Update Org
```bash
mainfluxlabs-cli orgs update '{"name":"<new_name>"}' <org_id> <user_token>
```

#### Delete Org
```bash
mainfluxlabs-cli orgs delete <org_id> <user_token>
```

#### Create Org Membership
```bash
mainfluxlabs-cli org_memberships create '[{"member_id":"<member_id>","email":"<email>","role":"<role>"}]' <org_id> <user_token>
```

#### Remove Org Membership
```bash
mainfluxlabs-cli org_memberships delete '["<member_id>"]' <org_id> <user_token>
```

#### Get Org Membership
```bash
mainfluxlabs-cli org_memberships get <all | member_id> <org_id> <user_token>
```

#### Update Org Membership
```bash
mainfluxlabs-cli org_memberships update '[{"member_id":"<member_id>","role":"<new_role>"}]' <org_id> <user_token>
```

#### List Org Membership
```bash
mainfluxlabs-cli org_memberships members <org_id> <user_token>
```

### Webhooks
#### Create Webhooks
```bash
mainfluxlabs-cli webhooks create '[{"name":"<webhook_name>","url":"<http://webhook-url.com>","headers":{"key":"value",...}}]' <thing_id> <user_token>
```

#### Get Webhook by ID
```bash
mainfluxlabs-cli webhooks get by-id <id> <user_token>
```

#### Get Webhooks by Group
```bash
mainfluxlabs-cli webhooks get by-group <group_id> <user_token>
```

#### Update Webhook
```bash
mainfluxlabs-cli webhooks update '{"name":"<new_name>","url":"<http://webhook-url.com>"}' <webhook_id> <user_token>
```

#### Delete Webhooks
```bash
mainfluxlabs-cli webhooks delete '["<webhook_id>"]' <user_token>
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

### Benchmark
Provision orgs, groups, transformer profiles, and things, then publish backfilled JSON messages over the HTTP adapter to load-test ingestion and storage.

#### Run a benchmark
```bash
mainfluxlabs-cli benchmark <username> <password>
```

#### Run against a remote deployment
```bash
mainfluxlabs-cli benchmark <username> <password> --url https://example.com
```

#### Flags
| Flag              | Default   | Description                                                     |
| ----------------- | --------- | --------------------------------------------------------------- |
| `--orgs`          | `3`       | Number of orgs to create                                        |
| `--groups`        | `5`       | Number of groups per org                                        |
| `--things`        | `100`     | Number of things to create                                      |
| `--messages`      | `10000`   | Number of messages to send                                      |
| `--rate`          | `0`       | Approximate maximum messages per second, `0` disables the limit |
| `--batch`         | `100`     | Maximum number of messages per HTTP request                     |
| `--backfill-days` | `30`      | Spread message timestamps over the last N days                  |
| `--mtls`          | `false`   | Authenticate a random half of the things over mutual TLS        |

Each request carries a random number of messages between 1 and `--batch`, sent as a JSON object for a single message and as a JSON array otherwise. The sustained ingestion rate is bounded by the database insert rate, so `--rate` should be set to that rate to prevent the message broker buffer from overflowing.

With `--mtls`, the selected things obtain client certificates from the certs service and publish over HTTPS without the `Authorization` header, so the deployment must verify client certificates and still accept the header from the remaining things.

The results report the requests and messages per authentication type and the failed requests per cause.
