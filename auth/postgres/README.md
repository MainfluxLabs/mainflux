# Auth - DB Schema

```sql
CREATE TABLE public.keys (
    id character varying(254) NOT NULL,
    type smallint,
    subject character varying(254) NOT NULL,
    issuer_id uuid NOT NULL,
    issued_at timestamp without time zone NOT NULL,
    expires_at timestamp without time zone
);

CREATE TABLE public.member_relations (
    member_id uuid NOT NULL,
    org_id uuid NOT NULL,
    role character varying(10) NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE TABLE public.orgs (
    id uuid NOT NULL,
    owner_id uuid NOT NULL,
    name character varying(254) NOT NULL,
    description character varying(1024),
    metadata jsonb,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE TABLE public.users_roles (
    role character varying(12),
    user_id uuid NOT NULL,
    CONSTRAINT users_roles_role_check CHECK (((role)::text = ANY ((ARRAY['root'::character varying, 'admin'::character varying])::text[])))
);

ALTER TABLE ONLY public.keys
    ADD CONSTRAINT keys_pkey PRIMARY KEY (id, issuer_id);

ALTER TABLE ONLY public.member_relations
    ADD CONSTRAINT member_relations_pkey PRIMARY KEY (member_id, org_id);

ALTER TABLE ONLY public.orgs
    ADD CONSTRAINT orgs_id_key UNIQUE (id);

ALTER TABLE ONLY public.orgs
    ADD CONSTRAINT orgs_pkey PRIMARY KEY (id, owner_id);

ALTER TABLE ONLY public.users_roles
    ADD CONSTRAINT users_roles_pkey PRIMARY KEY (user_id);

ALTER TABLE ONLY public.member_relations
    ADD CONSTRAINT member_relations_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.orgs(id) ON DELETE CASCADE;

```