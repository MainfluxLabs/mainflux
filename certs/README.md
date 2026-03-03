# Certs Service

Issues X.509 certificates for things, enabling mTLS authentication.

## Features

- **Issue** certificates for things (RSA or ECDSA)
- **List** certificates by thing ID
- **View** certificate details by serial number
- **Renew** certificates approaching expiration
- **Revoke** certificates by serial number

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
  -d '{"thing_id":"<thing_id>", "key_bits":2048, "key_type":"rsa"}'
```

Supported key types: `rsa` (default), `ec`/`ecdsa`

### List certificates for a thing

```bash
curl -s http://localhost:8204/certs?thing_id=<thing_id> \
  -H "Authorization: Bearer $TOK"
```

### View a certificate

```bash
curl -s http://localhost:8204/certs/<serial> \
  -H "Authorization: Bearer $TOK"
```

### Renew a certificate

Certificates can be renewed when they are within 30 days of expiration.

```bash
curl -s -X PUT http://localhost:8204/certs/<serial>/renew \
  -H "Authorization: Bearer $TOK"
```

### Revoke a certificate

```bash
curl -s -X DELETE http://localhost:8204/certs/<serial> \
  -H "Authorization: Bearer $TOK"
```

## Configuration

| Environment Variable        | Description                                | Default  |
|-----------------------------|--------------------------------------------|----------|
| `MF_CERTS_SIGN_CA_PATH`     | Path to CA certificate.                    | `ca.crt` |
| `MF_CERTS_SIGN_CA_KEY_PATH` | Path to CA private key                     | `ca.key` |
| `MF_CERTS_SIGN_HOURS_VALID` | Certificate validity period                | `2048h`  |
| `MF_CERTS_SIGN_RSA_BITS`    | RSA key size (if not specified in request) | `2048`   |
