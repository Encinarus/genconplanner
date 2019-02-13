
-- Table: public.event

-- DROP TABLE public.event;

CREATE TABLE public.event
(
  event_id character varying(12) COLLATE pg_catalog."default" NOT NULL,
  active boolean,
  org_group text COLLATE pg_catalog."default",
  title text COLLATE pg_catalog."default",
  short_description text COLLATE pg_catalog."default",
  long_description text COLLATE pg_catalog."default",
  event_type character varying(50) COLLATE pg_catalog."default",
  game_system text COLLATE pg_catalog."default",
  rules_edition text COLLATE pg_catalog."default",
  min_players integer,
  max_players integer,
  age_required character varying(50) COLLATE pg_catalog."default",
  experience_required text COLLATE pg_catalog."default",
  materials_provided boolean,
  start_time timestamp with time zone,
  duration integer,
  end_time timestamp with time zone,
  gm_names text COLLATE pg_catalog."default",
  website text COLLATE pg_catalog."default",
  email text COLLATE pg_catalog."default",
  tournament boolean,
  round_number integer,
  total_rounds integer,
  min_play_time integer,
  attendee_registration text COLLATE pg_catalog."default",
  cost integer,
  location text COLLATE pg_catalog."default",
  room_name text COLLATE pg_catalog."default",
  table_number text COLLATE pg_catalog."default",
  special_category text COLLATE pg_catalog."default",
  tickets_available integer,
  year integer,
  cluster_key tsvector,
  last_modified timestamp with time zone,
  short_category character varying(4) COLLATE pg_catalog."default",
  title_tsv tsvector,
  desc_tsv tsvector,
  CONSTRAINT event_pkey PRIMARY KEY (event_id)
)
  WITH (
    OIDS = FALSE
  )
  TABLESPACE pg_default;

-- Trigger: cluster_vectorupdate

-- DROP TRIGGER cluster_vectorupdate ON public.event;

CREATE TRIGGER cluster_vectorupdate
  BEFORE INSERT OR UPDATE
  ON public.event
  FOR EACH ROW
EXECUTE PROCEDURE tsvector_update_trigger('cluster_key', 'pg_catalog.english', 'title', 'short_description', 'long_description', 'event_type', 'game_system');

-- Trigger: desc_vectorupdate

-- DROP TRIGGER desc_vectorupdate ON public.event;

CREATE TRIGGER desc_vectorupdate
  BEFORE INSERT OR UPDATE
  ON public.event
  FOR EACH ROW
EXECUTE PROCEDURE tsvector_update_trigger('desc_tsv', 'pg_catalog.english', 'short_description', 'long_description');

-- Trigger: title_vectorupdate

-- DROP TRIGGER title_vectorupdate ON public.event;

CREATE TRIGGER title_vectorupdate
  BEFORE INSERT OR UPDATE
  ON public.event
  FOR EACH ROW
EXECUTE PROCEDURE tsvector_update_trigger('title_tsv', 'pg_catalog.english', 'title');