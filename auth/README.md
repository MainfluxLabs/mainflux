# Auth Service

The Auth service is the central authentication and authorization component of the Mainflux IoT platform. It provides identity verification, role-based access control (RBAC), multi-tenant organization management, and a member invitation system. All other platform services communicate with Auth over its gRPC API to authenticate requests and enforce access policies.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                       | Description                                                                 | Default                  |
|--------------------------------|-----------------------------------------------------------------------------|--------------------------|
| `MF_AUTH_LOG_LEVEL`            | Service log level (debug, info, warn, error)                                | error                    |
| `MF_AUTH_DB_HOST`              | Database host address                                                       | localhost                |
| `MF_AUTH_DB_PORT`              | Database host port                                                          | 5432                     |
| `MF_AUTH_DB_USER`              | Database user                                                               | mainflux                 |
| `MF_AUTH_DB_PASS`              | Database password                                                           | mainflux                 |
| `MF_USERS_ADMIN_EMAIL`         | Email of the default root admin user (created on first startup)             |                          |
| `MF_AUTH_DB`                   | Name of the database used by the service                                    | auth                     |
| `MF_AUTH_DB_SSL_MODE`          | Database connection SSL mode (disable, require, verify-ca, verify-full)     | disable                  |
| `MF_AUTH_DB_SSL_CERT`          | Path to the PEM encoded certificate file                                    |                          |
| `MF_AUTH_DB_SSL_KEY`           | Path to the PEM encoded key file                                            |                          |
| `MF_AUTH_DB_SSL_ROOT_CERT`     | Path to the PEM encoded root certificate file                               |                          |
| `MF_AUTH_HTTP_PORT`            | Auth service HTTP port                                                      | 8180                     |
| `MF_AUTH_GRPC_PORT`            | Auth service gRPC port                                                      | 8181                     |
| `MF_AUTH_SERVER_CERT`          | Path to server certificate in PEM format                                    |                          |
| `MF_AUTH_SERVER_KEY`           | Path to server key in PEM format                                            |                          |
| `MF_AUTH_SECRET`               | Secret string used for signing tokens                                       | auth                     |
| `MF_AUTH_LOGIN_TOKEN_DURATION` | Login key expiration period                                                 | 10h                      |
| `MF_INVITE_DURATION`           | Validity period for organization invitations                                | 168h                     |
| `MF_JAEGER_URL`                | Jaeger server URL for distributed tracing. Leave empty to disable tracing.  |                          |
| `MF_AUTH_ES_URL`               | Event store (Redis) URL                                                     | redis://localhost:6379/0 |
| `MF_AUTH_GRPC_TIMEOUT`         | Timeout for outgoing gRPC calls (Things and Users services)                 | 1s                       |
| `MF_THINGS_AUTH_GRPC_URL`      | Things service auth gRPC URL                                                | localhost:8183           |
| `MF_THINGS_CLIENT_TLS`         | Enable TLS for Things gRPC connection                                       | false                    |
| `MF_THINGS_CA_CERTS`           | Path to trusted CAs in PEM format for Things gRPC TLS                       |                          |
| `MF_USERS_GRPC_URL`            | Users service gRPC URL                                                      | localhost:8184           |
| `MF_USERS_CLIENT_TLS`          | Enable TLS for Users gRPC connection                                        | false                    |
| `MF_USERS_CA_CERTS`            | Path to trusted CAs in PEM format for Users gRPC TLS                        |                          |
| `MF_HOST`                      | Frontend URL base used in invitation email links                            | http://localhost         |
| `MF_EMAIL_HOST`                | Mail server host                                                            | localhost                |
| `MF_EMAIL_PORT`                | Mail server port                                                            | 25                       |
| `MF_EMAIL_USERNAME`            | Mail server username                                                        |                          |
| `MF_EMAIL_PASSWORD`            | Mail server password                                                        |                          |
| `MF_EMAIL_FROM_ADDRESS`        | Sender email address                                                        |                          |
| `MF_EMAIL_FROM_NAME`           | Sender display name                                                         |                          |
| `MF_EMAIL_TEMPLATES_DIR`       | Path to the directory containing email templates used for invitation emails | `.`                      |

> **Note:** If `MF_EMAIL_TEMPLATES_DIR` does not point to a valid directory containing the required templates, the service will start normally but invitation emails will not be sent.

## Deployment

The service is distributed as a Docker container. Refer to the [`auth` service section](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) in the Docker Compose file for a reference deployment configuration.

To build and run the service manually:

```bash
# Download the latest version
go get github.com/MainfluxLabs/mainflux

cd $GOPATH/src/github.com/MainfluxLabs/mainflux

# Compile the service
make auth

# Copy binary to bin
make install

# Run the service
MF_AUTH_LOG_LEVEL=[log level] \
MF_AUTH_DB_HOST=[db host] \
MF_AUTH_DB_PORT=[db port] \
MF_AUTH_DB_USER=[db user] \
MF_AUTH_DB_PASS=[db password] \
MF_AUTH_DB=[db name] \
MF_AUTH_DB_SSL_MODE=[ssl mode] \
MF_AUTH_HTTP_PORT=[http port] \
MF_AUTH_GRPC_PORT=[grpc port] \
MF_AUTH_SECRET=[signing secret] \
MF_AUTH_LOGIN_TOKEN_DURATION=[token duration] \
$GOBIN/mainfluxlabs-auth
```

## Usage

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).

For a broader overview of how authentication and authorization fit into the platform, refer to the [official documentation](https://mainfluxlabs.github.io/docs).
