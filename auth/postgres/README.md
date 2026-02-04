# Auth - DB Schema

```sql
CREATE TABLE keys (
    id         VARCHAR(254) NOT NULL,
    type       SMALLINT,
    subject    VARCHAR(254) NOT NULL,
    issuer_id  UUID NOT NULL,
    issued_at  TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITHOUT TIME ZONE,
    CONSTRAINT keys_pkey PRIMARY KEY (id, issuer_id)
);

CREATE TABLE org_memberships (
    member_id  UUID NOT NULL,
    org_id     UUID NOT NULL,
    role       VARCHAR(10) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TOME ZONE,
    CONSTRAINT org_memberships_pkey PRIMARY KEY (member_id, org_id),
    CONSTRAINT org_memberships_org_id_fkey FOREIGN KEY (org_id) REFERENCES orgs(id) ON DELETE CASCADE
);

CREATE TABLE orgs (
    id          UUID NOT NULL,
    owner_id    UUID NOT NULL,
    name        VARCHAR(254) NOT NULL,
    description VARCHAR(1024),
    metadata    JSONB,
    created_at  TIMESTAMP WITH TOME ZONE,
    updated_at  TIMESTAMP WITH TIME ZONE,
    CONSTRAINT  orgs_id_key UNIQUE (id),
    CONSTRAINT  orgs_pkey PRIMARY KEY (id, owner_id)
);

CREATE TABLE users_roles (
    role       VARCHAR(12) CHECK (role IN ('root', 'admin')),
    user_id    UUID NOT NULL,
    CONSTRAINT users_roles_pkey PRIMARY KEY (user_id),
);

CREATE TABLE IF NOT EXISTS org_invites (
    id           UUID NOT NULL,
    invitee_id   UUID NOT NULL,         
    inviter_id   UUID NOT NULL,
    org_id       UUID NOT NULL,
    invitee_role VARCHAR(12) NOT NULL,
    created_at   TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    state        VARCHAR DEFAULT 'pending' NOT NULL,      
    FOREIGN KEY  (org_id) REFERENCES orgs (id) ON DELETE CASCADE,
    PRIMARY KEY  (id)
);

CREATE UNIQUE INDEX ux_org_invites_invitee_id_org_id on org_invites (invitee_id, org_id) WHERE state='pending';

CREATE TABLE IF NOT EXISTS org_invites_groups (
    org_invite_id  UUID NOT NULL,
    group_id       UUID NOT NULL,
    member_role    VARCHAR NOT NULL,
    PRIMARY KEY (org_invite_id, group_id),
    FOREIGN KEY (org_invite_id) REFERENCES org_invites (id) ON DELETE CASCADE
);
```
