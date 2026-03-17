# Webhooks Service

The Webhooks service forwards incoming device messages to external HTTP endpoints. When a thing publishes a message and the `webhook` flag in its profile config is set to `true`, the service POSTs the message payload to every webhook registered for that thing.

## Webhooks

A webhook defines an outbound HTTP destination for message forwarding.

| Field      | Description                                                        |
|------------|--------------------------------------------------------------------|
| `id`       | Unique webhook identifier (UUID)                                   |
| `group_id` | ID of the group the webhook belongs to                             |
| `thing_id` | ID of the thing the webhook is associated with                     |
| `name`     | Human-readable webhook name                                        |
| `url`      | Destination URL. Must be a valid HTTP or HTTPS URL.                |
| `headers`  | Optional HTTP headers included in every forwarded request          |
| `metadata` | Arbitrary key-value pairs for custom attributes                    |

Webhooks are created per thing (`POST /things/:id/webhooks`) and are scoped to that thing's group. Multiple webhooks can be registered for a single thing.

## Configuration

The service is configured using the environment variables from the following table. Note that any unset variables will be replaced with their default values.

| Variable                     | Description                                                             | Default               |
|------------------------------|-------------------------------------------------------------------------|-----------------------|
| `MF_WEBHOOKS_LOG_LEVEL`        | Log level for Webhooks (debug, info, warn, error)                       | error                 |
| `MF_WEBHOOKS_DB_HOST`          | Database host address                                                   | localhost             |
| `MF_WEBHOOKS_DB_PORT`          | Database host port                                                      | 5432                  |
| `MF_WEBHOOKS_DB_USER`          | Database user                                                           | mainflux              |
| `MF_WEBHOOKS_DB_PASS`          | Database password                                                       | mainflux              |
| `MF_WEBHOOKS_DB`               | Name of the database used by the service                                | webhooks              |
| `MF_WEBHOOKS_DB_SSL_MODE`      | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable               |
| `MF_WEBHOOKS_DB_SSL_CERT`      | Path to the PEM encoded certificate file                                |                       |
| `MF_WEBHOOKS_DB_SSL_KEY`       | Path to the PEM encoded key file                                        |                       |
| `MF_WEBHOOKS_DB_SSL_ROOT_CERT` | Path to the PEM encoded root certificate file                           |                       |
| `MF_WEBHOOKS_CLIENT_TLS`       | Flag that indicates if TLS should be turned on                          | false                 |
| `MF_WEBHOOKS_CA_CERTS`         | Path to trusted CAs in PEM format                                       |                       |
| `MF_WEBHOOKS_HTTP_PORT`        | Webhooks service HTTP port                                              | 9021                  |
| `MF_WEBHOOKS_SERVER_CERT`      | Path to server certificate in PEM format                                |                       |
| `MF_WEBHOOKS_SERVER_KEY`       | Path to server key in PEM format                                        |                       |
| `MF_JAEGER_URL`                | Jaeger server URL                                                       | localhost:6831        |
| `MF_BROKER_URL`                | Message broker URL                                                      | nats://127.0.0.1:4222 |
| `MF_THINGS_AUTH_GRPC_URL`      | Things auth service gRPC URL                                            | localhost:8183        |
| `MF_THINGS_AUTH_GRPC_TIMEOUT`  | Things auth service gRPC request timeout                                | 1s                    |

## Deployment

The service is distributed as a Docker container. Check the [`webhooks`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the webhooks service
make webhooks

# copy binary to bin
make install

# set the environment variables and run the service
MF_WEBHOOKS_LOG_LEVEL=[Webhooks log level] \
MF_WEBHOOKS_DB_HOST=[Database host address] \
MF_WEBHOOKS_DB_PORT=[Database host port] \
MF_WEBHOOKS_DB_USER=[Database user] \
MF_WEBHOOKS_DB_PASS=[Database password] \
MF_WEBHOOKS_DB=[Name of the database used by the service] \
MF_WEBHOOKS_HTTP_PORT=[Service HTTP port] \
MF_BROKER_URL=[Message broker URL] \
MF_THINGS_AUTH_GRPC_URL=[Things auth service gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things auth service gRPC request timeout] \
$GOBIN/mainfluxlabs-webhooks
```

## Usage

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).
