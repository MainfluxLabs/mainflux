# Downlinks

Downlinks service manages scheduled outbound HTTP requests (downlinks) for things and groups, supporting cron-based and one-time scheduling with optional time-range parameter injection.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                        | Description                                                             | Default               |
|---------------------------------|-------------------------------------------------------------------------|-----------------------|
| MF_DOWNLINKS_LOG_LEVEL          | Log level for the Downlinks service (debug, info, warn, error)          | error                 |
| MF_BROKER_URL                   | Message broker instance URL                                             | nats://localhost:4222 |
| MF_DOWNLINKS_HTTP_PORT          | Downlinks service HTTP port                                             | 9025                  |
| MF_JAEGER_URL                   | Jaeger server URL                                                       |                       |
| MF_DOWNLINKS_DB_HOST            | Database host address                                                   | localhost             |
| MF_DOWNLINKS_DB_PORT            | Database host port                                                      | 5432                  |
| MF_DOWNLINKS_DB_USER            | Database user                                                           | mainflux              |
| MF_DOWNLINKS_DB_PASS            | Database password                                                       | mainflux              |
| MF_DOWNLINKS_DB                 | Name of the database used by the service                                | downlinks             |
| MF_DOWNLINKS_DB_SSL_MODE        | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable               |
| MF_DOWNLINKS_DB_SSL_CERT        | Path to the PEM encoded certificate file                                |                       |
| MF_DOWNLINKS_DB_SSL_KEY         | Path to the PEM encoded key file                                        |                       |
| MF_DOWNLINKS_DB_SSL_ROOT_CERT   | Path to the PEM encoded root certificate file                           |                       |
| MF_DOWNLINKS_CLIENT_TLS         | Flag that indicates if TLS should be turned on                          | false                 |
| MF_DOWNLINKS_CA_CERTS           | Path to trusted CAs in PEM format                                       |                       |
| MF_DOWNLINKS_SERVER_CERT        | Path to server certificate in PEM format                                |                       |
| MF_DOWNLINKS_SERVER_KEY         | Path to server key in PEM format                                        |                       |
| MF_THINGS_AUTH_GRPC_URL         | Things service Auth gRPC URL                                            | localhost:8183        |
| MF_THINGS_AUTH_GRPC_TIMEOUT     | Things service Auth gRPC request timeout in seconds                     | 1s                    |
| MF_AUTH_GRPC_URL                | Auth service gRPC URL                                                   | localhost:8181        |
| MF_AUTH_GRPC_TIMEOUT            | Auth service gRPC request timeout in seconds                            | 1s                    |
| MF_DOWNLINKS_ES_URL             | Event store URL                                                         | redis://localhost:6379/0 |
| MF_DOWNLINKS_EVENT_CONSUMER     | Event store consumer name                                               | downlinks             |

## Deployment

The service itself is distributed as Docker container. Check the [`downlinks`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the downlinks service
make downlinks

# copy binary to bin
make install

# Set the environment variables and run the service
MF_DOWNLINKS_LOG_LEVEL=[Downlinks log level] \
MF_BROKER_URL=[Message broker instance URL] \
MF_DOWNLINKS_HTTP_PORT=[Downlinks service HTTP port] \
MF_DOWNLINKS_DB_HOST=[Database host address] \
MF_DOWNLINKS_DB_PORT=[Database host port] \
MF_DOWNLINKS_DB_USER=[Database user] \
MF_DOWNLINKS_DB_PASS=[Database password] \
MF_DOWNLINKS_DB=[Downlinks database name] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout] \
MF_AUTH_GRPC_URL=[Auth service gRPC URL] \
MF_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout] \
$GOBIN/mainfluxlabs-downlinks
```

## Usage

Starting the service will load all persisted downlinks and schedule them according to their configured frequency (repeating cron or one-time). Each scheduled execution sends an HTTP request to the configured URL with the specified method, headers, and payload. Time filter parameters can be injected automatically into the request based on the schedule interval.

[doc]: https://mainfluxlabs.github.io/docs
