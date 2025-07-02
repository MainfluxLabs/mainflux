# MQTT - DB Schema

```sql
CREATE TABLE IF NOT EXISTS subscriptions (
    subtopic    VARCHAR(1024),
    group_id  	UUID,
    thing_id    UUID,
    client_id   VARCHAR(256),
    status      VARCHAR(128),
    created_at  FLOAT,
    PRIMARY KEY (client_id, subtopic, group_id, thing_id)
);
```
