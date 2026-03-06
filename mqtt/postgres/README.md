# MQTT - DB Schema

```sql
CREATE TABLE IF NOT EXISTS subscriptions (
    topic       VARCHAR(1024),
    group_id  	UUID,
    thing_id    UUID,
    client_id   VARCHAR(256),
    status      VARCHAR(128),
    created_at  FLOAT,
    PRIMARY KEY (client_id, topic, group_id, thing_id)
);
```
