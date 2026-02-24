# Webhooks

Webhooks service provides forwarding received messages to other platforms.

## Configuration

The service is configured using the environment variables from the following table. Note that any unset variables will be replaced with their default values.

| Variable                     | Description                                                             | Default               |
|------------------------------|-------------------------------------------------------------------------|-----------------------|
| MF_WEBHOOKS_LOG_LEVEL        | Log level for Webhooks (debug, info, warn, error)                       | error                 |
| MF_WEBHOOKS_DB_HOST          | Database host address                                                   | localhost             |
| MF_WEBHOOKS_DB_PORT          | Database host port                                                      | 5432                  |
| MF_WEBHOOKS_DB_USER          | Database user                                                           | mainflux              |
| MF_WEBHOOKS_DB_PASS          | Database password                                                       | mainflux              |
| MF_WEBHOOKS_DB               | Name of the database used by the service                                | webhooks              |
| MF_WEBHOOKS_DB_SSL_MODE      | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable               |
| MF_WEBHOOKS_DB_SSL_CERT      | Path to the PEM encoded certificate file                                |                       |
| MF_WEBHOOKS_DB_SSL_KEY       | Path to the PEM encoded key file                                        |                       |
| MF_WEBHOOKS_DB_SSL_ROOT_CERT | Path to the PEM encoded root certificate file                           |                       |
| MF_WEBHOOKS_CLIENT_TLS       | Flag that indicates if TLS should be turned on                          | false                 |
| MF_WEBHOOKS_CA_CERTS         | Path to trusted CAs in PEM format                                       |                       |
| MF_WEBHOOKS_HTTP_PORT        | Webhooks service HTTP port                                              | 9021                  |
| MF_WEBHOOKS_SERVER_CERT      | Path to server certificate in pem format                                |                       |
| MF_WEBHOOKS_SERVER_KEY       | Path to server key in pem format                                        |                       |
| MF_JAEGER_URL                | Jaeger server URL                                                       | localhost:6831        |
| MF_BROKER_URL                | Message broker URL                                                      | nats://127.0.0.1:4222 |
| MF_THINGS_AUTH_GRPC_URL      | Things auth service gRPC URL                                            | localhost:8183        |
| MF_THINGS_AUTH_GRPC_TIMEOUT  | Things auth service gRPC request timeout in seconds                     | 1s                    |

## Deployment

The service is distributed as a Docker container. Check the [`webhooks `](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml#L500-L523) service section in
docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/MainfluxLabs/mainflux

cd $GOPATH/src/github.com/MainfluxLabs/mainflux

# compile the webhooks
make webhooks

# copy binary to bin
make install

# set the environment variables and run the service
MF_WEBHOOKS_LOG_LEVEL=[Webhooks log level]
MF_WEBHOOKS_DB_HOST=[Database host address]
MF_WEBHOOKS_DB_PORT=[Database host port]
MF_WEBHOOKS_DB_USER=[Database user]
MF_WEBHOOKS_DB_PASS=[Database password]
MF_WEBHOOKS_DB=[Name of the database used by the service]
MF_WEBHOOKS_DB_SSL_MODE=[SSL mode to connect to the database with]
MF_WEBHOOKS_DB_SSL_CERT=[Path to the PEM encoded certificate file]
MF_WEBHOOKS_DB_SSL_KEY=[Path to the PEM encoded key file]
MF_WEBHOOKS_DB_SSL_ROOT_CERT=[Path to the PEM encoded root certificate file]
MF_WEBHOOKS_CLIENT_TLS=[Flag that indicates if TLS should be turned on]                                           
MF_WEBHOOKS_CA_CERTS=[Path to trusted CAs in PEM format]                            
MF_WEBHOOKS_HTTP_PORT=[Service HTTP port]
MF_WEBHOOKS_SERVER_CERT=[String path to server cert in pem format]
MF_WEBHOOKS_SERVER_KEY=[String path to server key in pem format]
MF_JAEGER_URL=[Jaeger server URL]
MF_BROKER_URL=[Message broker URL]
MF_THINGS_AUTH_GRPC_URL=[Things auth service gRPC URL]
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things auth service gRPC request timeout in seconds]
$GOBIN/mainflux-kit
```

## Usage

For more information about service capabilities and its usage, please check out the [API documentation](https://github.com/MainfluxLabs/mainflux/blob/master/api/openapi/webhooks.yml).

[doc]: http://mainflux.readthedocs.io
