# Readers

Readers provide an implementation of various `message readers`.
Message readers are services that consume normalized (in `SenML` format)
Mainflux messages from data storage and opens HTTP API for message consumption.

For an in-depth explanation of the usage of `reader`, as well as thorough
understanding of Mainflux, please check out the [official documentation][doc].

[doc]: https://mainfluxlabs.github.io/docs

## gRPC API

In addition to the HTTP API, the postgres-reader exposes a gRPC API that allows
other internal services to query messages by thing key.

### Environment Variables (postgres-reader)

| Variable | Description | Default |
|----------|-------------|---------|
| `MF_POSTGRES_READER_GRPC_PORT` | gRPC server port | `8184` |
| `MF_POSTGRES_READER_GRPC_SERVER_CERT` | Path to gRPC server TLS certificate (optional) | `""` |
| `MF_POSTGRES_READER_GRPC_SERVER_KEY` | Path to gRPC server TLS key (optional) | `""` |

### Methods

- `ListJSONMessages` — returns a page of JSON messages filtered by thing key and page metadata
- `ListSenMLMessages` — returns a page of SenML messages filtered by thing key and page metadata

Authentication is done via thing key (no user token required).
