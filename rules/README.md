# Rules

Rules service provides an HTTP API for managing [Kuiper](https://github.com/emqx/kuiper) rules engine entities. Use Rules to perform CRUD operations on streams - Kuiper entities defining message stream going from Mainflux into the Kuiper rules engine - and rules - Kuiper entities defining filtering and transforming operations on the message stream going from the Kuiper rules engine into the Mainflux.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                    | Description                                                          | Default               |
|-----------------------------|----------------------------------------------------------------------|-----------------------|
| MF_RULES_LOG_LEVEL          | Log level for Things (debug, info, warn, error)                      | error                 |
| MF_RULES_HTTP_PORT          | Rules service HTTP port                                              | 9099                  |
| MF_RULES_SERVER_CERT        | Path to trusted CAs in PEM format                                    |                       |
| MF_RULES_SERVER_KEY         | Path to server key in pem format                                     |                       |
| MF_RULES_SINGLE_USER_EMAIL  | User email for single user mode (no gRPC communication with users)   |                       |
| MF_RULES_SINGLE_USER_TOKEN  | User token for single user mode that should be passed in auth header |                       |
| MF_RULES_CLIENT_TLS         | Flag that indicates if TLS should be turned on                       | false                 |
| MF_RULES_CA_CERTS           | Path to trusted CAs in PEM format                                    |                       |
| MF_KUIPER_URL               | Kuiper rules engine url                                              | http://localhost:9081 |
| MF_JAEGER_URL               | Jaeger server url                                                    |                       |
| MF_AUTH_GRPC_URL            | Auth service gRPC url                                                | localhost:8181        |
| MF_AUTH_GRPC_TIMEOUT        | Auth service gRPC request timeout in seconds                         | 1s                    |
| MF_THINGS_AUTH_GRPC_URL     | Things service gRPC url                                              | localhost:8183        |
| MF_THINGS_AUTH_GRPC_TIMEOUT | Things service gRPC request timeout in seconds                       | 1s                    |


## Deployment

The service is distributed as Docker container. The following snippet provides a compose file template that can be used to deploy the service container locally:

```yaml
version: "3"
services:
  re:
    image: mainflux/re:[version]
    container_name: [instance name]
    ports:
      - [host machine port]:[configured HTTP port]
    environment:
      MF_RULES_LOG_LEVEL: [Rules log level]
      MF_RULES_HTTP_PORT: [Rules HTTP port]
      MF_RULES_SERVER_CERT: [Path to server certificate file]
      MF_RULES_SERVER_KEY: [Path to server key file]
      MF_RULES_SINGLE_USER_EMAIL: [User email for single user mode]
      MF_RULES_SINGLE_USER_TOKEN: [User token for single user mode]
      MF_RULES_CLIENT_TLS: [Flag to turn on/off TLS]
      MF_RULES_CA_CERTS: [Path to trusted CAs]
      MF_KUIPER_URL: [Kuiper rules engine url]
      MF_JAEGER_URL: [Jaeger server url]
      MF_AUTH_GRPC_URL: [Auth gRPC url]
      MF_AUTH_GRPC_TIMEOUT: [Auth request timeout in seconds]
      MF_THINGS_AUTH_GRPC_URL: [Things gRPC url]
      MF_THINGS_AUTH_GRPC_TIMEOUT: [Things request timeout in seconds]
```

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the rules service
make rules

# copy binary to bin
make install

# set the environment variables and run the service
MF_RULES_LOG_LEVEL: [Rules log level] \
MF_RULES_HTTP_PORT: [Rules HTTP port] \
MF_RULES_SERVER_CERT: [Path to server certificate file] \
MF_RULES_SERVER_KEY: [Path to server key file] \
MF_RULES_SINGLE_USER_EMAIL: [User email for single user mode] \
MF_RULES_SINGLE_USER_TOKEN: [User token for single user mode] \
MF_RULES_CLIENT_TLS: [Flag to turn on/off TLS] \
MF_RULES_CA_CERTS: [Path to trusted CAs] \
MF_KUIPER_URL: [Kuiper rules engine url] \
MF_JAEGER_URL: [Jaeger server url] \
MF_AUTH_GRPC_URL: [Auth gRPC url] \
MF_AUTH_GRPC_TIMEOUT: [Auth request timeout in seconds] \
MF_THINGS_AUTH_GRPC_URL: [Things gRPC url] \
MF_THINGS_AUTH_GRPC_TIMEOUT: [Things request timeout in seconds] \
$GOBIN/mainflux-rules
```

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](openapi.yaml).

[doc]: http://mainflux.readthedocs.io
