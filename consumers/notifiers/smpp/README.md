# SMPP Notifier

SMPP Notifier implements notifier for send SMS notifications and provides notifier management.

## Configuration

The Subscription service using SMPP Notifier is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                            | Description                                                                | Default                  |
|-------------------------------------|----------------------------------------------------------------------------|--------------------------|
| `MF_SMPP_NOTIFIER_LOG_LEVEL`        | Log level for SMPP Notifier (debug, info, warn, error)                     | error                    |
| `MF_JAEGER_URL`                     | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                          |
| `MF_BROKER_URL`                     | Message broker URL                                                         | nats://localhost:4222    |
| `MF_SMPP_ADDRESS`                   | SMPP address [host:port]                                                   |                          |
| `MF_SMPP_USERNAME`                  | SMPP Username                                                              |                          |
| `MF_SMPP_PASSWORD`                  | SMPP Password                                                              |                          |
| `MF_SMPP_SYSTEM_TYPE`               | SMPP System Type                                                           |                          |
| `MF_SMPP_SRC_ADDR_TON`              | SMPP source address TON                                                    | 0                        |
| `MF_SMPP_DST_ADDR_TON`              | SMPP destination address TON                                               | 0                        |
| `MF_SMPP_SRC_ADDR_NPI`              | SMPP source address NPI                                                    | 0                        |
| `MF_SMPP_DST_ADDR_NPI`              | SMPP destination address NPI                                               | 0                        |
| `MF_SMPP_NOTIFIER_SOURCE_ADDR`      | SMS sender address                                                         |                          |
| `MF_SMPP_NOTIFIER_PORT`             | SMPP-Notifiers service HTTP port                                           | 9024                     |
| `MF_SMPP_NOTIFIER_SERVER_CERT`      | Path to server certificate in pem format                                   |                          |
| `MF_SMPP_NOTIFIER_SERVER_KEY`       | Path to server key in pem format                                           |                          |
| `MF_SMPP_NOTIFIER_THINGS_TLS`       | Flag that indicates if TLS should be turned on for Things gRPC             | false                    |
| `MF_SMPP_NOTIFIER_THINGS_CA_CERTS`  | Path to trusted CAs in PEM format for Things gRPC                          |                          |
| `MF_SMPP_NOTIFIER_DB_HOST`          | Database host address                                                      | localhost                |
| `MF_SMPP_NOTIFIER_DB_PORT`          | Database host port                                                         | 5432                     |
| `MF_SMPP_NOTIFIER_DB_USER`          | Database user                                                              | mainflux                 |
| `MF_SMPP_NOTIFIER_DB_PASS`          | Database password                                                          | mainflux                 |
| `MF_SMPP_NOTIFIER_DB`               | Name of the database used by the service                                   | smpp-notifiers           |
| `MF_SMPP_NOTIFIER_DB_SSL_MODE`      | Database connection SSL mode (disable, require, verify-ca, verify-full)    | disable                  |
| `MF_SMPP_NOTIFIER_DB_SSL_CERT`      | Path to the PEM encoded certificate file                                   |                          |
| `MF_SMPP_NOTIFIER_DB_SSL_KEY`       | Path to the PEM encoded key file                                           |                          |
| `MF_SMPP_NOTIFIER_DB_SSL_ROOT_CERT` | Path to the PEM encoded root certificate file                              |                          |
| `MF_SMPP_NOTIFIER_ES_URL`           | Event store URL                                                            | redis://localhost:6379/0 |
| `MF_SMPP_NOTIFIER_EVENT_CONSUMER`   | Event store consumer name                                                  | smpp-notifier            |
| `MF_THINGS_AUTH_GRPC_URL`           | Things auth service gRPC URL                                               | localhost:8183           |
| `MF_THINGS_AUTH_GRPC_TIMEOUT`       | Things auth service gRPC request timeout in seconds                        | 1s                       |
| `MF_AUTH_GRPC_URL`                  | Auth service gRPC URL                                                      | localhost:8181           |
| `MF_AUTH_GRPC_TIMEOUT`              | Auth service gRPC request timeout                                          | 1s                       |

## Usage

Starting the service will begin consuming messages from the broker. When a message arrives for a group that has notifiers with phone number contacts, the service sends an SMS to each contact via the configured SMPP gateway.

For more information about notifier management, see the [Notifiers service documentation](../README.md) and the [API documentation](https://mainfluxlabs.github.io/docs/swagger/).

