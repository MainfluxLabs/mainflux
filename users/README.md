# Users Service

The Users service manages user accounts, authentication, and platform-level access control. It provides multiple registration flows, JWT-based token issuance, OAuth integration, and a platform invite system for onboarding new users into organizations and groups.

## Users

A user account is the principal identity for all platform interactions.

| Field      | Description                                                                      |
|------------|----------------------------------------------------------------------------------|
| `id`       | Unique user identifier (UUID)                                                    |
| `email`    | Email address; serves as the unique login identifier                             |
| `password` | Bcrypt-hashed password (never returned in API responses)                         |
| `metadata` | Arbitrary key-value pairs for custom attributes                                  |
| `status`   | Account status: `enabled` or `disabled`                                          |
| `role`     | Platform-level role. Set to `root` for the root administrator; empty otherwise   |

## Registration Flows

The service supports three distinct registration paths:

| Flow                   | Endpoint                          | Description                                                                                  |
|------------------------|-----------------------------------|----------------------------------------------------------------------------------------------|
| Admin registration     | `POST /users`                     | Root or admin creates a user account directly. Requires an admin token.                      |
| Self-registration      | `POST /register`                  | User creates their own account. If `MF_REQUIRE_EMAIL_VERIFICATION=true`, a verification email is sent and `POST /register/verify` must be called with the token to complete registration. |
| Invite-based           | `POST /register/invite/{inviteId}`| User registers using a valid Platform Invite. The submitted email must match the invite.     |

## Authentication

Obtain a JWT by posting credentials to `POST /tokens`. The returned token is used as a Bearer token for all subsequent requests.

OAuth is supported for Google and GitHub via `GET /users/oauth/{provider}` (initiate) and `GET /users/oauth/{provider}/callback` (complete). On success the callback returns a JWT.

## Platform Invites

Platform Invites allow admins to invite unregistered users to the platform, optionally pre-assigning them to an organization and groups.

