# Consumers - Alarms - DB Schema


```sql
CREATE TABLE IF NOT EXISTS alarms (
    id          UUID PRIMARY KEY,
    thing_id    UUID NOT NULL,
    group_id    UUID NOT NULL,
    subtopic    VARCHAR(254),
    protocol    TEXT,
    payload     JSONB,
    created     BIGINT
)
```