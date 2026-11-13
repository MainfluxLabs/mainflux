# Mainflux CoAP Adapter

Mainflux CoAP adapter provides an [CoAP](http://coap.technology/) API for sending messages through the
platform.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                            | Description                                                                | Default               |
| ----------------------------------- | -------------------------------------------------------------------------- | --------------------- |
| `MF_COAP_ADAPTER_PORT`              | Service listening port                                                     | 5683                  |
| `MF_COAP_ADAPTER_PLAIN_ENABLED`     | Flag that enables the plain CoAP listener on `MF_COAP_ADAPTER_PORT`        | true                  |
| `MF_COAP_ADAPTER_DTLS_PORT`         | DTLS (`coaps`) listening port                                              | 5684                  |
| `MF_COAP_ADAPTER_DTLS_IDLE_TIMEOUT` | Time after which an idle DTLS session is closed                            | 5m                    |
| `MF_BROKER_URL`                     | Message broker instance URL                                                | nats://localhost:4222 |
| `MF_COAP_ADAPTER_LOG_LEVEL`         | Service log level                                                          | error                 |
| `MF_COAP_ADAPTER_CLIENT_TLS`        | Flag that indicates if TLS should be turned on                             | false                 |
| `MF_COAP_ADAPTER_CA_CERTS`          | Path to trusted CAs in PEM format                                          |                       |
| `MF_JAEGER_URL`                     | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                       |
| `MF_THINGS_AUTH_GRPC_URL`           | Things service Auth gRPC URL                                               | localhost:8183        |
| `MF_THINGS_AUTH_GRPC_TIMEOUT`       | Things service Auth gRPC request timeout in seconds                        | 1s                    |

## Deployment

The service itself is distributed as Docker container. Check the [`coap-adapter`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

Running this service outside of container requires working instance of the message broker service.
To start the service outside of the container, execute the following shell script:

```bash
# Download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the http
make coap

# Copy binary to bin
make install

# set the environment variables and run the service
MF_BROKER_URL=[Message broker instance URL] \
MF_COAP_ADAPTER_PORT=[Service HTTP port] \
MF_COAP_ADAPTER_PLAIN_ENABLED=[Flag that enables the plain CoAP listener] \
MF_COAP_ADAPTER_DTLS_PORT=[DTLS listening port] \
MF_COAP_ADAPTER_DTLS_IDLE_TIMEOUT=[Time after which an idle DTLS session is closed] \
MF_COAP_ADAPTER_LOG_LEVEL=[Service log level] \
MF_COAP_ADAPTER_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MF_COAP_ADAPTER_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
$GOBIN/mainfluxlabs-coap
```

## Usage

### DTLS

Over DTLS (`coaps://<host>:5684`) a thing authenticates in the handshake with a pre-shared key:

- PSK identity: thing ID
- PSK: the first 32 hex characters of the SHA-256 hash of the thing key

The PSK is derived from the key because constrained DTLS stacks (mbedTLS, tinydtls, nRF91 modems) accept
PSKs of at most 32 bytes, and the key is longer. Compute it with:

```bash
echo -n <thing_key> | sha256sum | cut -c1-32
```

A wrong key or an unknown thing ID fails the handshake. Requests over the session need no `auth` and
`type` query parameters, and the key is never sent over the network. The adapter supports the
`TLS_PSK_WITH_AES_128_CCM_8` and `TLS_PSK_WITH_AES_128_GCM_SHA256` cipher suites.

The adapter supports DTLS Connection IDs (RFC 9146). A client that negotiates one keeps its session when
NAT changes its address or port, instead of repeating the handshake.

A session with no traffic from the thing for `MF_COAP_ADAPTER_DTLS_IDLE_TIMEOUT` is closed, together with
its observations. A thing that only observes commands should send something more often, for example a
message or a CoAP ping, which also keeps carrier NAT mappings alive.

```bash
PSK=$(echo -n <thing_key> | sha256sum | cut -c1-32)
coap-client-openssl -m post -e '{"v":1}' -u <thing_id> -k "$PSK" "coaps://localhost/things/<thing_id>/messages"
```

### Plain CoAP

Plain CoAP sends everything in cleartext, including the thing key, so it should be disabled with
`MF_COAP_ADAPTER_PLAIN_ENABLED=false` once things use DTLS.

Since CoAP protocol does not support `Authorization` header (option) and since options have a limited size, authentication is supplied in the form of URI Query parameters:

- `auth`
  - Thing authentication key
- `type`
  - Type of Thing key. One of: `external`, `internal`.

### Publishing messages

To publish a message to a thing, optionally with a subtopic:

```
coap://localhost/things/<thing_id>/messages?auth=<thing_key>&type=<key_type>
coap://localhost/things/<thing_id>/messages/<subtopic>?auth=<thing_key>&type=<key_type>
```

The thing is determined by the auth key, not the path.

### Sending commands

To send a command to a specific thing (M2M, authorized by thing key):

```
coap://localhost/things/<thing_id>/commands?auth=<thing_key>&type=<key_type>
```

To send a command to all things in a group:

```
coap://localhost/groups/<group_id>/commands?auth=<thing_key>&type=<key_type>
```

Both endpoints support an optional subtopic:

```
coap://localhost/things/<thing_id>/commands/<subtopic>?auth=<thing_key>&type=<key_type>
coap://localhost/groups/<group_id>/commands/<subtopic>?auth=<thing_key>&type=<key_type>
```

Commands use `POST` (same as publishing messages). The thing must have command permission over the target thing or group.
