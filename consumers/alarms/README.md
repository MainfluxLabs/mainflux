# Alarms

Alarms service consumes messages published by the Rules service and persists triggered alarms to a database.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                       | Description                                                             | Default               |
|--------------------------------|-------------------------------------------------------------------------|-----------------------|
| MF_ALARMS_LOG_LEVEL            | Log level for the Alarms service (debug, info, warn, error)             | error                 |
| MF_BROKER_URL                  | Message broker instance URL                                             | nats://localhost:4222 |
| MF_ALARMS_HTTP_PORT            | Alarms service HTTP port                                                | 9026                  |
| MF_JAEGER_URL                  | Jaeger server URL                                                       |                       |
| MF_ALARMS_DB_HOST              | Database host address                                                   | localhost             |
| MF_ALARMS_DB_PORT              | Database host port                                                      | 5432                  |
| MF_ALARMS_DB_USER              | Database user                                                           | mainflux              |
| MF_ALARMS_DB_PASS              | Database password                                                       | mainflux              |
| MF_ALARMS_DB                   | Name of the database used by the service                                | alarms                |
| MF_ALARMS_DB_SSL_MODE          | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable               |
| MF_ALARMS_DB_SSL_CERT          | Path to the PEM encoded certificate file                                |                       |
| MF_ALARMS_DB_SSL_KEY           | Path to the PEM encoded key file                                        |                       |
| MF_ALARMS_DB_SSL_ROOT_CERT     | Path to the PEM encoded root certificate file                           |                       |
| MF_ALARMS_CLIENT_TLS           | Flag that indicates if TLS should be turned on                          | false                 |
| MF_ALARMS_CA_CERTS             | Path to trusted CAs in PEM format                                       |                       |
| MF_ALARMS_SERVER_CERT          | Path to server certificate in PEM format                                |                       |
| MF_ALARMS_SERVER_KEY           | Path to server key in PEM format                                        |                       |
| MF_THINGS_AUTH_GRPC_URL        | Things service Auth gRPC URL                                            | localhost:8183        |
| MF_THINGS_AUTH_GRPC_TIMEOUT    | Things service Auth gRPC request timeout in seconds                     | 1s                    |
| MF_ALARMS_ES_URL               | Event store URL                                                         | redis://localhost:6379/0 |
| MF_ALARMS_EVENT_CONSUMER       | Event store consumer name                                               | alarms                |

## Deployment

The service itself is distributed as Docker container. Check the [`alarms`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the alarms service
make alarms

# copy binary to bin
make install

# Set the environment variables and run the service
MF_ALARMS_LOG_LEVEL=[Alarms log level] \
MF_BROKER_URL=[Message broker instance URL] \
MF_ALARMS_HTTP_PORT=[Alarms service HTTP port] \
MF_ALARMS_DB_HOST=[Database host address] \
MF_ALARMS_DB_PORT=[Database host port] \
MF_ALARMS_DB_USER=[Database user] \
MF_ALARMS_DB_PASS=[Database password] \
MF_ALARMS_DB=[Alarms database name] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout] \
$GOBIN/mainfluxlabs-alarms
```

## Usage

Starting the service will begin consuming alarm messages from the message broker and persisting them to the database. For more information about service capabilities and its usage, please check out the [API documentation](https://github.com/MainfluxLabs/mainflux/blob/master/api/openapi/alarms.yml).

[doc]: https://mainfluxlabs.github.io/docs
