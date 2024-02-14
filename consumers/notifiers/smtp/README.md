# SMTP Notifier

SMTP Notifier implements notifier for send SMTP notifications.

## Configuration

The Subscription service using SMTP Notifier is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                          | Description                                                             | Default               |
| --------------------------------- | ----------------------------------------------------------------------- | --------------------- |
| MF_JAEGER_URL                     | Jaeger server URL                                                       | localhost:6831        |
| MF_BROKER_URL                     | Message broker URL                                                      | nats://127.0.0.1:4222 |
| MF_EMAIL_HOST                     | Mail server host                                                        | localhost             |
| MF_EMAIL_PORT                     | Mail server port                                                        | 25                    |
| MF_EMAIL_USERNAME                 | Mail server username                                                    |                       |
| MF_EMAIL_PASSWORD                 | Mail server password                                                    |                       |
| MF_EMAIL_FROM_ADDRESS             | Email "from" address                                                    |                       |
| MF_EMAIL_FROM_NAME                | Email "from" name                                                       |                       |
| MF_EMAIL_TEMPLATE                 | Email template for sending notification emails                          | email.tmpl            |
| MF_AUTH_GRPC_URL                  | Auth service gRPC URL                                                   | localhost:8181        |
| MF_AUTH_GRPC_TIMEOUT              | Auth service gRPC request timeout in seconds                            | 1s                    |
| MF_AUTH_CLIENT_TLS                | Auth client TLS flag                                                    | false                 |
| MF_AUTH_CA_CERTS                  | Path to Auth client CA certs in pem format                              |                       |

## Usage

Starting service will start consuming messages and sending emails when a message is received.

[doc]: https://mainfluxlabs.github.io/docs
