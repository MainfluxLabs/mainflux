# Filestore Service

The Filestore service provides file storage for things and groups. File contents are written to the local filesystem, while file metadata is persisted to a database. Files can be scoped to an individual thing (authenticated with a thing key) or to a group (authenticated with a user bearer token at editor level or above).

## Files

Each stored file is described by the following metadata fields:

| Field      | Description                                                                  |
|------------|------------------------------------------------------------------------------|
| `name`     | File name; used as the unique identifier within the thing or group scope     |
| `class`    | Logical file class (e.g. `image`, `document`, `bim`, `pointcloud`, `binary`) |
| `format`   | File format / MIME subtype (e.g. `csv`, `pdf`, `png`, `ifc`)                 |
| `time`     | Unix timestamp (floating-point seconds) associated with the file             |
| `metadata` | Arbitrary key-value pairs for custom attributes                              |

## Scopes

| Scope       | Auth header                        | Description                                              |
|-------------|------------------------------------|----------------------------------------------------------|
| Thing files | `Authorization: Thing <thing_key>` | Files private to a specific thing                        |
| Group files | `Authorization: Bearer <token>`    | Files shared across a group; require editor-level access |

A thing can also retrieve its own group's files directly using its thing key via `GET /groupfiles/{name}`.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                        | Description                                                                | Default                  |
|---------------------------------|----------------------------------------------------------------------------|--------------------------|
| `MF_FILESTORE_LOG_LEVEL`        | Log level for the Filestore service (debug, info, warn, error)             | error                    |
| `MF_FILESTORE_HTTP_PORT`        | Filestore service HTTP port                                                | 9024                     |
| `MF_JAEGER_URL`                 | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                          |
| `MF_FILESTORE_DB_HOST`          | Database host address                                                      | localhost                |
| `MF_FILESTORE_DB_PORT`          | Database host port                                                         | 5432                     |
| `MF_FILESTORE_DB_USER`          | Database user                                                              | mainflux                 |
| `MF_FILESTORE_DB_PASS`          | Database password                                                          | mainflux                 |
| `MF_FILESTORE_DB`               | Name of the database used by the service                                   | filestore                |
| `MF_FILESTORE_DB_SSL_MODE`      | Database connection SSL mode (disable, require, verify-ca, verify-full)    | disable                  |
| `MF_FILESTORE_DB_SSL_CERT`      | Path to the PEM encoded certificate file                                   |                          |
| `MF_FILESTORE_DB_SSL_KEY`       | Path to the PEM encoded key file                                           |                          |
| `MF_FILESTORE_DB_SSL_ROOT_CERT` | Path to the PEM encoded root certificate file                              |                          |
| `MF_FILESTORE_TLS`              | Flag that indicates if TLS should be turned on                             | false                    |
| `MF_FILESTORE_CA_CERTS`         | Path to trusted CAs in PEM format                                          |                          |
| `MF_FILESTORE_SERVER_CERT`      | Path to server certificate in PEM format                                   |                          |
| `MF_FILESTORE_SERVER_KEY`       | Path to server key in PEM format                                           |                          |
| `MF_THINGS_AUTH_GRPC_URL`       | Things service Auth gRPC URL                                               | localhost:8183           |
| `MF_THINGS_AUTH_GRPC_TIMEOUT`   | Things service Auth gRPC request timeout in seconds                        | 1s                       |
| `MF_FILESTORE_ES_URL`           | Event store URL                                                            | redis://localhost:6379/0 |
| `MF_FILESTORE_EVENT_CONSUMER`   | Event store consumer name                                                  | filestore                |

## Deployment

The service itself is distributed as Docker container. Check the [`filestore`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# Download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the filestore service
make filestore

# Copy binary to bin
make install

# Set the environment variables and run the service
MF_FILESTORE_LOG_LEVEL=[Filestore log level] \
MF_FILESTORE_HTTP_PORT=[Filestore service HTTP port] \
MF_FILESTORE_DB_HOST=[Database host address] \
MF_FILESTORE_DB_PORT=[Database host port] \
MF_FILESTORE_DB_USER=[Database user] \
MF_FILESTORE_DB_PASS=[Database password] \
MF_FILESTORE_DB=[Filestore database name] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout] \
$GOBIN/mainfluxlabs-filestore
```

## Usage

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).
