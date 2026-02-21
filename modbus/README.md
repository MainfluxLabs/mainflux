# Modbus

Modbus service manages Modbus TCP clients for things and groups, polling registers or coils on a schedule and publishing the results as messages to the platform.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                       | Description                                                             | Default               |
|--------------------------------|-------------------------------------------------------------------------|-----------------------|
| MF_MODBUS_LOG_LEVEL            | Log level for the Modbus service (debug, info, warn, error)             | error                 |
| MF_BROKER_URL                  | Message broker instance URL                                             | nats://localhost:4222 |
| MF_MODBUS_HTTP_PORT            | Modbus service HTTP port                                                | 9028                  |
| MF_JAEGER_URL                  | Jaeger server URL                                                       |                       |
| MF_MODBUS_DB_HOST              | Database host address                                                   | localhost             |
| MF_MODBUS_DB_PORT              | Database host port                                                      | 5432                  |
| MF_MODBUS_DB_USER              | Database user                                                           | mainflux              |
| MF_MODBUS_DB_PASS              | Database password                                                       | mainflux              |
| MF_MODBUS_DB                   | Name of the database used by the service                                | modbus                |
| MF_MODBUS_DB_SSL_MODE          | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable               |
| MF_MODBUS_DB_SSL_CERT          | Path to the PEM encoded certificate file                                |                       |
| MF_MODBUS_DB_SSL_KEY           | Path to the PEM encoded key file                                        |                       |
| MF_MODBUS_DB_SSL_ROOT_CERT     | Path to the PEM encoded root certificate file                           |                       |
| MF_MODBUS_CLIENT_TLS           | Flag that indicates if TLS should be turned on                          | false                 |
| MF_MODBUS_CA_CERTS             | Path to trusted CAs in PEM format                                       |                       |
| MF_MODBUS_SERVER_CERT          | Path to server certificate in PEM format                                |                       |
| MF_MODBUS_SERVER_KEY           | Path to server key in PEM format                                        |                       |
| MF_THINGS_AUTH_GRPC_URL        | Things service Auth gRPC URL                                            | localhost:8183        |
| MF_THINGS_AUTH_GRPC_TIMEOUT    | Things service Auth gRPC request timeout in seconds                     | 1s                    |
| MF_MODBUS_ES_URL               | Event store URL                                                         | redis://localhost:6379/0 |
| MF_MODBUS_EVENT_CONSUMER       | Event store consumer name                                               | modbus                |

## Deployment

The service itself is distributed as Docker container. Check the [`modbus`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the modbus service
make modbus

# copy binary to bin
make install

# Set the environment variables and run the service
MF_MODBUS_LOG_LEVEL=[Modbus log level] \
MF_BROKER_URL=[Message broker instance URL] \
MF_MODBUS_HTTP_PORT=[Modbus service HTTP port] \
MF_MODBUS_DB_HOST=[Database host address] \
MF_MODBUS_DB_PORT=[Database host port] \
MF_MODBUS_DB_USER=[Database user] \
MF_MODBUS_DB_PASS=[Database password] \
MF_MODBUS_DB=[Modbus database name] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout] \
$GOBIN/mainfluxlabs-modbus
```

## Usage

Starting the service will load all persisted Modbus clients and begin scheduling polls according to each client's configured frequency (repeating cron or one-time). Each poll reads the specified registers or coils from the Modbus TCP device using the configured function code, formats the result as a JSON payload, and publishes it as a platform message scoped to the associated thing. Supported data types are `bool`, `int16`, `uint16`, `int32`, `uint32`, `float32`, and `string`. Byte order (ABCD, DCBA, CDAB, BADC) can be configured per data field.

[doc]: https://mainfluxlabs.github.io/docs
