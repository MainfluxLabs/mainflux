# Things - DB Schema

```sql
CREATE TABLE public.group_roles (
    group_id uuid NOT NULL,
    member_id uuid NOT NULL,
    role character varying(15),

    CONSTRAINT group_policies_pkey PRIMARY KEY (group_id, member_id),
    CONSTRAINT group_policies_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.groups(id) ON DELETE CASCADE
);

CREATE TABLE public.groups (
    id uuid UNIQUE NOT NULL,
    org_id uuid NOT NULL,
    name character varying(254) NOT NULL,
    description character varying(1024),
    metadata jsonb,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,

    CONSTRAINT groups_pkey PRIMARY KEY (id),
    CONSTRAINT org_name UNIQUE (org_id, name)
);

CREATE TABLE public.profiles (
    id uuid UNIQUE NOT NULL,
    group_id uuid NOT NULL,
    name character varying(1024) NOT NULL,
    config jsonb,
    metadata jsonb,

    CONSTRAINT group_name_prs UNIQUE (group_id, name),
    CONSTRAINT profiles_pkey PRIMARY KEY (id),
    CONSTRAINT channels_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.groups(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE public.things (
    id uuid UNIQUE NOT NULL,
    group_id uuid NOT NULL,
    key character varying(4096) UNIQUE NOT NULL,
    name character varying(1024) NOT NULL,
    metadata jsonb,
    profile_id uuid NOT NULL,

    CONSTRAINT group_name_ths UNIQUE (group_id, name),
    CONSTRAINT things_pkey PRIMARY KEY (id),
    CONSTRAINT fk_things_profile_id FOREIGN KEY (profile_id) REFERENCES public.profiles(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT things_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.groups(id) ON UPDATE CASCADE ON DELETE CASCADE
);
```