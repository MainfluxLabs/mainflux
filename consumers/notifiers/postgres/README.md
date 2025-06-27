# Consumers - Notifiers - DB Schema
    
```sql
CREATE TABLE IF NOT EXISTS notifiers (
    id          UUID PRIMARY KEY,
    group_id    UUID NOT NULL,
    name        VARCHAR(254) NOT NULL,						
    contacts    VARCHAR(512) NOT NULL,
    metadata    JSONB,
    CONSTRAINT  unique_group_name UNIQUE (group_id, name)
)
```