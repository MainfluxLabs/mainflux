# Users - DB Schema

```sql
CREATE TYPE user_status AS ENUM ('enabled', 'disabled');

CREATE TABLE IF NOT EXISTS users (
    id          UUID UNIQUE NOT NULL,
    email       VARCHAR(254) UNIQUE NOT NULL,
    password    CHAR(60) NOT NULL,
    metadata    JSONB,
    status      USER_STATUS NOT NULL DEFAULT 'enabled',
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS verifications (
    email      VARCHAR(254) NOT NULL,
    password   CHAR(60) NOT NULL,
    token      UUID UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS invites_platform (
    id            UUID NOT NULL,
    invitee_email VARCHAR NOT NULL,
    created_at    TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ,
    state         VARCHAR DEFAULT 'pending' NOT NULL
);

CREATE UNIQUE INDEX unique_invitee_email_pending on invites_platform (invitee_email) WHERE state='pending';
```