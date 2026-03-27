# Certs Service

The Certs service issues and manages X.509 certificates for IoT things, enabling mutual TLS (mTLS) authentication between devices and the platform. Certificates are signed by a local CA and identified by their serial number. Each certificate records the associated thing ID in the subject, making it straightforward to trace a TLS connection back to a specific device.

## Certificates

| Field              | Description                                                           |
|--------------------|-----------------------------------------------------------------------|
| `serial`           | Certificate serial number (hex string); used as the unique identifier |
| `thing_id`         | ID of the thing the certificate was issued for                        |
| `certificate`      | PEM-encoded X.509 certificate                                         |
| `issuing_ca`       | PEM-encoded certificate of the issuing CA                             |
| `ca_chain`         | Full CA certificate chain in PEM format                               |
| `private_key`      | PEM-encoded private key corresponding to the certificate              |
| `private_key_type` | Key algorithm: `rsa` or `ec`                                          |
| `expires_at`       | Certificate expiration timestamp (RFC 3339)                           |

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                    | Description                                                                | Default          |
|-----------------------------|----------------------------------------------------------------------------|------------------|
| `MF_CERTS_LOG_LEVEL`        | Log level for the Certs service (debug, info, warn, error)                 | error            |
| `MF_CERTS_HTTP_PORT`        | Certs service HTTP port                                                    | 8204             |
| `MF_JAEGER_URL`             | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                  |
| `MF_CERTS_DB_HOST`          | Database host address                                                      | localhost        |
| `MF_CERTS_DB_PORT`          | Database host port                                                         | 5432             |
| `MF_CERTS_DB_USER`          | Database user                                                              | mainflux         |
| `MF_CERTS_DB_PASS`          | Database password                                                          | mainflux         |
| `MF_CERTS_DB`               | Name of the database used by the service                                   | certs            |
| `MF_CERTS_DB_SSL_MODE`      | Database connection SSL mode (disable, require, verify-ca, verify-full)    | disable          |
| `MF_CERTS_DB_SSL_CERT`      | Path to the PEM encoded certificate file                                   |                  |
| `MF_CERTS_DB_SSL_KEY`       | Path to the PEM encoded key file                                           |                  |
| `MF_CERTS_DB_SSL_ROOT_CERT` | Path to the PEM encoded root certificate file                              |                  |
| `MF_CERTS_CLIENT_TLS`       | Flag that indicates if TLS should be turned on                             | false            |
| `MF_CERTS_CA_CERTS`         | Path to trusted CAs in PEM format                                          |                  |
| `MF_CERTS_SERVER_CERT`      | Path to server certificate in PEM format                                   |                  |
| `MF_CERTS_SERVER_KEY`       | Path to server key in PEM format                                           |                  |
| `MF_CERTS_SIGN_CA_PATH`     | Path to the CA certificate used for signing                                | ca.crt           |
| `MF_CERTS_SIGN_CA_KEY_PATH` | Path to the CA private key used for signing                                | ca.key           |
| `MF_CERTS_SIGN_HOURS_VALID` | Default certificate validity period (Go duration string)                   | 2048h            |
| `MF_CERTS_SIGN_RSA_BITS`    | RSA key size used when not specified in the request                        | 2048             |
| `MF_AUTH_GRPC_URL`          | Auth service gRPC URL                                                      | localhost:8181   |
| `MF_AUTH_GRPC_TIMEOUT`      | Auth service gRPC request timeout                                          | 1s               |
| `MF_THINGS_GRPC_URL`        | Things service gRPC URL (used to verify thing ownership)                   | localhost:8183   |
| `MF_THINGS_GRPC_TIMEOUT`    | Things service gRPC request timeout                                        | 1s               |
| `MF_SDK_CERTS_URL`          | Base URL of the Certs service, used by the SDK client                      | http://localhost |

## Deployment

The service itself is distributed as Docker container. Check the [`certs`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# Download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# Compile the certs service
make certs

# Copy binary to bin
make install

# Set the environment variables and run the service
MF_CERTS_LOG_LEVEL=[Certs log level] \
MF_CERTS_HTTP_PORT=[Certs service HTTP port] \
MF_CERTS_DB_HOST=[Database host address] \
MF_CERTS_DB_PORT=[Database host port] \
MF_CERTS_DB_USER=[Database user] \
MF_CERTS_DB_PASS=[Database password] \
MF_CERTS_DB=[Name of the database used by the service] \
MF_CERTS_SIGN_CA_PATH=[Path to CA certificate] \
MF_CERTS_SIGN_CA_KEY_PATH=[Path to CA private key] \
MF_CERTS_SIGN_HOURS_VALID=[Certificate validity period] \
MF_AUTH_GRPC_URL=[Auth service gRPC URL] \
$GOBIN/mainfluxlabs-certs
```

## Usage

First, obtain an authentication token:

```bash
TOK=$(curl -s -X POST http://localhost/tokens \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"12345678"}' | jq -r '.token')
```

### Issue a certificate

```bash
curl -s -X POST http://localhost:8204/certs \
  -H "Authorization: Bearer $TOK" \
  -H 'Content-Type: application/json' \
  -d '{"thing_id":"<thing_id>","key_bits":2048,"key_type":"rsa"}'
```

Supported key types: `rsa` (default), `ec`.

### List certificates for a thing

```bash
curl -s "http://localhost/svccerts/things/<thing_id>/serials" \
  -H "Authorization: Bearer $TOK"
```

### View a certificate

```bash
curl -s http://localhost:8204/certs/<serial> \
  -H "Authorization: Bearer $TOK"
```

### Renew a certificate

A certificate can only be renewed when it is **within 30 days of its expiration date**. Attempting to renew earlier returns an error. Renewal issues a new certificate with a fresh serial and extended validity; the old certificate is revoked.

```bash
curl -s -X PUT http://localhost:8204/certs/<serial> \
  -H "Authorization: Bearer $TOK"
```

### Revoke a certificate

```bash
curl -s -X DELETE http://localhost:8204/certs/<serial> \
  -H "Authorization: Bearer $TOK"
```

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).
