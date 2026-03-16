# Rules - DB Schema

```sql
CREATE TABLE IF NOT EXISTS rules (
    id          UUID PRIMARY KEY,
    group_id    UUID NOT NULL, 
    name        VARCHAR(254) NOT NULL,
    description VARCHAR(1024),
    conditions  JSONB NOT NULL,
    operator    VARCHAR(3) NOT NULL,
    actions     JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS rules_things (
    rule_id   UUID NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    thing_id  UUID NOT NULL,
    PRIMARY KEY (rule_id, thing_id)
);

CREATE TABLE IF NOT EXISTS lua_scripts (
    id          UUID NOT NULL,
    group_id    UUID NOT NULL,
    script      VARCHAR(65535) NOT NULL,
    name        VARCHAR NOT NULL,
    description VARCHAR NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS lua_scripts_things (
    thing_id      UUID NOT NULL,
    lua_script_id UUID NOT NULL,
    PRIMARY KEY (thing_id, lua_script_id),
    FOREIGN KEY (lua_script_id) REFERENCES lua_scripts (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS lua_script_runs (
    id          UUID NOT NULL,
    script_id   UUID NOT NULL,
    thing_id    UUID NOT NULL,
    logs        JSONB NOT NULL,
    started_at  TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    status      TEXT NOT NULL,
    error       TEXT NULL,
    PRIMARY KEY (id),
    FOREIGN KEY (script_id) REFERENCES lua_scripts (id)
);
```