| Field           | Description                                                                     |
|-----------------|---------------------------------------------------------------------------------|
| `id`            | Unique invite identifier (UUID)                                                 |
| `invitee_email` | Email address of the invited user                                               |
| `state`         | Invite state: `pending`, `expired`, `revoked`, or `accepted`                   |
| `created_at`    | Creation timestamp (ISO 8601)                                                   |
| `expires_at`    | Expiration timestamp (ISO 8601). Controlled by `MF_INVITE_DURATION`.           |
| `org_id`        | Optional. Organization the invitee will be added to upon registration           |
| `role`          | Invitee's role in the organization: `admin`, `editor`, or `viewer`             |
| `group_invites` | Optional list of group assignments (`group_id` + `member_role`)                |

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                        | Description                                                             | Default          |
|---------------------------------|-------------------------------------------------------------------------|------------------|
| MF_USERS_LOG_LEVEL              | Log level for Users (debug, info, warn, error)                          | error            |
| MF_USERS_DB_HOST                | Database host address                                                   | localhost        |
| MF_USERS_DB_PORT                | Database host port                                                      | 5432             |
| MF_USERS_DB_USER                | Database user                                                           | mainflux         |
| MF_USERS_DB_PASSWORD            | Database password                                                       | mainflux         |
| MF_USERS_DB                     | Name of the database used by the service                                | users            |
| MF_USERS_DB_SSL_MODE            | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable          |
| MF_USERS_DB_SSL_CERT            | Path to the PEM encoded certificate file                                |                  |
| MF_USERS_DB_SSL_KEY             | Path to the PEM encoded key file                                        |                  |
| MF_USERS_DB_SSL_ROOT_CERT       | Path to the PEM encoded root certificate file                           |                  |
| MF_USERS_HTTP_PORT              | Users service HTTP port                                                 | 8180             |
| MF_USERS_GRPC_PORT              | Users service gRPC port (used by Auth service)                          | 8184             |
| MF_USERS_SERVER_CERT            | Path to server certificate in PEM format                                |                  |
| MF_USERS_SERVER_KEY             | Path to server key in PEM format                                        |                  |
| MF_USERS_ADMIN_EMAIL            | Default root admin user email, created on startup                       |                  |
| MF_USERS_ADMIN_PASSWORD         | Default root admin user password, created on startup                    |                  |
| MF_USERS_PASS_REGEX             | Regex pattern for validating user passwords                             | `^\S{8,}$`       |
| MF_USERS_SELF_REGISTER_ENABLED  | Allow users to self-register. If false, only admins can create accounts | true             |
| MF_AUTH_GRPC_URL                | Auth service gRPC URL                                                   | localhost:8181   |
| MF_AUTH_GRPC_TIMEOUT            | Auth service gRPC request timeout                                       | 1s               |
| MF_AUTH_CLIENT_TLS              | Enable TLS for Auth gRPC connection                                     | false            |
| MF_AUTH_CA_CERTS                | Path to trusted CAs in PEM format for Auth gRPC TLS                    |                  |
| MF_JAEGER_URL                   | Jaeger server URL for distributed tracing. Leave empty to disable tracing. Docker value: `jaeger:6831` |                  |
| MF_EMAIL_HOST                   | Mail server host                                                        | localhost        |
| MF_EMAIL_PORT                   | Mail server port                                                        | 25               |
| MF_EMAIL_USERNAME               | Mail server username                                                    |                  |
| MF_EMAIL_PASSWORD               | Mail server password                                                    |                  |
| MF_EMAIL_FROM_ADDRESS           | Email "from" address                                                    |                  |
| MF_EMAIL_FROM_NAME              | Email "from" name                                                       |                  |
| MF_EMAIL_TEMPLATES_DIR          | Path to the directory containing email templates                        | `.`              |
| MF_HOST                         | Frontend URL base included in user-facing emails                        | http://localhost |
| MF_REQUIRE_EMAIL_VERIFICATION   | Whether email verification is required during self-registration         | false            |
| MF_INVITE_DURATION              | Validity duration of created Platform Invites                           | 168h             |
| MF_REDIRECT_LOGIN_URL           | Path appended to `MF_HOST` for post-OAuth login redirects               |                  |
| MF_GOOGLE_CLIENT_ID             | Google OAuth application client ID                                      |                  |
| MF_GOOGLE_CLIENT_SECRET         | Google OAuth application client secret                                  |                  |
| MF_GOOGLE_REDIRECT_URL          | Google OAuth callback path (appended to `MF_HOST`)                      |                  |
| MF_GOOGLE_USER_INFO             | Google API endpoint for fetching user profile                           |                  |
| MF_GITHUB_CLIENT_ID             | GitHub OAuth application client ID                                      |                  |
| MF_GITHUB_CLIENT_SECRET         | GitHub OAuth application client secret                                  |                  |
| MF_GITHUB_REDIRECT_URL          | GitHub OAuth callback path (appended to `MF_HOST`)                      |                  |
| MF_GITHUB_USER_INFO             | GitHub API endpoint for fetching user profile                           |                  |
| MF_GITHUB_USER_EMAILS           | GitHub API endpoint for fetching user email addresses                   |                  |

> **Note:** If `MF_EMAIL_TEMPLATES_DIR` does not point to a valid directory containing the required templates, the service will start normally but email-based features (email verification, platform invites) will not work.

## Deployment

The service itself is distributed as Docker container. Check the [`users`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the users service
make users

# copy binary to bin
make install

# Set the environment variables and run the service
MF_USERS_LOG_LEVEL=[Users log level] \
MF_USERS_DB_HOST=[Database host address] \
MF_USERS_DB_PORT=[Database host port] \
MF_USERS_DB_USER=[Database user] \
MF_USERS_DB_PASSWORD=[Database password] \
MF_USERS_DB=[Name of the database used by the service] \
MF_USERS_HTTP_PORT=[Service HTTP port] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_EMAIL_HOST=[Mail server host] \
MF_EMAIL_PORT=[Mail server port] \
MF_EMAIL_USERNAME=[Mail server username] \
MF_EMAIL_PASSWORD=[Mail server password] \
MF_EMAIL_FROM_ADDRESS=[Email from address] \
MF_EMAIL_FROM_NAME=[Email from name] \
$GOBIN/mainfluxlabs-users
```

## Usage

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).
