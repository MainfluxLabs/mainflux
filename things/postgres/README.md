# Things - DB Schema

```sql
CREATE TABLE public.group_roles (
    group_id uuid NOT NULL,
    member_id uuid NOT NULL,
    role character varying(15)
);

CREATE TABLE public.groups (
    id uuid NOT NULL,
    org_id uuid NOT NULL,
    name character varying(254) NOT NULL,
    description character varying(1024),
    metadata jsonb,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE TABLE public.profiles (
    id uuid NOT NULL,
    group_id uuid NOT NULL,
    name character varying(1024) NOT NULL,
    config jsonb,
    metadata jsonb
);

CREATE TABLE public.things (
    id uuid NOT NULL,
    group_id uuid NOT NULL,
    key character varying(4096) NOT NULL,
    name character varying(1024) NOT NULL,
    metadata jsonb,
    profile_id uuid NOT NULL
);

ALTER TABLE ONLY public.profiles
    ADD CONSTRAINT channels_id_key UNIQUE (id);

ALTER TABLE ONLY public.gorp_migrations
    ADD CONSTRAINT gorp_migrations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.profiles
    ADD CONSTRAINT group_name_prs UNIQUE (group_id, name);

ALTER TABLE ONLY public.things
    ADD CONSTRAINT group_name_ths UNIQUE (group_id, name);

ALTER TABLE ONLY public.group_roles
    ADD CONSTRAINT group_policies_pkey PRIMARY KEY (group_id, member_id);

ALTER TABLE ONLY public.groups
    ADD CONSTRAINT groups_id_key UNIQUE (id);

ALTER TABLE ONLY public.groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.groups
    ADD CONSTRAINT org_name UNIQUE (org_id, name);

ALTER TABLE ONLY public.profiles
    ADD CONSTRAINT profiles_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.things
    ADD CONSTRAINT things_id_key UNIQUE (id);

ALTER TABLE ONLY public.things
    ADD CONSTRAINT things_key_key UNIQUE (key);

ALTER TABLE ONLY public.things
    ADD CONSTRAINT things_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.profiles
    ADD CONSTRAINT channels_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.groups(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.things
    ADD CONSTRAINT fk_things_profile_id FOREIGN KEY (profile_id) REFERENCES public.profiles(id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE ONLY public.group_roles
    ADD CONSTRAINT group_policies_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.groups(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.things
    ADD CONSTRAINT things_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.groups(id) ON UPDATE CASCADE ON DELETE CASCADE;
```