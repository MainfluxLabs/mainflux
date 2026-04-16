# HTTP adapter

HTTP adapter provides an HTTP API for sending messages and commands through the platform.

## Authentication

The `Authorization` header identifies the Thing making the request. Three formats are supported:

| Format       | Header value                    | Notes                          |
|--------------|---------------------------------|--------------------------------|
| Internal key | `Authorization: Thing <key>`    | Standard Thing key             |
| External key | `Authorization: External <key>` | External/third-party Thing key |

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                      | Description                                                                | Default               |
|-------------------------------|----------------------------------------------------------------------------|-----------------------|
| `MF_HTTP_ADAPTER_LOG_LEVEL`   | Log level (debug, info, warn, error)                                       | error                 |
| `MF_HTTP_ADAPTER_PORT`        | Service HTTP port                                                          | 8180                  |
| `MF_HTTP_ADAPTER_CLIENT_TLS`  | Flag that indicates if TLS should be turned on for gRPC                    | false                 |
| `MF_HTTP_ADAPTER_CA_CERTS`    | Path to trusted CAs in PEM format for gRPC TLS                             |                       |
| `MF_BROKER_URL`               | Message broker instance URL                                                | nats://localhost:4222 |
| `MF_THINGS_AUTH_GRPC_URL`     | Things service Auth gRPC URL                                               | localhost:8183        |
| `MF_THINGS_AUTH_GRPC_TIMEOUT` | Things service Auth gRPC request timeout in seconds                        | 1s                    |
| `MF_JAEGER_URL`               | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                       |

## Deployment

The service itself is distributed as Docker container. Check the [`http-adapter`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how the service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# Download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the http adapter
make http

# Copy binary to bin
make install

# Set the environment variables and run the service
MF_HTTP_ADAPTER_LOG_LEVEL=[Log level] \
MF_HTTP_ADAPTER_PORT=[Service HTTP port] \
MF_HTTP_ADAPTER_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_BROKER_URL=[Message broker instance URL] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout] \
MF_JAEGER_URL=[Jaeger server URL] \
$GOBIN/mainfluxlabs-http
```

## Usage

For the full API reference, see the [API documentation](https://mainfluxlabs.github.io/docs/swagger/).

