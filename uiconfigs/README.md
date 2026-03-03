# UI Configs

UI Configs service persists and manages UI configuration settings scoped per organization and per thing, with support for full backup and restore.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                         | Description                                                             | Default               |
|----------------------------------|-------------------------------------------------------------------------|-----------------------|
| MF_UI_CONFIGS_LOG_LEVEL          | Log level for the UI Configs service (debug, info, warn, error)         | error                 |
| MF_UI_CONFIGS_HTTP_PORT          | UI Configs service HTTP port                                            | 9029                  |
| MF_JAEGER_URL                    | Jaeger server URL                                                       |                       |
| MF_UI_CONFIGS_DB_HOST            | Database host address                                                   | localhost             |
| MF_UI_CONFIGS_DB_PORT            | Database host port                                                      | 5432                  |
| MF_UI_CONFIGS_DB_USER            | Database user                                                           | mainflux              |
| MF_UI_CONFIGS_DB_PASS            | Database password                                                       | mainflux              |
| MF_UI_CONFIGS_DB                 | Name of the database used by the service                                | uiconfigs             |
| MF_UI_CONFIGS_DB_SSL_MODE        | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable               |
| MF_UI_CONFIGS_DB_SSL_CERT        | Path to the PEM encoded certificate file                                |                       |
| MF_UI_CONFIGS_DB_SSL_KEY         | Path to the PEM encoded key file                                        |                       |
| MF_UI_CONFIGS_DB_SSL_ROOT_CERT   | Path to the PEM encoded root certificate file                           |                       |
| MF_UI_CONFIGS_CLIENT_TLS         | Flag that indicates if TLS should be turned on                          | false                 |
| MF_UI_CONFIGS_CA_CERTS           | Path to trusted CAs in PEM format                                       |                       |
| MF_UI_CONFIGS_SERVER_CERT        | Path to server certificate in PEM format                                |                       |
| MF_UI_CONFIGS_SERVER_KEY         | Path to server key in PEM format                                        |                       |
| MF_THINGS_AUTH_GRPC_URL          | Things service Auth gRPC URL                                            | localhost:8183        |
| MF_THINGS_AUTH_GRPC_TIMEOUT      | Things service Auth gRPC request timeout in seconds                     | 1s                    |
| MF_AUTH_GRPC_URL                 | Auth service gRPC URL                                                   | localhost:8181        |
| MF_AUTH_GRPC_TIMEOUT             | Auth service gRPC request timeout in seconds                            | 1s                    |
| MF_UI_CONFIGS_ES_URL             | Event store URL                                                         | redis://localhost:6379/0 |
| MF_UI_CONFIGS_EVENT_CONSUMER     | Event store consumer name                                               | uiconfigs             |

## Deployment

The service itself is distributed as Docker container. Check the [`uiconfigs`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the uiconfigs service
make uiconfigs

# copy binary to bin
make install

# Set the environment variables and run the service
MF_UI_CONFIGS_LOG_LEVEL=[UI Configs log level] \
MF_UI_CONFIGS_HTTP_PORT=[UI Configs service HTTP port] \
MF_UI_CONFIGS_DB_HOST=[Database host address] \
MF_UI_CONFIGS_DB_PORT=[Database host port] \
MF_UI_CONFIGS_DB_USER=[Database user] \
MF_UI_CONFIGS_DB_PASS=[Database password] \
MF_UI_CONFIGS_DB=[UI Configs database name] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout] \
MF_AUTH_GRPC_URL=[Auth service gRPC URL] \
MF_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout] \
$GOBIN/mainfluxlabs-uiconfigs
```

## Usage

The service stores UI configuration per organization (org config) and per thing (thing config). Users can view or update their own org and thing configs. Administrators can list all configs. The service also exposes backup and restore endpoints to export and re-import the full configuration state.

[doc]: https://mainfluxlabs.github.io/docs
