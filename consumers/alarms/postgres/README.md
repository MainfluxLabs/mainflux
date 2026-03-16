# Consumers - Alarms - DB Schema


```sql
CREATE TABLE IF NOT EXISTS alarms (
    id          UUID PRIMARY KEY,
    thing_id    UUID NOT NULL,
    group_id    UUID NOT NULL,
    rule_id     UUID,
    script_id   UUID,
    subtopic    VARCHAR(254),
    protocol    TEXT,
    payload     JSONB,
    created     BIGINT
)
```