# OTA - DB Schema

```sql
CREATE TABLE IF NOT EXISTS updates (
    id          UUID PRIMARY KEY,
    group_id    UUID NOT NULL,
    file_name   VARCHAR(255) NOT NULL,
    version     VARCHAR(254) NOT NULL,
    file_size   BIGINT NOT NULL DEFAULT 0,
    checksum    VARCHAR(64) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    timeout     BIGINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_updates_group_id ON updates (group_id);

CREATE TABLE IF NOT EXISTS thing_updates (
    update_id        UUID NOT NULL REFERENCES updates (id) ON DELETE CASCADE,
    thing_id         UUID NOT NULL,
    status           VARCHAR(16) NOT NULL,
    error            VARCHAR(255) NOT NULL DEFAULT '',
    reported_version VARCHAR(254) NOT NULL DEFAULT '',
    updated_at       TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (update_id, thing_id)
);

CREATE INDEX IF NOT EXISTS idx_thing_updates_thing_id_status ON thing_updates (thing_id, status);
CREATE INDEX IF NOT EXISTS idx_thing_updates_status_updated_at ON thing_updates (status, updated_at);
```

`timeout` is stored in seconds.
