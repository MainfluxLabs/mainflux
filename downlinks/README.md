# Downlinks Service

The Downlinks service manages scheduled outbound HTTP requests for things and groups. Each downlink defines a target URL, HTTP method, optional payload and headers, a schedule, and an optional time filter for automatic time-range parameter injection. On startup, all persisted downlinks are loaded and scheduled; new downlinks are scheduled immediately upon creation.

## Downlinks

A downlink represents a single scheduled outbound HTTP request.

| Field         | Description                                                       |
|---------------|-------------------------------------------------------------------|
| `id`          | Unique downlink identifier (UUID)                                 |
| `group_id`    | ID of the group the downlink belongs to                           |
| `thing_id`    | ID of the thing the downlink is associated with                   |
| `name`        | Human-readable downlink name                                      |
| `url`         | Destination URL                                                   |
| `method`      | HTTP method: `GET`, `POST`, `PUT`, or `PATCH`                     |
| `payload`     | Optional request body (string)                                    |
| `headers`     | Optional HTTP headers sent with each request                      |
| `scheduler`   | Schedule configuration (see below)                                |
| `time_filter` | Optional time-range parameter injection configuration (see below) |
| `metadata`    | Arbitrary key-value pairs for custom attributes                   |

### Scheduler

The `scheduler` object controls when a downlink executes.

| Field       | Description                                                                                                                                                            |
|-------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `frequency` | Execution frequency: `once`, `minutely`, `hourly`, `daily`, or `weekly`                                                                                                |
| `time_zone` | IANA timezone name (e.g. `Europe/Berlin`, `UTC`)                                                                                                                       |
| `date_time` | Date and time in `YYYY-MM-DD HH:MM` format (e.g. `2026-03-25 14:30`). Required when `frequency` is `once`; ignored otherwise. Must be at least 1 minute in the future. |
| `minute`    | Minute interval (1–59). Used with `minutely` frequency                                                                                                                 |
| `hour`      | Hour interval (1–23). Used with `hourly` frequency                                                                                                                     |
| `day_time`  | Time of day in `HH:MM` format. Used with `daily` frequency                                                                                                             |
| `week`      | Weekly schedule: `{ days: [...], time: "HH:MM" }`. Days must be from `SUN`, `MON`, `TUE`, `WED`, `THU`, `FRI`, `SAT`. Used with `weekly` frequency                     |

### Time Filter

The `time_filter` object injects computed time-range parameters into the request URL at execution time.

| Field         | Description                                                                                                                                                                                                                                                                                                        |
|---------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `start_param` | Query parameter name for the start time (e.g. `from`)                                                                                                                                                                                                                                                              |
| `end_param`   | Query parameter name for the end time (e.g. `to`)                                                                                                                                                                                                                                                                  |
| `format`      | Time format for injected values (case-insensitive). Supported: `unix`, `unix_ms`, `unix_us`, `unix_ns`, `iso8601`, `compactiso8601`, `rfc3339`, `rfc3339nano`, `rfc822`, `rfc822z`, `rfc850`, `rfc1123`, `rfc1123z`, `ansic`, `unixdate`, `rubydate`, `stamp`, `stampmilli`, `stampmicro`, `stampnano`, `datetime` |
| `interval`    | Time range unit: `minute`, `hour`, or `day`                                                                                                                                                                                                                                                                        |
| `value`       | Number of interval units to include in the time range                                                                                                                                                                                                                                                              |
| `forecast`    | If `true`, uses a future time range instead of a past one                                                                                                                                                                                                                                                          |

**Example:** a downlink with `interval: hour`, `value: 1`, `forecast: false` will append `?from=<1 hour ago>&to=<now>` to the URL on each execution (formatted according to `format`).

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                        | Description                                                                | Default                  |
|---------------------------------|----------------------------------------------------------------------------|--------------------------|
| `MF_DOWNLINKS_LOG_LEVEL`        | Log level for the Downlinks service (debug, info, warn, error)             | error                    |
| `MF_BROKER_URL`                 | Message broker instance URL                                                | nats://localhost:4222    |
| `MF_DOWNLINKS_HTTP_PORT`        | Downlinks service HTTP port                                                | 9025                     |
| `MF_JAEGER_URL`                 | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                          |
| `MF_DOWNLINKS_DB_HOST`          | Database host address                                                      | localhost                |
| `MF_DOWNLINKS_DB_PORT`          | Database host port                                                         | 5432                     |
| `MF_DOWNLINKS_DB_USER`          | Database user                                                              | mainflux                 |
| `MF_DOWNLINKS_DB_PASS`          | Database password                                                          | mainflux                 |
| `MF_DOWNLINKS_DB`               | Name of the database used by the service                                   | downlinks                |
| `MF_DOWNLINKS_DB_SSL_MODE`      | Database connection SSL mode (disable, require, verify-ca, verify-full)    | disable                  |
| `MF_DOWNLINKS_DB_SSL_CERT`      | Path to the PEM encoded certificate file                                   |                          |
| `MF_DOWNLINKS_DB_SSL_KEY`       | Path to the PEM encoded key file                                           |                          |
| `MF_DOWNLINKS_DB_SSL_ROOT_CERT` | Path to the PEM encoded root certificate file                              |                          |
| `MF_DOWNLINKS_CLIENT_TLS`       | Flag that indicates if TLS should be turned on                             | false                    |
| `MF_DOWNLINKS_CA_CERTS`         | Path to trusted CAs in PEM format                                          |                          |
| `MF_DOWNLINKS_SERVER_CERT`      | Path to server certificate in PEM format                                   |                          |
| `MF_DOWNLINKS_SERVER_KEY`       | Path to server key in PEM format                                           |                          |
| `MF_THINGS_AUTH_GRPC_URL`       | Things service Auth gRPC URL                                               | localhost:8183           |
| `MF_THINGS_AUTH_GRPC_TIMEOUT`   | Things service Auth gRPC request timeout in seconds                        | 1s                       |
| `MF_AUTH_GRPC_URL`              | Auth service gRPC URL                                                      | localhost:8181           |
| `MF_AUTH_GRPC_TIMEOUT`          | Auth service gRPC request timeout in seconds                               | 1s                       |
| `MF_DOWNLINKS_ES_URL`           | Event store URL                                                            | redis://localhost:6379/0 |
| `MF_DOWNLINKS_EVENT_CONSUMER`   | Event store consumer name                                                  | downlinks                |

## Deployment

The service itself is distributed as Docker container. Check the [`downlinks`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# Download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the downlinks service
make downlinks

# Copy binary to bin
make install

# Set the environment variables and run the service
MF_DOWNLINKS_LOG_LEVEL=[Downlinks log level] \
MF_BROKER_URL=[Message broker instance URL] \
MF_DOWNLINKS_HTTP_PORT=[Downlinks service HTTP port] \
MF_DOWNLINKS_DB_HOST=[Database host address] \
MF_DOWNLINKS_DB_PORT=[Database host port] \
MF_DOWNLINKS_DB_USER=[Database user] \
MF_DOWNLINKS_DB_PASS=[Database password] \
MF_DOWNLINKS_DB=[Downlinks database name] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout] \
MF_AUTH_GRPC_URL=[Auth service gRPC URL] \
MF_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout] \
$GOBIN/mainfluxlabs-downlinks
```

## Usage

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).
