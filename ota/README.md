# OTA Service

The OTA service rolls out firmware to things over the air. A user creates an **update** for a set of
things in a group, the service notifies those things, and each thing reports its progress until it
runs the new firmware or fails. Firmware files are stored by the filestore service as group files.

## Updates

An update describes the firmware and who should install it.

| Field         | Description                                                                 |
| ------------- | --------------------------------------------------------------------------- |
| `id`          | Update ID (UUID)                                                            |
| `group_id`    | Group of the things being updated                                           |
| `file_name`   | Name of the firmware group file in the filestore service                    |
| `version`     | Version things must report on success                                       |
| `file_size`   | Firmware size in bytes                                                      |
| `checksum`    | Hex-encoded SHA-256 of the firmware, for the thing to verify the download   |
| `description` | Free text                                                                   |
| `timeout`     | Seconds without progress after which a thing is marked failed (default 24h) |
| `stats`       | Number of the update's things by status                                     |

Each thing of an update has its own progress: `status`, `error`, `reported_version` and `updated_at`.

### Statuses

```
pending ─► downloading ─► installing ─► succeeded
   │            │              │
   └────────────┴──────────────┴──────► failed
```

| Status        | Meaning                                                                          |
| ------------- | -------------------------------------------------------------------------------- |
| `pending`     | The thing has not started the update yet                                         |
| `downloading` | The thing is downloading the firmware                                            |
| `installing`  | The thing is installing the firmware                                             |
| `succeeded`   | The thing reported the update's version                                          |
| `failed`      | The thing reported an error or another version, timed out, or was superseded     |

## How it works

1. **Create.** A user creates an update for things of a group (`POST /groups/{id}/updates`). Every
   thing starts as `pending`. A thing runs at most one update at a time: its older running update is
   marked `failed` with `superseded by update <id>`.
2. **Notify.** The service publishes the `ota` command on each thing's command subject
   (`things.<id>.commands.ota`). Online things receive it over MQTT or CoAP. Things that were offline
   fetch it on reconnect with `GET /ota/pending`. Both carry the same payload:

   ```json
   {
     "update_id": "…",
     "version": "1.4.0",
     "file_name": "fw-sensor-1.4.0.bin",
     "url": "https://<host>/fs/groupfiles/fw-sensor-1.4.0.bin",
     "size": 421888,
     "checksum": "9f2c…"
   }
   ```

3. **Download.** The thing downloads the firmware from `url` with its thing key, resuming with HTTP
   Range requests if needed, and verifies the SHA-256 checksum.
4. **Report.** The thing reports `{"update_id", "status", "version", "error"}` either with
   `POST /ota/status` or by publishing on the `ota` subtopic of its messages over any protocol
   (e.g. MQTT topic `things/<id>/messages/ota`). Reports on the `ota` subtopic always reach the
   service, regardless of the profile's flags. A `succeeded` report must include `version`; without
   it the report is rejected and the status is unchanged. If `version` differs from the update's
   version, the thing is marked `failed`.
5. **Timeout.** Things without progress for the update's `timeout` are marked `failed` with
   `timed out`. The service checks this every `MF_OTA_TIMEOUT_CHECK_INTERVAL`.
6. **Retry.** A user can move the failed things of an update back to `pending`
   (`POST /updates/{id}/retry`); they are notified again. The service never retries on its own.

Authorization is delegated to the Things service: viewing updates requires `viewer` access on the
group, while creating, retrying or removing updates requires `editor` access. Things authenticate
with their thing key. Updates and progress are removed when their group or thing is removed.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                        | Description                                                                | Default                         |
| ------------------------------- | -------------------------------------------------------------------------- | ------------------------------- |
| `MF_OTA_LOG_LEVEL`              | Log level for the OTA service (debug, info, warn, error)                   | error                           |
| `MF_BROKER_URL`                 | Message broker instance URL                                                | nats://localhost:4222           |
| `MF_OTA_HTTP_PORT`              | OTA service HTTP port                                                      | 9032                            |
| `MF_JAEGER_URL`                 | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                                 |
| `MF_OTA_DB_HOST`                | Database host address                                                      | localhost                       |
| `MF_OTA_DB_PORT`                | Database host port                                                         | 5432                            |
| `MF_OTA_DB_USER`                | Database user                                                              | mainflux                        |
| `MF_OTA_DB_PASS`                | Database password                                                          | mainflux                        |
| `MF_OTA_DB`                     | Name of the database used by the service                                   | ota                             |
| `MF_OTA_DB_SSL_MODE`            | Database connection SSL mode (disable, require, verify-ca, verify-full)    | disable                         |
| `MF_OTA_DB_SSL_CERT`            | Path to the PEM encoded certificate file                                   |                                 |
| `MF_OTA_DB_SSL_KEY`             | Path to the PEM encoded key file                                           |                                 |
| `MF_OTA_DB_SSL_ROOT_CERT`       | Path to the PEM encoded root certificate file                              |                                 |
| `MF_OTA_CLIENT_TLS`             | Flag that indicates if TLS should be turned on                             | false                           |
| `MF_OTA_CA_CERTS`               | Path to trusted CAs in PEM format                                          |                                 |
| `MF_OTA_SERVER_CERT`            | Path to server certificate in PEM format                                   |                                 |
| `MF_OTA_SERVER_KEY`             | Path to server key in PEM format                                           |                                 |
| `MF_THINGS_AUTH_GRPC_URL`       | Things service Auth gRPC URL                                               | localhost:8183                  |
| `MF_THINGS_AUTH_GRPC_TIMEOUT`   | Things service Auth gRPC request timeout in seconds                        | 1s                              |
| `MF_AUTH_GRPC_URL`              | Auth service gRPC URL                                                      | localhost:8181                  |
| `MF_AUTH_GRPC_TIMEOUT`          | Auth service gRPC request timeout in seconds                               | 1s                              |
| `MF_OTA_ES_URL`                 | Event store URL                                                            | redis://localhost:6379/0        |
| `MF_OTA_FIRMWARE_URL`           | Public base URL things download firmware group files from                  | https://localhost/fs/groupfiles |
| `MF_OTA_TIMEOUT_CHECK_INTERVAL` | How often things without progress are checked for timeout                  | 10m                             |

## Deployment

The service itself is distributed as Docker container. Check the [`ota`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# Download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the ota service
make ota

# Copy binary to bin
make install

# Set the environment variables and run the service
MF_OTA_LOG_LEVEL=[OTA log level] \
MF_BROKER_URL=[Message broker instance URL] \
MF_OTA_HTTP_PORT=[OTA service HTTP port] \
MF_OTA_DB_HOST=[Database host address] \
MF_OTA_DB_PORT=[Database host port] \
MF_OTA_DB_USER=[Database user] \
MF_OTA_DB_PASS=[Database password] \
MF_OTA_DB=[OTA database name] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout] \
MF_AUTH_GRPC_URL=[Auth service gRPC URL] \
MF_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout] \
MF_OTA_FIRMWARE_URL=[Public base URL of firmware group files] \
$GOBIN/mainfluxlabs-ota
```

## Usage

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).
