# Auth Service

The Auth service is the central authentication and authorization component of the Mainflux IoT platform. It provides identity verification, role-based access control (RBAC), multi-tenant organization management, and a member invitation system. All other platform services communicate with Auth over its gRPC API to authenticate requests and enforce access policies.

## Authentication

The Auth service issues and validates [JSON Web Tokens (JWT)](https://jwt.io) used to authenticate users and services across the platform. Every authentication key contains the following fields:

| Field       | Description                                            |
|-------------|--------------------------------------------------------|
| `id`        | Unique key identifier                                  |
| `type`      | Key type (Login, API, or Recovery)                     |
| `issuer_id` | ID of the user who issued the key                      |
| `subject`   | Email address of the key owner                         |
| `issued_at` | Timestamp when the key was created                     |
| `expires_at`| Timestamp after which the key is no longer valid       |

### Key Types

**Login key** — Issued automatically when a user authenticates via the Users service. Short-lived (default: 10 hours). Required for all authenticated API requests except registration and login.

**API key** — Issued on-demand by the user. Unlike login keys, API keys have a configurable expiration period; if no expiration is set, the key never expires. API keys are the only key type that can be explicitly revoked. An API key grants the same permissions as a login key, with the exception that API keys cannot be used to issue new API keys.

**Recovery key** — A short-lived token (5 minutes) issued as part of the password recovery flow. It is single-use and cannot be listed or revoked manually.

Supported operations per key type:

| Operation | Login key | API key | Recovery key |
|-----------|:---------:|:-------:|:------------:|
| Issue     | ✓         | ✓       | ✓            |
| Validate  | ✓         | ✓       | ✓            |
| Retrieve  |           | ✓       |              |
| List      |           | ✓       |              |
| Revoke    |           | ✓       |              |

## Organizations

Organizations provide multi-tenancy within Mainflux. An organization is an isolated namespace that groups users and platform resources (things, groups, profiles) under a shared administrative boundary.

Every organization has the following attributes:

| Field         | Description                                          |
|---------------|------------------------------------------------------|
| `id`          | Unique organization identifier (UUID)                |
| `name`        | Human-readable name, unique within the platform      |
| `owner_id`    | ID of the user who created the organization          |
| `description` | Optional free-form description                       |
| `metadata`    | Arbitrary key-value pairs for custom attributes      |
| `created_at`  | Creation timestamp                                   |
| `updated_at`  | Last-modified timestamp                              |

The user who creates an organization is automatically assigned the `owner` role. Platform administrators can list and manage all organizations; regular users can only access organizations they are members of.

An organization can only be deleted when it has no members other than the owner.

## Role-Based Access Control

The Auth service enforces a two-level RBAC model.

### Platform roles

Platform roles apply globally across the entire Mainflux deployment and are assigned directly to user accounts.

| Role    | Description                                                        |
|---------|--------------------------------------------------------------------|
| `root`  | Super-administrator. Full access to all resources and operations.  |
| `admin` | Platform administrator. Can manage all organizations and users.    |

### Organization roles

Organization roles define a member's permissions within a specific organization.

| Role     | Description                                                                   |
|----------|-------------------------------------------------------------------------------|
| `owner`  | Full control over the organization. Cannot be changed or removed.             |
| `admin`  | Can manage memberships, update organization details, and manage invitations.  |
| `editor` | Can interact with organization resources.                                     |
| `viewer` | Read-only access to organization resources.                                   |

The `owner` role is assigned exclusively to the user who creates the organization and cannot be transferred or demoted.

## Organization Memberships

Members are users who belong to an organization and have been assigned an organization role. Membership management requires at least the `admin` role within the target organization.

Key membership rules:

- Members are added by email address; the platform resolves the corresponding user account.
- The default role for new members is `viewer`.
- The `owner` role cannot be assigned through the membership API; it is set automatically at organization creation.
- The owner cannot be removed from their own organization.
- A single organization can have multiple `admin` members.

## Organization Invites

The invitation system allows organization administrators to onboard new members asynchronously. An invite is sent via email and must be accepted or declined by the recipient.

An invitation includes:

| Field           | Description                                                            |
|-----------------|------------------------------------------------------------------------|
| `id`            | Unique invite identifier                                               |
| `invitee_email` | Email of the user being invited                                        |
| `invitee_role`  | Role to assign to the invitee upon acceptance                          |
| `org_id`        | Target organization                                                    |
| `group_invites` | Optional list of groups and roles to assign the invitee upon acceptance|
| `redirect_path` | URL path included in the invitation email for deep-linking             |
| `expires_at`    | Expiration timestamp (default: 7 days from creation)                  |
| `state`         | Current invite state                                                   |

### Invite states

| State      | Description                                              |
|------------|----------------------------------------------------------|
| `pending`  | Awaiting a response from the invitee.                    |
| `accepted` | Invitee accepted; they have been added to the organization. |
| `declined` | Invitee declined the invitation.                         |
| `revoked`  | Cancelled by the inviter before a response was received. |
| `expired`  | Invite was not responded to before its expiration time.  |

An invite can only be revoked by the user who sent it, and only while it is in the `pending` state. Only the invitee can accept or decline an invite.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                      | Description                                                              | Default        |
|-------------------------------|--------------------------------------------------------------------------|----------------|
| `MF_AUTH_LOG_LEVEL`           | Service log level (debug, info, warn, error)                             | error          |
| `MF_AUTH_DB_HOST`             | Database host address                                                    | localhost      |
| `MF_AUTH_DB_PORT`             | Database host port                                                       | 5432           |
| `MF_AUTH_DB_USER`             | Database user                                                            | mainflux       |
| `MF_AUTH_DB_PASSWORD`         | Database password                                                        | mainflux       |
| `MF_AUTH_DB`                  | Name of the database used by the service                                 | auth           |
| `MF_AUTH_DB_SSL_MODE`         | Database connection SSL mode (disable, require, verify-ca, verify-full)  | disable        |
| `MF_AUTH_DB_SSL_CERT`         | Path to the PEM encoded certificate file                                 |                |
| `MF_AUTH_DB_SSL_KEY`          | Path to the PEM encoded key file                                         |                |
| `MF_AUTH_DB_SSL_ROOT_CERT`    | Path to the PEM encoded root certificate file                            |                |
| `MF_AUTH_HTTP_PORT`           | Auth service HTTP port                                                   | 8180           |
| `MF_AUTH_GRPC_PORT`           | Auth service gRPC port                                                   | 8181           |
| `MF_AUTH_SERVER_CERT`         | Path to server certificate in PEM format                                 |                |
| `MF_AUTH_SERVER_KEY`          | Path to server key in PEM format                                         |                |
| `MF_AUTH_SECRET`              | Secret string used for signing tokens                                    | auth           |
| `MF_AUTH_LOGIN_TOKEN_DURATION`| Login key expiration period                                              | 10h            |
| `MF_INVITE_DURATION`          | Validity period for organization invitations                             | 168h           |
| `MF_JAEGER_URL`               | Jaeger server URL for distributed tracing. Leave empty to disable tracing. Docker value: `jaeger:6831` |                |
| `MF_AUTH_ES_URL`              | Event store (Redis) URL                                                  | redis://localhost:6379/0 |
| `MF_AUTH_GRPC_TIMEOUT`        | Timeout for outgoing gRPC calls (Things and Users services)              | 1s             |
| `MF_THINGS_AUTH_GRPC_URL`     | Things service auth gRPC URL                                             | localhost:8183 |
| `MF_THINGS_CLIENT_TLS`        | Enable TLS for Things gRPC connection                                    | false          |
| `MF_THINGS_CA_CERTS`          | Path to trusted CAs in PEM format for Things gRPC TLS                   |                |
| `MF_USERS_GRPC_URL`           | Users service gRPC URL                                                   | localhost:8184 |
| `MF_USERS_CLIENT_TLS`         | Enable TLS for Users gRPC connection                                     | false          |
| `MF_USERS_CA_CERTS`           | Path to trusted CAs in PEM format for Users gRPC TLS                    |                |
| `MF_HOST`                     | Frontend URL base used in invitation email links                         | http://localhost |
| `MF_EMAIL_HOST`               | Mail server host                                                         | localhost      |
| `MF_EMAIL_PORT`               | Mail server port                                                         | 25             |
| `MF_EMAIL_USERNAME`           | Mail server username                                                     |                |
| `MF_EMAIL_PASSWORD`           | Mail server password                                                     |                |
| `MF_EMAIL_FROM_ADDRESS`       | Sender email address                                                     |                |
| `MF_EMAIL_FROM_NAME`          | Sender display name                                                      |                |
| `MF_EMAIL_TEMPLATES_DIR`      | Path to the directory containing email templates used for invitation emails | `.`         |

> **Note:** If `MF_EMAIL_TEMPLATES_DIR` does not point to a valid directory containing the required templates, the service will start normally but invitation emails will not be sent.

## Deployment

The service is distributed as a Docker container. Refer to the [`auth` service section](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml#L71-L94) in the Docker Compose file for a reference deployment configuration.

To build and run the service manually:

```bash
# Download the latest version
go get github.com/MainfluxLabs/mainflux

cd $GOPATH/src/github.com/MainfluxLabs/mainflux

# Compile the service
make auth

# Copy binary to bin
make install

# Run the service
MF_AUTH_LOG_LEVEL=[log level] \
MF_AUTH_DB_HOST=[db host] \
MF_AUTH_DB_PORT=[db port] \
MF_AUTH_DB_USER=[db user] \
MF_AUTH_DB_PASSWORD=[db password] \
MF_AUTH_DB=[db name] \
MF_AUTH_DB_SSL_MODE=[ssl mode] \
MF_AUTH_HTTP_PORT=[http port] \
MF_AUTH_GRPC_PORT=[grpc port] \
MF_AUTH_SECRET=[signing secret] \
MF_AUTH_LOGIN_TOKEN_DURATION=[token duration] \
$GOBIN/mainfluxlabs-auth
```

## Usage

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).

For a broader overview of how authentication and authorization fit into the platform, refer to the [official documentation](https://mainfluxlabs.github.io/docs).
