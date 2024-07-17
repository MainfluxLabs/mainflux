# SMPP Notifier

SMPP Notifier implements notifier for send SMS notifications and provides notifier management.

## Configuration

The Subscription service using SMPP Notifier is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                          | Description                                                             | Default               |
|-----------------------------------|-------------------------------------------------------------------------|-----------------------|
| MF_SMPP_NOTIFIER_LOG_LEVEL        | Log level for SMPP Notifier (debug, info, warn, error)                  | error                 |
| MF_JAEGER_URL                     | Jaeger server URL                                                       | localhost:6831        |
| MF_BROKER_URL                     | Message broker URL                                                      | nats://127.0.0.1:4222 |
| MF_SMPP_ADDRESS                   | SMPP address [host:port]                                                |                       |
| MF_SMPP_USERNAME                  | SMPP Username                                                           |                       |
| MF_SMPP_PASSWORD                  | SMPP Password                                                           |                       |
| MF_SMPP_SYSTEM_TYPE               | SMPP System Type                                                        |                       |
| MF_SMPP_SRC_ADDR_TON              | SMPP source address TON                                                 |                       |
| MF_SMPP_DST_ADDR_TON              | SMPP destination address TON                                            |                       |
| MF_SMPP_SRC_ADDR_NPI              | SMPP source address NPI                                                 |                       |
| MF_SMPP_DST_ADDR_NPI              | SMPP destination address NPI                                            |                       |
| MF_SMPP_NOTIFIER_PORT             | SMPP-Notifiers service HTTP port                                        | 8907                  | 
| MF_SMPP_NOTIFIER_SERVER_CERT      | Path to server certificate in pem format                                |                       |
| MF_SMPP_NOTIFIER_SERVER_KEY       | Path to server key in pem format                                        |                       |
| MF_SMPP_NOTIFIER_LOG_LEVEL        | Log level for SMPP-Notifiers (debug, info, warn, error)                 | debug                 |
| MF_SMPP_NOTIFIER_DB_HOST          | Database host address                                                   | localhost             |
| MF_SMPP_NOTIFIER_DB_PORT          | Database host port                                                      | 5432                  |
| MF_SMPP_NOTIFIER_DB_USER          | Database user                                                           | mainflux              |
| MF_SMPP_NOTIFIER_DB_PASS          | Database password                                                       | mainflux              |
| MF_SMPP_NOTIFIER_DB               | Name of the database used by the service                                | smpp-notifiers        |
| MF_SMPP_NOTIFIER_DB_SSL_MODE      | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable               |
| MF_SMPP_NOTIFIER_DB_SSL_CERT      | Path to the PEM encoded certificate file                                |                       |
| MF_SMPP_NOTIFIER_DB_SSL_KEY       | Path to the PEM encoded key file                                        |                       |
| MF_SMPP_NOTIFIER_DB_SSL_ROOT_CERT | Path to the PEM encoded root certificate file                           |                       |
| MF_THINGS_AUTH_GRPC_URL           | Things auth service gRPC URL                                            | localhost:8183        |
| MF_THINGS_AUTH_GRPC_TIMEOUT       | Things auth service gRPC request timeout in seconds                     | 1s                    |
## Usage

Starting service will start consuming messages and sending SMS when a message is received.
 
[doc]: http://mainflux.readthedocs.io
