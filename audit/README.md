# Audit Service

The Audit service consumes Redis Stream events published by all other Mainflux services, persists them in its own PostgreSQL database, and exposes an HTTP API for authorized users to query the recorded audit trail.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                       | Description                                                                 | Default                  |
|--------------------------------|-----------------------------------------------------------------------------|--------------------------|
| `MF_AUDIT_LOG_LEVEL`           | Service log level (debug, info, warn, error)                                | error                    |
| `MF_AUDIT_DB_HOST`             | Database host address                                                       | localhost                |
| `MF_AUDIT_DB_PORT`             | Database host port                                                          | 5432                     |
| `MF_AUDIT_DB_USER`             | Database user                                                               | mainflux                 |
| `MF_AUDIT_DB_PASS`             | Database password                                                           | mainflux                 |
| `MF_AUDIT_DB`                  | Name of the database used by the service                                    | audit                    |
| `MF_AUDIT_DB_SSL_MODE`         | Database connection SSL mode (disable, require, verify-ca, verify-full)     | disable                  |
| `MF_AUDIT_DB_SSL_CERT`         | Path to the PEM encoded certificate file                                    |                          |
| `MF_AUDIT_DB_SSL_KEY`          | Path to the PEM encoded key file                                            |                          |
| `MF_AUDIT_DB_SSL_ROOT_CERT`    | Path to the PEM encoded root certificate file                               |                          |
| `MF_AUDIT_HTTP_PORT`           | Audit service HTTP port                                                     | 9030                     |
| `MF_AUDIT_SERVER_CERT`         | Path to server certificate in PEM format                                    |                          |
| `MF_AUDIT_SERVER_KEY`          | Path to server key in PEM format                                            |                          |
| `MF_AUDIT_ES_URL`              | Event store (Redis) URL the service subscribes to                           | redis://localhost:6379/0 |
| `MF_JAEGER_URL`                | Jaeger server URL for distributed tracing. Leave empty to disable tracing.  |                          |
| `MF_AUTH_GRPC_URL`             | Auth service gRPC URL                                                       | localhost:8181           |
| `MF_AUTH_GRPC_TIMEOUT`         | Timeout for outgoing Auth gRPC calls                                        | 1s                       |
| `MF_AUTH_CLIENT_TLS`           | Enable TLS for Auth gRPC connection                                         | false                    |
| `MF_AUTH_CA_CERTS`             | Path to trusted CAs in PEM format for Auth gRPC TLS                         |                          |
| `MF_THINGS_AUTH_GRPC_URL`      | Things service auth gRPC URL                                                | localhost:8183           |
| `MF_THINGS_GRPC_TIMEOUT`       | Timeout for outgoing Things gRPC calls                                      | 1s                       |
| `MF_THINGS_CLIENT_TLS`         | Enable TLS for Things gRPC connection                                       | false                    |
| `MF_THINGS_CA_CERTS`           | Path to trusted CAs in PEM format for Things gRPC TLS                       |                          |

## Deployment

The service is shipped as part of the standard Mainflux Docker stack. After populating `docker/.env`, start the platform with:

```bash
make run
```

Or build and run the binary directly:

```bash
make audit
MF_AUDIT_HTTP_PORT=9030 \
MF_AUDIT_ES_URL=redis://localhost:6379/0 \
MF_AUTH_GRPC_URL=localhost:8181 \
MF_THINGS_AUTH_GRPC_URL=localhost:8183 \
./build/mainfluxlabs-audit
```
