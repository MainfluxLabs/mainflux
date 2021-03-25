# Rules

Rules service provides an HTTP API for managing [Kuiper](https://github.com/emqx/kuiper) rules engine entities. Use Rules to perform CRUD operations on streams - Kuiper entities defining message stream going from Mainflux into the Kuiper rules engine - and rules - Kuiper entities defining filtering and transforming operations on the message stream going from the Kuiper rules engine into the Mainflux.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable              | Description                                             | Default |
|-----------------------|---------------------------------------------------------|---------|
| MF_RULES_LOG_LEVEL   | Log level for re service (debug, info, warn, error) | error   |
| MF_RULES_HTTP_PORT   | Re service HTTP port                                | 9021    |
| MF_RULES_SERVER_CERT | Path to server certificate in pem format                |         |
| MF_RE_SERVER_KEY  | Path to server key in pem format                        |         |
| MF_JAEGER_URL         | Jaeger server URL                                       |         |
| MF_RULES_SECRET      | Re service secret                                   | secret  |

## Deployment

The service itself is distributed as Docker container. The following snippet
provides a compose file template that can be used to deploy the service container
locally:

```yaml
version: "3"
services:
  re:
    image: mainflux/re:[version]
    container_name: [instance name]
    ports:
      - [host machine port]:[configured HTTP port]
    environment:
      MF_RE_LOG_LEVEL: [Kit log level]
      MF_RE_HTTP_PORT: [Service HTTP port]
      MF_RE_SERVER_CERT: [String path to server cert in pem format]
      MF_RE_SERVER_KEY: [String path to server key in pem format]
      MF_JAEGER_URL: [Jaeger server URL]      
      MF_RE_SECRET: [Re service secret]
```

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the re
make re

# copy binary to bin
make install

# set the environment variables and run the service
MF_RE_LOG_LEVEL=[Kit log level] MF_RE_HTTP_PORT=[Service HTTP port] MF_RE_SERVER_CERT: [String path to server cert in pem format] MF_RE_SERVER_KEY: [String path to server key in pem format] MF_JAEGER_URL=[Jaeger server URL] MF_RE_SECRET: [Re service secret] $GOBIN/mainflux-kit
```

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](swagger.yaml).

[doc]: http://mainflux.readthedocs.io
