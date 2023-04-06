# InfluxDB writer

InfluxDB writer provides message repository implementation for InfluxDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                      | Description                                                                       | Default                |
| ----------------------------- | --------------------------------------------------------------------------------- | ---------------------- |
| MF_BROKER_URL                 | Message broker instance URL                                                       | nats://localhost:4222  |
| MF_INFLUX_WRITER_LOG_LEVEL    | Log level for InfluxDB writer (debug, info, warn, error)                          | error                  |
| MF_INFLUX_WRITER_PORT         | Service HTTP port                                                                 | 8180                   |
| MF_INFLUX_WRITER_DB_HOST      | InfluxDB host                                                                     | localhost              |
| MF_INFLUXDB_PORT              | Default port of InfluxDB database                                                 | 8086                   |
| MF_INFLUXDB_ADMIN_USER        | Default user of InfluxDB database                                                 | mainflux               |
| MF_INFLUXDB_ADMIN_PASSWORD    | Default password of InfluxDB user                                                 | mainflux               |
| MF_INFLUXDB_DB                | InfluxDB database name                                                            | mainflux               |
| MF_INFLUX_WRITER_CONFIG_PATH  | Config file path with message broker subjects list, payload type and content-type | /configs.toml          |

## Deployment

The service itself is distributed as Docker container. Check the [`influxdb-writer`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/addons/influxdb-writer/docker-compose.yml#L35-L58) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the influxdb
make influxdb

# copy binary to bin
make install

# Set the environment variables and run the service
MF_BROKER_URL=[Message broker instance URL] \
MF_INFLUX_WRITER_LOG_LEVEL=[Influx writer log level] \
MF_INFLUX_WRITER_PORT=[Service HTTP port] \
MF_INFLUXDB_DB=[InfluxDB database name] \
MF_INFLUXDB_HOST=[InfluxDB database host] \
MF_INFLUXDB_PORT=[InfluxDB database port] \
MF_INFLUXDB_ADMIN_USER=[InfluxDB admin user] \
MF_INFLUXDB_ADMIN_PASSWORD=[InfluxDB admin password] \
MF_INFLUX_WRITER_CONFIG_PATH=[Config file path with Message broker subjects list, payload type and content-type] \
$GOBIN/mainfluxlabs-influxdb
```

### Using docker-compose

This service can be deployed using docker containers.
Docker compose file is available in `<project_root>/docker/addons/influxdb-writer/docker-compose.yml`. Besides database and writer service, it contains [InfluxDB UI](https://docs.influxdata.com/influxdb/v2.6/visualize-data/) which can be used for database exploration and data visualization and analytics. In order to run Mainflux InfluxDB writer, execute the following command:

```bash
docker-compose -f docker/addons/influxdb-writer/docker-compose.yml up -d
```

_Please note that you need to start core services before the additional ones._

## Usage

Starting service will start consuming normalized messages in SenML format.

[doc]: https://mainfluxlabs.github.io/docs
