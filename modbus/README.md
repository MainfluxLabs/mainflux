# Modbus Service

The Modbus service manages Modbus TCP polling clients for things and groups. Each client connects to a Modbus TCP device on a schedule, reads the configured registers or coils using the specified function code, and publishes the result as a JSON message to the platform on behalf of the associated thing.

## Modbus Clients

A Modbus client defines the connection parameters, poll schedule, and data fields to read from a Modbus TCP device.

| Field           | Description                                                  |
|-----------------|--------------------------------------------------------------|
| `id`            | Unique client identifier (UUID)                              |
| `group_id`      | ID of the group the client belongs to                        |
| `thing_id`      | ID of the thing the client is associated with                |
| `name`          | Human-readable client name                                   |
| `ip_address`    | IP address of the Modbus TCP device                          |
| `port`          | TCP port of the Modbus device (default Modbus port is `502`) |
| `slave_id`      | Modbus slave/unit ID (0–255)                                 |
| `function_code` | Modbus read function code (see below)                        |
| `scheduler`     | Poll schedule configuration (see below)                      |
| `data_fields`   | List of registers or coils to read (see below)               |
| `metadata`      | Arbitrary key-value pairs for custom attributes              |

### Function Codes

| Value                  | Modbus Code | Description                                |
|------------------------|-------------|--------------------------------------------|
| `ReadCoils`            | 0x01        | Read output coils (digital output)         |
| `ReadDiscreteInputs`   | 0x02        | Read discrete inputs (digital input)       |
| `ReadHoldingRegisters` | 0x03        | Read holding registers (read/write analog) |
| `ReadInputRegisters`   | 0x04        | Read input registers (read-only analog)    |

### Scheduler

The `scheduler` object controls when the client polls the device.

| Field       | Description                                                                                                                                                            |
|-------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `frequency` | Poll frequency: `once`, `minutely`, `hourly`, `daily`, or `weekly`                                                                                                     |
| `time_zone` | IANA timezone name (e.g. `Europe/Berlin`, `UTC`)                                                                                                                       |
| `date_time` | Date and time in `YYYY-MM-DD HH:MM` format (e.g. `2026-03-25 14:30`). Required when `frequency` is `once`; ignored otherwise. Must be at least 1 minute in the future. |
| `minute`    | Minute interval (1–59). Used with `minutely` frequency                                                                                                                 |
| `hour`      | Hour interval (1–23). Used with `hourly` frequency                                                                                                                     |
| `day_time`  | Time of day in `HH:MM` format. Used with `daily` frequency                                                                                                             |
| `week`      | Weekly schedule: `{ days: [...], time: "HH:MM" }`. Days must be from `SUN`, `MON`, `TUE`, `WED`, `THU`, `FRI`, `SAT`. Used with `weekly` frequency                     |

### Data Fields

Each entry in `data_fields` describes a single register or coil to read.

| Field        | Description                                                                                                           |
|--------------|-----------------------------------------------------------------------------------------------------------------------|
| `name`       | Field name used as the JSON key in the published message                                                              |
| `type`       | Data type: `bool`, `int16`, `uint16`, `int32`, `uint32`, `float32`, or `string`                                       |
| `unit`       | Optional unit label (e.g. `°C`, `%`)                                                                                  |
| `scale`      | Optional multiplier applied to the raw numeric value before publishing                                                |
| `byte_order` | Multi-byte word order: `ABCD` (big-endian), `DCBA` (little-endian), `CDAB` (PDP/middle-endian), `BADC` (byte-swapped) |
| `address`    | Starting register or coil address                                                                                     |
| `length`     | Register count. For numeric types this is calculated automatically from `type`; set manually for `string` fields only |

The poll result is published as a JSON object keyed by field `name`, for example:

```json
{"temperature": 23.5, "humidity": 61}
```

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                      | Description                                                                | Default                  |
|-------------------------------|----------------------------------------------------------------------------|--------------------------|
| `MF_MODBUS_LOG_LEVEL`         | Log level for the Modbus service (debug, info, warn, error)                | error                    |
| `MF_BROKER_URL`               | Message broker instance URL                                                | nats://localhost:4222    |
| `MF_MODBUS_HTTP_PORT`         | Modbus service HTTP port                                                   | 9028                     |
| `MF_JAEGER_URL`               | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                          |
| `MF_MODBUS_DB_HOST`           | Database host address                                                      | localhost                |
| `MF_MODBUS_DB_PORT`           | Database host port                                                         | 5432                     |
| `MF_MODBUS_DB_USER`           | Database user                                                              | mainflux                 |
| `MF_MODBUS_DB_PASS`           | Database password                                                          | mainflux                 |
| `MF_MODBUS_DB`                | Name of the database used by the service                                   | modbus                   |
| `MF_MODBUS_DB_SSL_MODE`       | Database connection SSL mode (disable, require, verify-ca, verify-full)    | disable                  |
| `MF_MODBUS_DB_SSL_CERT`       | Path to the PEM encoded certificate file                                   |                          |
| `MF_MODBUS_DB_SSL_KEY`        | Path to the PEM encoded key file                                           |                          |
| `MF_MODBUS_DB_SSL_ROOT_CERT`  | Path to the PEM encoded root certificate file                              |                          |
| `MF_MODBUS_CLIENT_TLS`        | Flag that indicates if TLS should be turned on                             | false                    |
| `MF_MODBUS_CA_CERTS`          | Path to trusted CAs in PEM format                                          |                          |
| `MF_MODBUS_SERVER_CERT`       | Path to server certificate in PEM format                                   |                          |
| `MF_MODBUS_SERVER_KEY`        | Path to server key in PEM format                                           |                          |
| `MF_THINGS_AUTH_GRPC_URL`     | Things service Auth gRPC URL                                               | localhost:8183           |
| `MF_THINGS_AUTH_GRPC_TIMEOUT` | Things service Auth gRPC request timeout in seconds                        | 1s                       |
| `MF_MODBUS_ES_URL`            | Event store URL                                                            | redis://localhost:6379/0 |
| `MF_MODBUS_EVENT_CONSUMER`    | Event store consumer name                                                  | modbus                   |

## Deployment

The service itself is distributed as Docker container. Check the [`modbus`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# Download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the modbus service
make modbus

# Copy binary to bin
make install

# Set the environment variables and run the service
MF_MODBUS_LOG_LEVEL=[Modbus log level] \
MF_BROKER_URL=[Message broker instance URL] \
MF_MODBUS_HTTP_PORT=[Modbus service HTTP port] \
MF_MODBUS_DB_HOST=[Database host address] \
MF_MODBUS_DB_PORT=[Database host port] \
MF_MODBUS_DB_USER=[Database user] \
MF_MODBUS_DB_PASS=[Database password] \
MF_MODBUS_DB=[Modbus database name] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout] \
$GOBIN/mainfluxlabs-modbus
```

## Usage

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).
