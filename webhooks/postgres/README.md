# Webhooks - DB Schema

```sql
CREATE TABLE IF NOT EXISTS webhooks (
    id          UUID UNIQUE, 
    thing_id    UUID NOT NULL,
    group_id    UUID NOT NULL,
    name        VARCHAR(254) NOT NULL,
    url         VARCHAR(254) NOT NULL,
    headers     JSONB,
    metadata    JSONB,    
    PRIMARY KEY (thing_id, name)
);
```