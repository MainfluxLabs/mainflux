# Things - DB Schema

```sql
CREATE TABLE group_memberships (
    group_id   UUID NOT NULL,
    member_id  UUID NOT NULL,
    role       VARCHAR(15),

    CONSTRAINT group_memberships_pkey PRIMARY KEY (group_id, member_id),
    CONSTRAINT group_memberships_group_id_fkey FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

CREATE TABLE groups (
    id          UUID UNIQUE NOT NULL,
    org_id      UUID NOT NULL,
    name        VARCHAR(254) NOT NULL,
    description VARCHAR(1024),
    metadata    JSONB,
    created_at  TIMESTAMP WITH TIME ZONE,
    updated_at  TIMESTAMP WITH TIME ZONE,

    CONSTRAINT  groups_pkey PRIMARY KEY (id),
    CONSTRAINT  org_name UNIQUE (org_id, name)
);

CREATE TABLE profiles (
    id        UUID UNIQUE NOT NULL,
    group_id  UUID NOT NULL,
    name      VARCHAR(1024) NOT NULL,
    config    JSONB,
    metadata  JSONB,

    CONSTRAINT group_name_prs UNIQUE (group_id, name),
    CONSTRAINT profiles_pkey PRIMARY KEY (id),
    CONSTRAINT channels_group_id_fkey FOREIGN KEY (group_id) REFERENCES groups(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE things (
    id           UUID UNIQUE NOT NULL,
    group_id     UUID NOT NULL,
    key          VARCHAR(4096) UNIQUE NOT NULL,
    external_key VARCHAR UNIQUE NULL,
    name         VARCHAR(1024) NOT NULL,
    metadata     JSONB,
    profile_id   UUID NOT NULL,

    CONSTRAINT group_name_ths UNIQUE (group_id, name),
    CONSTRAINT things_pkey PRIMARY KEY (id),
    CONSTRAINT fk_things_profile_id FOREIGN KEY (profile_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT things_group_id_fkey FOREIGN KEY (group_id) REFERENCES groups(id) ON UPDATE CASCADE ON DELETE CASCADE
);
```
