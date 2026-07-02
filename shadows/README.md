# Shadows Service

The Shadows service maintains a **device shadow** for each thing — a persisted record holding the
thing's last _reported_ state and its _desired_ state. It lets users read and set a device's
state even while the device is offline, and aligns the two whenever the device reconnects.

## Shadows

A shadow is a single record per thing.

| Field         | Description                                                                     |
| ------------- | ------------------------------------------------------------------------------- |
| `thing_id`    | ID of the thing the shadow belongs to (UUID)                                    |
| `state`       | Nested object holding the `desired`, `reported`, and `delta` states (see below) |
| `reported_at` | Unix timestamp (seconds) of the last reported-state update                      |
| `updated_at`  | Unix timestamp (seconds) of the last desired-state update                       |

### State

`desired`, `reported`, and `delta` are each a free-form JSON object (a set of key/value pairs).

| Field      | Description                                                                                              |
| ---------- | -------------------------------------------------------------------------------------------------------- |
| `desired`  | State the application wants the device to reach.                                                         |
| `reported` | State the device last reported. Merged from the device's telemetry messages.                             |
| `delta`    | Computed subset of `desired` whose values differ from (or are absent in) `reported`. Omitted when empty. |

The `delta` is derived on read and on every state change; it is never stored directly. Keys present
only in `reported` are not part of the delta.

## How it works

- **Desired state** is set by a user through the HTTP API (`PUT /things/{id}/shadows`). On update, the
  service recomputes the delta and publishes it to the device on its command subject
  (`things.<id>.commands.shadow`, protocol `shadows`).
- **Reported state** is updated automatically as the thing publishes messages. The service consumes
  messages from the broker, flattens each into a state patch, and merges the patch into `reported`
  (no-op writes are skipped).
- On each reported-state change, any still-pending delta is re-published, so a reconnecting device
  receives commands it missed while offline.

Authorization is delegated to the Things service: reading a shadow requires `viewer` access on the
thing's group, while updating or removing a shadow requires `editor` access.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                      | Description                                                                | Default                  |
| ----------------------------- | -------------------------------------------------------------------------- | ------------------------ |
| `MF_SHADOWS_LOG_LEVEL`        | Log level for the Shadows service (debug, info, warn, error)               | error                    |
| `MF_BROKER_URL`               | Message broker instance URL                                                | nats://localhost:4222    |
| `MF_SHADOWS_HTTP_PORT`        | Shadows service HTTP port                                                  | 9031                     |
| `MF_JAEGER_URL`               | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                          |
| `MF_SHADOWS_DB_HOST`          | Database host address                                                      | localhost                |
| `MF_SHADOWS_DB_PORT`          | Database host port                                                         | 5432                     |
| `MF_SHADOWS_DB_USER`          | Database user                                                              | mainflux                 |
| `MF_SHADOWS_DB_PASS`          | Database password                                                          | mainflux                 |
| `MF_SHADOWS_DB`               | Name of the database used by the service                                   | shadows                  |
| `MF_SHADOWS_DB_SSL_MODE`      | Database connection SSL mode (disable, require, verify-ca, verify-full)    | disable                  |
| `MF_SHADOWS_DB_SSL_CERT`      | Path to the PEM encoded certificate file                                   |                          |
| `MF_SHADOWS_DB_SSL_KEY`       | Path to the PEM encoded key file                                           |                          |
| `MF_SHADOWS_DB_SSL_ROOT_CERT` | Path to the PEM encoded root certificate file                              |                          |
| `MF_SHADOWS_CLIENT_TLS`       | Flag that indicates if TLS should be turned on                             | false                    |
| `MF_SHADOWS_CA_CERTS`         | Path to trusted CAs in PEM format                                          |                          |
| `MF_SHADOWS_SERVER_CERT`      | Path to server certificate in PEM format                                   |                          |
| `MF_SHADOWS_SERVER_KEY`       | Path to server key in PEM format                                           |                          |
| `MF_THINGS_AUTH_GRPC_URL`     | Things service Auth gRPC URL                                               | localhost:8183           |
| `MF_THINGS_AUTH_GRPC_TIMEOUT` | Things service Auth gRPC request timeout in seconds                        | 1s                       |
| `MF_AUTH_GRPC_URL`            | Auth service gRPC URL                                                      | localhost:8181           |
| `MF_AUTH_GRPC_TIMEOUT`        | Auth service gRPC request timeout in seconds                               | 1s                       |
| `MF_SHADOWS_ES_URL`           | Event store URL                                                            | redis://localhost:6379/0 |

## Deployment

The service itself is distributed as Docker container. Check the [`shadows`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# Download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the shadows service
make shadows

# Copy binary to bin
make install

# Set the environment variables and run the service
MF_SHADOWS_LOG_LEVEL=[Shadows log level] \
MF_BROKER_URL=[Message broker instance URL] \
MF_SHADOWS_HTTP_PORT=[Shadows service HTTP port] \
MF_SHADOWS_DB_HOST=[Database host address] \
MF_SHADOWS_DB_PORT=[Database host port] \
MF_SHADOWS_DB_USER=[Database user] \
MF_SHADOWS_DB_PASS=[Database password] \
MF_SHADOWS_DB=[Shadows database name] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout] \
MF_AUTH_GRPC_URL=[Auth service gRPC URL] \
MF_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout] \
$GOBIN/mainfluxlabs-shadows
```

## Usage

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).
