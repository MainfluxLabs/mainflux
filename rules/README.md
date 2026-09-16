# Rules Service

The Rules service provides a **rule engine** for condition matching on incoming device messages — by threshold comparison, or by running a Lua script and using its result.

The engine is driven by the event stream: every message published by a thing is evaluated against all rules assigned to that thing.

## Resource Model

```
Group
├── Rule
│   ├── assigned to → Things
│   └── condition may reference → Script
└── Script
```

Rules are created within a group and then assigned to individual things. When a thing publishes a message, the service evaluates all rules assigned to that thing against the message payload.

Scripts are created within a group; a rule references one by ID from a script condition.

## Rules

A rule evaluates a set of conditions against an incoming payload. When conditions are met according to the configured operator (AND or OR), the rule triggers one or more actions.

| Field         | Description                                                                                                      |
| ------------- | ---------------------------------------------------------------------------------------------------------------- |
| `id`          | Unique rule identifier (UUID)                                                                                    |
| `group_id`    | ID of the group the rule belongs to                                                                              |
| `name`        | Human-readable rule name                                                                                         |
| `description` | Optional free-form description                                                                                   |
| `input`       | Defines what triggers rule evaluation (see below)                                                                |
| `conditions`  | List of conditions to evaluate (see below)                                                                       |
| `operator`    | Logical operator applied across all conditions: `AND` or `OR`. Required when more than one condition is defined. |
| `actions`     | List of actions to trigger when conditions are met (see below)                                                   |

### Input

The `input` field defines what triggers rule evaluation.

| Field       | Description                                       |
| ----------- | ------------------------------------------------- |
| `type`      | Trigger type: `message` or `alarm`                |
| `thing_ids` | IDs of things to which this rule applies          |
| `config`    | Optional input-type-specific settings (see below) |

### Input Config

The `config` field holds optional, input-type-specific settings.

| Field      | Description                                                                                  |
| ---------- | -------------------------------------------------------------------------------------------- |
| `subtopic` | Filters messages by subtopic (e.g. `sensors.room1`). If omitted, all messages are evaluated. |

### Conditions

Each condition has a required `type`, which selects how it's evaluated.

