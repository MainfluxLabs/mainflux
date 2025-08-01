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
```