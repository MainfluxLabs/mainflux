# Rules

Rules service provides a rule engine that evaluates conditions on incoming messages and triggers actions such as alarms, SMTP notifications, or SMPP notifications.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                       | Description                                                             | Default               |
|--------------------------------|-------------------------------------------------------------------------|-----------------------|
| MF_RULES_LOG_LEVEL             | Log level for the Rules service (debug, info, warn, error)              | error                 |
| MF_BROKER_URL                  | Message broker instance URL                                             | nats://localhost:4222 |
| MF_RULES_HTTP_PORT             | Rules service HTTP port                                                 | 9027                  |
| MF_JAEGER_URL                  | Jaeger server URL                                                       |                       |
| MF_RULES_DB_HOST               | Database host address                                                   | localhost             |
| MF_RULES_DB_PORT               | Database host port                                                      | 5432                  |
| MF_RULES_DB_USER               | Database user                                                           | mainflux              |
| MF_RULES_DB_PASS               | Database password                                                       | mainflux              |
| MF_RULES_DB                    | Name of the database used by the service                                | rules                 |
| MF_RULES_DB_SSL_MODE           | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable               |
| MF_RULES_DB_SSL_CERT           | Path to the PEM encoded certificate file                                |                       |
| MF_RULES_DB_SSL_KEY            | Path to the PEM encoded key file                                        |                       |
| MF_RULES_DB_SSL_ROOT_CERT      | Path to the PEM encoded root certificate file                           |                       |
| MF_RULES_CLIENT_TLS            | Flag that indicates if TLS should be turned on                          | false                 |
| MF_RULES_CA_CERTS              | Path to trusted CAs in PEM format                                       |                       |
| MF_RULES_SERVER_CERT           | Path to server certificate in PEM format                                |                       |
| MF_RULES_SERVER_KEY            | Path to server key in PEM format                                        |                       |
| MF_THINGS_AUTH_GRPC_URL        | Things service Auth gRPC URL                                            | localhost:8183        |
| MF_THINGS_AUTH_GRPC_TIMEOUT    | Things service Auth gRPC request timeout in seconds                     | 1s                    |
| MF_RULES_ES_URL                | Event store URL                                                         | redis://localhost:6379/0 |
| MF_RULES_EVENT_CONSUMER        | Event store consumer name                                               | rules                 |

## Deployment

The service itself is distributed as Docker container. Check the [`rules`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the rules service
make rules

# copy binary to bin
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

Starting the service will begin consuming messages from the message broker. Each incoming message is evaluated against the rules assigned to the publishing thing. When all conditions of a rule are met (using AND or OR logic), the service publishes an action message to the appropriate subject — triggering alarms, SMTP notifications, or SMPP notifications. For more information about service capabilities and its usage, please check out the [API documentation](https://github.com/MainfluxLabs/mainflux/blob/master/api/openapi/rules.yml).

[doc]: https://mainfluxlabs.github.io/docs
