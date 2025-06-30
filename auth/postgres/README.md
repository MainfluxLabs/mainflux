# Auth - DB Schema

```sql
CREATE TABLE keys (
    id character varying(254) NOT NULL,
    type smallint,
    subject character varying(254) NOT NULL,
    issuer_id uuid NOT NULL,
    issued_at timestamp without time zone NOT NULL,
    expires_at timestamp without time zone,
    CONSTRAINT keys_pkey PRIMARY KEY (id, issuer_id)
);

CREATE TABLE member_relations (
    member_id uuid NOT NULL,
    org_id uuid NOT NULL,
    role character varying(10) NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT member_relations_pkey PRIMARY KEY (member_id, org_id),
    CONSTRAINT member_relations_org_id_fkey FOREIGN KEY (org_id) REFERENCES orgs(id) ON DELETE CASCADE
);

CREATE TABLE orgs (
    id uuid NOT NULL,
    owner_id uuid NOT NULL,
    name character varying(254) NOT NULL,
    description character varying(1024),
    metadata jsonb,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT orgs_id_key UNIQUE (id),
    CONSTRAINT orgs_pkey PRIMARY KEY (id, owner_id)
);

CREATE TABLE users_roles (
    role character varying(12) CHECK (role IN ('root', 'admin')),
    user_id uuid NOT NULL,
    CONSTRAINT users_roles_pkey PRIMARY KEY (user_id),
);
```