# Readers

Readers provide an implementation of various `message readers`. The service exposes an HTTP API for querying messages that have been persisted by the Writers (consumers).

## Implementations

Two backend implementations are provided, each as its own binary:

| Implementation | Binary                          | Description                                    |
|----------------|---------------------------------|------------------------------------------------|
| PostgreSQL     | `mainfluxlabs-postgres-reader`  | Reads SenML and JSON messages from PostgreSQL  |
| TimescaleDB    | `mainfluxlabs-timescale-reader` | Reads SenML and JSON messages from TimescaleDB |

## Message Formats

The service supports two message formats:

**SenML** — structured IoT measurement records with well-known fields (`n`, `v`, `vb`, `vs`, `vd`, `t`, `u`).

**JSON** — free-form JSON records, queried by arbitrary filter expressions.

## Authentication

Requests are authenticated using the Thing key:

```
Authorization: Thing <thing_key>
```

The authenticated thing determines which messages are accessible.

## Configuration

### PostgreSQL Reader

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                              | Description                                                                | Default        |
|---------------------------------------|----------------------------------------------------------------------------|----------------|
| `MF_POSTGRES_READER_LOG_LEVEL`        | Log level (debug, info, warn, error)                                       | error          |
| `MF_POSTGRES_READER_PORT`             | HTTP port                                                                  | 8180           |
| `MF_POSTGRES_READER_CLIENT_TLS`       | Flag that indicates if TLS should be turned on                             | false          |
| `MF_POSTGRES_READER_CA_CERTS`         | Path to trusted CAs in PEM format                                          |                |
| `MF_POSTGRES_READER_SERVER_CERT`      | Path to server certificate in PEM format                                   |                |
| `MF_POSTGRES_READER_SERVER_KEY`       | Path to server key in PEM format                                           |                |
| `MF_POSTGRES_READER_DB_HOST`          | Database host address                                                      | localhost      |
| `MF_POSTGRES_READER_DB_PORT`          | Database host port                                                         | 5432           |
| `MF_POSTGRES_READER_DB_USER`          | Database user                                                              | mainflux       |
| `MF_POSTGRES_READER_DB_PASS`          | Database password                                                          | mainflux       |
| `MF_POSTGRES_READER_DB`               | Database name                                                              | mainflux       |
| `MF_POSTGRES_READER_DB_SSL_MODE`      | Database connection SSL mode (disable, require, verify-ca, verify-full)    | disable        |
| `MF_POSTGRES_READER_DB_SSL_CERT`      | Path to the PEM encoded certificate file                                   |                |
| `MF_POSTGRES_READER_DB_SSL_KEY`       | Path to the PEM encoded key file                                           |                |
| `MF_POSTGRES_READER_DB_SSL_ROOT_CERT` | Path to the PEM encoded root certificate file                              |                |
| `MF_THINGS_AUTH_GRPC_URL`             | Things service Auth gRPC URL                                               | localhost:8183 |
| `MF_THINGS_AUTH_GRPC_TIMEOUT`         | Things service Auth gRPC request timeout in seconds                        | 1s             |
| `MF_AUTH_GRPC_URL`                    | Auth service gRPC URL                                                      | localhost:8181 |
| `MF_AUTH_GRPC_TIMEOUT`                | Auth service gRPC request timeout                                          | 1s             |
| `MF_JAEGER_URL`                       | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                |

### TimescaleDB Reader

Uses the same environment variable names with `MF_TIMESCALE_READER_` prefix (except shared vars like `MF_THINGS_AUTH_GRPC_URL`).


## Deployment

The service itself is distributed as Docker container. Check the [`postgres-reader`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how the service is deployed.

To start the PostgreSQL reader service, execute the following shell script:

```bash
# Download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the postgres-reader service
make postgres-reader

# Copy binary to bin
make install

# Set the environment variables and run the service
MF_POSTGRES_READER_LOG_LEVEL=[Log level] \
MF_POSTGRES_READER_PORT=[HTTP port] \
MF_POSTGRES_READER_DB_HOST=[Database host address] \
MF_POSTGRES_READER_DB_PORT=[Database host port] \
MF_POSTGRES_READER_DB_USER=[Database user] \
MF_POSTGRES_READER_DB_PASS=[Database password] \
MF_POSTGRES_READER_DB=[Database name] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
$GOBIN/mainfluxlabs-postgres-reader
```

## Usage

Starting the service exposes the HTTP API for querying persisted messages. Authentication is performed using the Thing key passed in the `Authorization` header.

For the full API reference, see the [API documentation](https://mainfluxlabs.github.io/docs/swagger/).

