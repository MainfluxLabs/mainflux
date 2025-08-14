# Rules - DB Schema

```sql
CREATE TABLE IF NOT EXISTS rules (
    id          UUID PRIMARY KEY,
    profile_id  UUID NOT NULL,
    group_id    UUID NOT NULL, 
    name        VARCHAR(254) NOT NULL,
    description VARCHAR(1024),
    conditions  JSONB NOT NULL,
    operator    VARCHAR(3) NOT NULL,
    actions     JSONB NOT NULL
);
```
