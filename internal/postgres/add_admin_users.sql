-- Table: public.admin_users
CREATE TABLE public.admin_users (
  email text NOT NULL,
  CONSTRAINT admin_users_pkey PRIMARY KEY (email)
);

ALTER TABLE public.admin_users
  OWNER to postgres;
