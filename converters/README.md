# Converters Service

The Converters service accepts CSV file uploads and publishes the parsed rows as SenML or JSON messages to the platform on behalf of a thing. It is designed for bulk historical data ingestion — common in migration scenarios or periodic batch uploads from data-logging devices.

The service authenticates the request using the thing's internal key and publishes messages to NATS using the same subject scheme as live messages, so the data flows through the normal consumer pipeline.

Large files are processed in batches of 50,000 records. A 30-second pause is inserted between batches to avoid overwhelming downstream consumers.

## CSV Formats

### SenML (`POST /csv/senml`)

| Row       | First column                                    | Remaining columns                         |
|-----------|-------------------------------------------------|-------------------------------------------|
| Header    | Ignored (placeholder for the timestamp column)  | Measurement names (string)                |
| Data rows | Unix timestamp as a floating-point number       | Floating-point measurement values         |

Each data row produces one SenML record per measurement column with fields `n` (name), `v` (value), and `t` (timestamp).

### JSON (`POST /csv/json`)

| Row       | Content                                                                                      |
|-----------|----------------------------------------------------------------------------------------------|
| Header    | Column names. Must include a `created` column; other columns become arbitrary JSON fields.   |
| Data rows | Values are parsed as floating-point numbers where possible; otherwise kept as strings.       |

Each data row produces one JSON object keyed by column name.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                     | Description                                                     | Default               |
|------------------------------|-----------------------------------------------------------------|-----------------------|
| MF_CONVERTERS_LOG_LEVEL      | Log level for the Converters service (debug, info, warn, error) | error                 |
| MF_CONVERTERS_PORT           | Converters service HTTP port                                    | 8180                  |
| MF_JAEGER_URL                | Jaeger server URL                                               |                       |
| MF_BROKER_URL                | Message broker instance URL                                     | nats://localhost:4222 |
| MF_CONVERTERS_CLIENT_TLS     | Flag that indicates if TLS should be turned on                  | false                 |
| MF_CONVERTERS_CA_CERTS       | Path to trusted CAs in PEM format                               |                       |
| MF_THINGS_AUTH_GRPC_URL      | Things service Auth gRPC URL                                    | localhost:8183        |
| MF_THINGS_AUTH_GRPC_TIMEOUT  | Things service Auth gRPC request timeout in seconds             | 1s                    |

## Deployment

The service itself is distributed as Docker container. Check the [`converters`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the converters service
make converters

# copy binary to bin
make install

# Set the environment variables and run the service
MF_CONVERTERS_LOG_LEVEL=[Converters log level] \
MF_CONVERTERS_PORT=[Converters service HTTP port] \
MF_BROKER_URL=[Message broker instance URL] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout] \
$GOBIN/mainfluxlabs-converters
```

## Usage

Requests must be authenticated with the thing's internal key using the `Authorization: Thing <key>` header.

```bash
# Convert and publish a CSV file as SenML messages
curl -X POST http://localhost:8180/csv/senml \
  -H "Authorization: Thing <thing_key>" \
  -F "file=@/path/to/data.csv"

# Convert and publish a CSV file as JSON messages
curl -X POST http://localhost:8180/csv/json \
  -H "Authorization: Thing <thing_key>" \
  -F "file=@/path/to/data.csv"
```

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).
