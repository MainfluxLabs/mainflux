# Rules Service

The Rules service provides a **rule engine** for threshold-based condition matching on incoming device messages.

The engine is driven by the event stream: every message published by a thing is evaluated against all rules assigned to that thing.

## Resource Model

```
Group
└── Rule
    └── assigned to → Things
```

Rules are created within a group and then assigned to individual things. When a thing publishes a message, the service evaluates all rules assigned to that thing against the message payload.

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

Each condition compares a named field in the message payload against a numeric threshold.

| Field        | Description                                                                                                                                                            |
| ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `field`      | The payload field name to evaluate. For SenML messages, this matches the `name` key. For JSON messages, dot-notation paths are supported (e.g. `sensors.temperature`). |
| `comparator` | Comparison operator: `==`, `>=`, `<=`, `>`, `<`                                                                                                                        |
| `threshold`  | Numeric value to compare against                                                                                                                                       |

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

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                          | Description                                                                | Default                  |
| --------------------------------- | -------------------------------------------------------------------------- | ------------------------ |
| `MF_RULES_LOG_LEVEL`              | Log level for the Rules service (debug, info, warn, error)                 | error                    |
| `MF_BROKER_URL`                   | Message broker instance URL                                                | nats://localhost:4222    |
| `MF_RULES_HTTP_PORT`              | Rules service HTTP port                                                    | 9027                     |
| `MF_JAEGER_URL`                   | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                          |
| `MF_RULES_DB_HOST`                | Database host address                                                      | localhost                |
| `MF_RULES_DB_PORT`                | Database host port                                                         | 5432                     |
| `MF_RULES_DB_USER`                | Database user                                                              | mainflux                 |
| `MF_RULES_DB_PASS`                | Database password                                                          | mainflux                 |
| `MF_RULES_DB`                     | Name of the database used by the service                                   | rules                    |
| `MF_RULES_DB_SSL_MODE`            | Database connection SSL mode (disable, require, verify-ca, verify-full)    | disable                  |
| `MF_RULES_DB_SSL_CERT`            | Path to the PEM encoded certificate file                                   |                          |
| `MF_RULES_DB_SSL_KEY`             | Path to the PEM encoded key file                                           |                          |
| `MF_RULES_DB_SSL_ROOT_CERT`       | Path to the PEM encoded root certificate file                              |                          |
| `MF_RULES_CLIENT_TLS`             | Flag that indicates if TLS should be turned on                             | false                    |
| `MF_RULES_CA_CERTS`               | Path to trusted CAs in PEM format                                          |                          |
| `MF_RULES_SERVER_CERT`            | Path to server certificate in PEM format                                   |                          |
| `MF_RULES_SERVER_KEY`             | Path to server key in PEM format                                           |                          |
| `MF_THINGS_AUTH_GRPC_URL`         | Things service Auth gRPC URL                                               | localhost:8183           |
| `MF_THINGS_AUTH_GRPC_TIMEOUT`     | Things service Auth gRPC request timeout                                   | 1s                       |
| `MF_RULES_ES_URL`                 | Event store URL                                                            | redis://localhost:6379/0 |
| `MF_RULES_EVENT_CONSUMER`         | Event store consumer name                                                  | rules                    |

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