| Field        | Description                                                                                                                                                                            |
| ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `type`       | `threshold` or `script`                                                                                                                                                                |
| `field`      | Threshold only. The payload field name to evaluate. For SenML messages, this matches the `name` key. For JSON messages, dot-notation paths are supported (e.g. `sensors.temperature`). |
| `comparator` | Threshold only. Comparison operator: `==`, `>=`, `<=`, `>`, `<`                                                                                                                        |
| `threshold`  | Threshold only. Numeric value to compare against                                                                                                                                       |
| `script_id`  | Script only. ID of the [script](#scripts) to run                                                                                                                                       |

#### Script conditions

A `script` condition runs the referenced Lua script and uses its return value as the result: a strict `return true` counts as met, anything else — `return false`, no return, or a runtime error — counts as not met. The script's result is combined with any other conditions on the rule the same way a threshold result would be, under the rule's `operator`.

Script conditions run with a **read-only** API — the only bound function is `mfx.log`, since a condition must not have side effects. Every evaluation is recorded as a [script run](#scripts).

Script conditions are only supported on `message`-input rules, not `alarm`-input rules.

### Actions

Each action specifies what to do when a rule fires.

| Field   | Description                                                                                 |
| ------- | ------------------------------------------------------------------------------------------- |
| `type`  | Action type: `alarm`, `smtp`, or `smpp`                                                     |
| `id`    | Required for `smtp` and `smpp` types — the ID of the configured notifier to trigger         |
| `level` | Required for `alarm` type — severity level: 1=info, 2=warning, 3=minor, 4=major, 5=critical |

- **`alarm`** — publishes an alarm event with the specified severity level, consumed by the Alarms service
- **`smtp`** — triggers an SMTP email notification via the registered notifier with the given `id`
- **`smpp`** — triggers an SMPP SMS notification via the registered notifier with the given `id`

## Scripts

A Lua script is a group-scoped resource, referenced by ID from a rule's [script conditions](#script-conditions).

| Field         | Description                           |
| ------------- | ------------------------------------- |
| `id`          | Unique script identifier (UUID)       |
| `group_id`    | ID of the group the script belongs to |
| `name`        | Human-readable script name            |
| `description` | Optional free-form description        |
| `script`      | Lua source code (max 65,535 bytes)    |

Each execution receives an isolated Lua environment: the standard `base`, `math`, `string`, and `table` libraries, and an `mfx` table exposing the triggering message:

| Field                      | Type   | Description                                        |
| -------------------------- | ------ | -------------------------------------------------- |
| `mfx.message.payload`      | table  | Parsed message payload (JSON object or array item) |
| `mfx.message.subtopic`     | string | Message subtopic                                   |
| `mfx.message.created`      | number | Message creation timestamp (Unix)                  |
| `mfx.message.publisher_id` | string | Thing ID that published the message                |

As a condition, the only bound API function is `mfx.log(message)` — appends a message to the run's log (max 256 lines, 2048 chars each). Execution is capped at 1,000,000 VM instructions.

### Example

A condition script that flags condensation risk from a payload's `temperature` and `humidity` fields:

```lua
local payload = mfx.message.payload

local temp = tonumber(payload["temperature"])
local hum  = tonumber(payload["humidity"])

if not temp or not hum or hum <= 0 then
  mfx.log("Invalid or missing fields")
  return
end

-- Magnus formula: dew point from temperature and relative humidity
local gamma = math.log(hum / 100.0) + (17.625 * temp) / (243.04 + temp)
local dew_point = 243.04 * gamma / (17.625 - gamma)
local spread = temp - dew_point  -- smaller spread → closer to condensation

mfx.log(string.format("temp=%.1f  hum=%.1f%%  dew_point=%.1f  spread=%.1f",
  temp, hum, dew_point, spread))

-- condensation risk if the surface is within 2°C of the dew point
return spread <= 2.0
```

### Script Runs

Every condition evaluation is recorded as a script run: outcome, logs, and any runtime error.

| Field         | Description                                  |
| ------------- | -------------------------------------------- |
| `id`          | Unique run identifier (UUID)                 |
| `script_id`   | ID of the script that was executed           |
| `thing_id`    | ID of the thing that triggered the execution |
| `logs`        | Log lines written via `mfx.log()`            |
| `started_at`  | Execution start timestamp (RFC 3339)         |
| `finished_at` | Execution end timestamp (RFC 3339)           |
| `status`      | `success` or `fail`                          |
| `error`       | Runtime error message, if any                |

`status` reflects only whether the script ran without a Lua runtime error — it does not reflect whether the script's return value counted as the condition being met.

Run records are retrievable per thing and can be bulk-deleted via the API.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                      | Description                                                                | Default                  |
| ----------------------------- | -------------------------------------------------------------------------- | ------------------------ |
| `MF_RULES_LOG_LEVEL`          | Log level for the Rules service (debug, info, warn, error)                 | error                    |
| `MF_BROKER_URL`               | Message broker instance URL                                                | nats://localhost:4222    |
| `MF_RULES_HTTP_PORT`          | Rules service HTTP port                                                    | 9027                     |
| `MF_JAEGER_URL`               | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                          |
| `MF_RULES_DB_HOST`            | Database host address                                                      | localhost                |
| `MF_RULES_DB_PORT`            | Database host port                                                         | 5432                     |
| `MF_RULES_DB_USER`            | Database user                                                              | mainflux                 |
| `MF_RULES_DB_PASS`            | Database password                                                          | mainflux                 |
| `MF_RULES_DB`                 | Name of the database used by the service                                   | rules                    |
| `MF_RULES_DB_SSL_MODE`        | Database connection SSL mode (disable, require, verify-ca, verify-full)    | disable                  |
| `MF_RULES_DB_SSL_CERT`        | Path to the PEM encoded certificate file                                   |                          |
| `MF_RULES_DB_SSL_KEY`         | Path to the PEM encoded key file                                           |                          |
| `MF_RULES_DB_SSL_ROOT_CERT`   | Path to the PEM encoded root certificate file                              |                          |
| `MF_RULES_CLIENT_TLS`         | Flag that indicates if TLS should be turned on                             | false                    |
| `MF_RULES_CA_CERTS`           | Path to trusted CAs in PEM format                                          |                          |
| `MF_RULES_SERVER_CERT`        | Path to server certificate in PEM format                                   |                          |
| `MF_RULES_SERVER_KEY`         | Path to server key in PEM format                                           |                          |
| `MF_THINGS_AUTH_GRPC_URL`     | Things service Auth gRPC URL                                               | localhost:8183           |
| `MF_THINGS_AUTH_GRPC_TIMEOUT` | Things service Auth gRPC request timeout                                   | 1s                       |
| `MF_RULES_ES_URL`             | Event store URL                                                            | redis://localhost:6379/0 |
| `MF_RULES_EVENT_CONSUMER`     | Event store consumer name                                                  | rules                    |

## Deployment

The service itself is distributed as Docker container. Check the [`rules`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# Download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the rules service
make rules

# Copy binary to bin
make install

# Set the environment variables and run the service
MF_RULES_LOG_LEVEL=[Rules log level] \
MF_BROKER_URL=[Message broker instance URL] \
MF_RULES_HTTP_PORT=[Rules service HTTP port] \
MF_RULES_DB_HOST=[Database host address] \
MF_RULES_DB_PORT=[Database host port] \
MF_RULES_DB_USER=[Database user] \
MF_RULES_DB_PASS=[Database password] \
MF_RULES_DB=[Rules database name] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout] \
$GOBIN/mainfluxlabs-rules
```

## Usage

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).
