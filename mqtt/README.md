# MQTT adapter

MQTT adapter provides an MQTT API for sending and receiving messages through the platform. It uses [mProxy](https://github.com/MainfluxLabs/mproxy) to proxy traffic between MQTT clients and the underlying MQTT broker, intercepting connections to authenticate things and forward messages to the internal message broker.

## MQTT Topics

Messages are published and subscribed to using the following topic format:

```
messages/<subtopic>
```

`<subtopic>` is optional and can be any path-like string (e.g. `messages/sensors/temperature`).

## Authentication

Things authenticate over MQTT using:

- **Username**: Thing key type — `internal` or `external`
- **Password**: Thing key value

## Ports

| Port | Protocol | Description                      |
|------|----------|----------------------------------|
| 1883 | MQTT     | Plain MQTT                       |
| 8883 | MQTTS    | MQTT over TLS (requires `MF_MQTT_ADAPTER_SERVER_CERT` and `MF_MQTT_ADAPTER_SERVER_KEY`) |
| 8285 | WS       | MQTT over WebSocket              |

## Subscriptions

The adapter persists active MQTT subscriptions to PostgreSQL (database `subscriptions` by default). The schema:

```sql
CREATE TABLE IF NOT EXISTS subscriptions (
    topic       VARCHAR(1024),
    group_id    UUID,
    thing_id    UUID,
    client_id   VARCHAR(256),
    status      VARCHAR(128),
    created_at  FLOAT,
    PRIMARY KEY (client_id, topic, group_id, thing_id)
);
```

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                                 | Description                                                             | Default                  |
|------------------------------------------|-------------------------------------------------------------------------|--------------------------|
| MF_MQTT_ADAPTER_LOG_LEVEL                | Log level (debug, info, warn, error)                                    | error                    |
| MF_MQTT_ADAPTER_MQTT_PORT                | Listening MQTT port                                                     | 1883                     |
| MF_MQTT_ADAPTER_MQTT_TARGET_HOST         | Upstream MQTT broker host                                               | 0.0.0.0                  |
| MF_MQTT_ADAPTER_MQTT_TARGET_PORT         | Upstream MQTT broker port                                               | 1883                     |
| MF_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK | URL of upstream broker health check endpoint                            |                          |
| MF_MQTT_ADAPTER_FORWARDER                | Enable multiprotocol message forwarder                                  | false                    |
| MF_MQTT_ADAPTER_FORWARDER_TIMEOUT        | Multiprotocol forwarder timeout                                         | 30s                      |
| MF_MQTT_ADAPTER_HTTP_PORT                | mProxy HTTP/WS listening port                                           | 8080                     |
| MF_MQTT_ADAPTER_WS_PORT                  | mProxy WebSocket listening port                                         | 8285                     |
| MF_MQTT_ADAPTER_WS_TARGET_HOST           | Upstream MQTT broker host for MQTT over WebSocket                       | localhost                |
| MF_MQTT_ADAPTER_WS_TARGET_PORT           | Upstream MQTT broker port for MQTT over WebSocket                       | 8080                     |
| MF_MQTT_ADAPTER_WS_TARGET_PATH           | Upstream MQTT broker WebSocket path                                     | /mqtt                    |
| MF_MQTT_ADAPTER_SERVER_CERT              | Path to server TLS certificate (PEM) — enables MQTTS on port 8883      |                          |
| MF_MQTT_ADAPTER_SERVER_KEY               | Path to server TLS key (PEM)                                            |                          |
| MF_MQTT_ADAPTER_CLIENT_TLS               | Enable TLS for outbound gRPC connections                                | false                    |
| MF_MQTT_ADAPTER_CA_CERTS                 | Path to CA certificates for gRPC TLS (PEM)                             |                          |
| MF_MQTT_ADAPTER_DB_HOST                  | Subscriptions database host address                                     | localhost                |
| MF_MQTT_ADAPTER_DB_PORT                  | Subscriptions database host port                                        | 5432                     |
| MF_MQTT_ADAPTER_DB_USER                  | Subscriptions database user                                             | mainflux                 |
| MF_MQTT_ADAPTER_DB_PASS                  | Subscriptions database password                                         | mainflux                 |
| MF_MQTT_ADAPTER_DB                       | Subscriptions database name                                             | subscriptions            |
| MF_MQTT_ADAPTER_DB_SSL_MODE              | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                  |
| MF_MQTT_ADAPTER_DB_SSL_CERT              | Path to the PEM encoded certificate file                                |                          |
| MF_MQTT_ADAPTER_DB_SSL_KEY               | Path to the PEM encoded key file                                        |                          |
| MF_MQTT_ADAPTER_DB_SSL_ROOT_CERT         | Path to the PEM encoded root certificate file                           |                          |
| MF_MQTT_ADAPTER_ES_URL                   | Event store URL                                                         | redis://localhost:6379/0 |
| MF_MQTT_ADAPTER_EVENT_CONSUMER           | Event store consumer name                                               | mqtt-adapter             |
| MF_AUTH_CACHE_URL                        | Auth cache URL                                                          | redis://localhost:6379/0 |
| MF_THINGS_AUTH_GRPC_URL                  | Things service Auth gRPC URL                                            | localhost:8183           |
| MF_THINGS_AUTH_GRPC_TIMEOUT              | Things service Auth gRPC request timeout                                | 1s                       |
| MF_AUTH_GRPC_URL                         | Auth service gRPC URL                                                   | localhost:8181           |
| MF_AUTH_GRPC_TIMEOUT                     | Auth service gRPC request timeout                                       | 1s                       |
| MF_BROKER_URL                            | Internal message broker URL                                             | nats://localhost:4222     |
| MF_JAEGER_URL                            | Jaeger server URL for distributed tracing. Leave empty to disable tracing. Docker value: `jaeger:6831` |                          |

## Deployment

The service itself is distributed as Docker container. Check the [`mqtt-adapter`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how the service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the mqtt adapter
make mqtt

# copy binary to bin
make install

# Set the environment variables and run the service
MF_MQTT_ADAPTER_LOG_LEVEL=[Log level] \
MF_MQTT_ADAPTER_MQTT_PORT=[MQTT listening port] \
MF_MQTT_ADAPTER_MQTT_TARGET_HOST=[Upstream MQTT broker host] \
MF_MQTT_ADAPTER_MQTT_TARGET_PORT=[Upstream MQTT broker port] \
MF_MQTT_ADAPTER_WS_PORT=[WebSocket listening port] \
MF_MQTT_ADAPTER_WS_TARGET_HOST=[Upstream WS broker host] \
MF_MQTT_ADAPTER_WS_TARGET_PORT=[Upstream WS broker port] \
MF_MQTT_ADAPTER_WS_TARGET_PATH=[Upstream WS path] \
MF_MQTT_ADAPTER_DB_HOST=[Subscriptions database host] \
MF_MQTT_ADAPTER_DB_PORT=[Subscriptions database port] \
MF_MQTT_ADAPTER_DB_USER=[Subscriptions database user] \
MF_MQTT_ADAPTER_DB_PASS=[Subscriptions database password] \
MF_MQTT_ADAPTER_DB=[Subscriptions database name] \
MF_BROKER_URL=[Internal message broker URL] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC timeout] \
MF_AUTH_GRPC_URL=[Auth service gRPC URL] \
MF_JAEGER_URL=[Jaeger service URL] \
MF_AUTH_CACHE_URL=[Auth cache URL] \
MF_MQTT_ADAPTER_ES_URL=[Event store URL] \
$GOBIN/mainfluxlabs-mqtt
```

## Usage

Connect any MQTT client to port `1883` (plain) or `8883` (TLS) using a Thing key as the password. Publish messages to `messages/<subtopic>` to send them through the platform.

For the full API reference, see the [AsyncAPI documentation](https://github.com/MainfluxLabs/mainflux/blob/master/api/asyncapi/mqtt.yml).

[doc]: https://mainfluxlabs.github.io/docs
