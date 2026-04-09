# Rules Service

The Rules service provides two complementary automation engines for processing incoming device messages: a **rule engine** for threshold-based condition matching, and a **Lua scripting engine** for arbitrary message processing logic.

Both engines are driven by the same event stream: every message published by a thing is evaluated against all rules and scripts assigned to that thing.

## Resource Model

```
Group
└── Rule / Script
    └── assigned to → Things
```

Rules and scripts are created within a group and then assigned to individual things. When a thing publishes a message, the service evaluates all rules and scripts assigned to that thing against the message payload.

## Rules

A rule evaluates a set of conditions against an incoming message payload. When all conditions are met (using AND or OR logic), the rule triggers one or more actions.

| Field         | Description                                                                                                      |
|---------------|------------------------------------------------------------------------------------------------------------------|
| `id`          | Unique rule identifier (UUID)                                                                                    |
| `group_id`    | ID of the group the rule belongs to                                                                              |
| `name`        | Human-readable rule name                                                                                         |
| `description` | Optional free-form description                                                                                   |
| `conditions`  | List of conditions to evaluate (see below)                                                                       |
| `operator`    | Logical operator applied across all conditions: `AND` or `OR`. Required when more than one condition is defined. |
| `actions`     | List of actions to trigger when conditions are met (see below)                                                   |

### Conditions

Each condition compares a named field in the message payload against a numeric threshold.

| Field        | Description                                                                                                                                                            |
|--------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `field`      | The payload field name to evaluate. For SenML messages, this matches the `name` key. For JSON messages, dot-notation paths are supported (e.g. `sensors.temperature`). |
| `comparator` | Comparison operator: `==`, `>=`, `<=`, `>`, `<`                                                                                                                        |
| `threshold`  | Numeric value to compare against                                                                                                                                       |

### Actions

Each action specifies what to do when a rule fires.

| Field   | Description                                                                                              |
|---------|----------------------------------------------------------------------------------------------------------|
| `type`  | Action type: `alarm`, `smtp`, or `smpp`                                                                  |
| `id`    | Required for `smtp` and `smpp` types — the ID of the configured notifier to trigger                     |
| `level` | Required for `alarm` type — severity level: 1=info, 2=warning, 3=minor, 4=major, 5=critical             |

- **`alarm`** — publishes an alarm event with the specified severity level, consumed by the Alarms service
- **`smtp`** — triggers an SMTP email notification via the registered notifier with the given `id`
- **`smpp`** — triggers an SMPP SMS notification via the registered notifier with the given `id`

## Lua Scripts

Lua scripts provide a programmable alternative to condition-based rules. A script is arbitrary Lua code that runs once per incoming message (or once per array element, for array payloads). Scripts can read the message payload, make decisions, and call platform API functions.

### Script Fields

| Field         | Description                           |
|---------------|---------------------------------------|
| `id`          | Unique script identifier (UUID)       |
| `group_id`    | ID of the group the script belongs to |
| `name`        | Human-readable script name            |
| `description` | Optional free-form description        |
| `script`      | Lua source code (max 65,535 bytes)    |

### Lua Execution Environment

Each script execution receives an isolated Lua environment. The following global is available:

#### `mfx` table

| Field                      | Type   | Description                                        |
|----------------------------|--------|----------------------------------------------------|
| `mfx.message.payload`      | table  | Parsed message payload (JSON object or array item) |
| `mfx.message.subtopic`     | string | Message subtopic                                   |
| `mfx.message.created`      | number | Message creation timestamp (Unix)                  |
| `mfx.message.publisher_id` | string | Thing ID that published the message                |

#### `mfx` API functions

| Function                       | Returns             | Description                                                                    |
|--------------------------------|---------------------|--------------------------------------------------------------------------------|
| `mfx.smtp_notify(notifier_id)` | `bool[, error_msg]` | Triggers an SMTP notification via the specified notifier. Max 2 calls per run. |
| `mfx.create_alarm(level)`      | `bool[, error_msg]` | Creates an alarm event attributed to this script. Level is an integer (1=info, 2=warning, 3=minor, 4=major, 5=critical). Max 1 call per run. |
| `mfx.log(message)`             | `bool[, error_msg]` | Appends a message to the run log (max 256 lines, 2048 chars each).             |

Available Lua standard libraries: `base`, `math`, `string`, `table`. The `print` function is disabled.

#### Execution Limits

| Limit                 | Value       |
|-----------------------|-------------|
| Max instructions      | 1,000,000   |
| Max log lines per run | 256         |
| Max log line length   | 2,048 chars |

#### Example Script

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

-- Condensation risk: surface is near or below dew point
if spread <= 2.0 then
  mfx.log("Condensation risk: spread=" .. string.format("%.1f", spread) .. "°C")
  mfx.create_alarm(3)
  mfx.smtp_notify("654e4567-e89b-12d3-a456-426614174999")
end
```

### Script Runs

Every script execution is recorded as a script run. Run records capture the outcome, logs, and any runtime error.

| Field         | Description                                  |
|---------------|----------------------------------------------|
| `id`          | Unique run identifier (UUID)                 |
| `script_id`   | ID of the script that was executed           |
| `thing_id`    | ID of the thing that triggered the execution |
| `logs`        | Log lines written via `mfx.log()`            |
| `started_at`  | Execution start timestamp (RFC 3339)         |
| `finished_at` | Execution end timestamp (RFC 3339)           |
| `status`      | `success` or `fail`                          |
| `error`       | Runtime error message, if any                |

Run records are retrievable per thing and can be bulk-deleted via the API.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                      | Description                                                                | Default                  |
|-------------------------------|----------------------------------------------------------------------------|--------------------------|
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
| `MF_RULES_SCRIPTS_ENABLED`    | Enable Lua scripting engine                                                | false                    |

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
