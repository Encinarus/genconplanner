-- Table: public.boardgame

-- DROP TABLE public.boardgame;

CREATE TABLE public.boardgame
(
    name text COLLATE pg_catalog."default" NOT NULL,
    bgg_id integer NOT NULL,
    family_ids integer[],
    last_update date,
    CONSTRAINT boardgame_pkey PRIMARY KEY (bgg_id)
)
    WITH (
        OIDS = FALSE
    )
    TABLESPACE pg_default;

ALTER TABLE public.boardgame
    OWNER to postgres;

-- Index: bg_name_idx

-- DROP INDEX public.bg_name_idx;

CREATE INDEX bg_name_idx
    ON public.boardgame USING btree
        (name COLLATE pg_catalog."default")
    TABLESPACE pg_default;

-- Table: public.starred_events

-- DROP TABLE public.starred_events;

CREATE TABLE public.starred_events
(
  email text COLLATE pg_catalog."default" NOT NULL,
  event_id character varying(12) COLLATE pg_catalog."default" NOT NULL,
  CONSTRAINT starred_events_pkey PRIMARY KEY (event_id, email)
)
  WITH (
    OIDS = FALSE
  )
  TABLESPACE pg_default;

ALTER TABLE public.starred_events
  OWNER to postgres;

-- Table: public.users

-- DROP TABLE public.users;

CREATE TABLE public.users
(
  email text COLLATE pg_catalog."default" NOT NULL,
  display_name text COLLATE pg_catalog."default",
  CONSTRAINT users_pkey PRIMARY KEY (email)
)
  WITH (
    OIDS = FALSE
  )
  TABLESPACE pg_default;

ALTER TABLE public.users
  OWNER to postgres;

-- Table: public.events

-- DROP TABLE public.events;

CREATE TABLE public.events
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

ALTER TABLE public.events
  OWNER to postgres;

-- Index: dow_index

-- DROP INDEX public.dow_index;

CREATE INDEX dow_index
  ON public.events USING btree
    (day_of_week)
  TABLESPACE pg_default;

-- Index: cat_hash_index

-- DROP INDEX public.cat_hash_index;

CREATE INDEX cat_hash_index
  ON public.events USING hash
    (short_category COLLATE pg_catalog."default")
  TABLESPACE pg_default;

-- Index: cluster_key_index

-- DROP INDEX public.cluster_key_index;

CREATE INDEX cluster_key_index
  ON public.events USING gin
    (cluster_key)
  TABLESPACE pg_default;

-- Index: year_hash_index

-- DROP INDEX public.year_hash_index;

CREATE INDEX year_hash_index
  ON public.events USING hash
    (year)
  TABLESPACE pg_default;

-- Index: start_time_index

-- DROP INDEX public.start_time_index;

CREATE INDEX start_time_index
  ON public.events USING hash
    (start_time)
  TABLESPACE pg_default;

-- Index: title_index

-- DROP INDEX public.title_index;

CREATE INDEX title_index
  ON public.events USING btree
    (title COLLATE pg_catalog."default")
  TABLESPACE pg_default;

-- Index: search_index

-- DROP INDEX public.search_index;

CREATE INDEX search_index
  ON public.events USING gin
    (search_key)
  TABLESPACE pg_default;

-- Trigger: update_dow

-- DROP TRIGGER update_dow on public.events

CREATE FUNCTION update_dow() RETURNS trigger AS $update_dow$
BEGIN
  NEW.day_of_week = EXTRACT (DOW FROM new.start_time AT TIME ZONE 'EDT');
  RETURN NEW;
END;
$update_dow$ LANGUAGE plpgsql;

CREATE TRIGGER update_dow BEFORE INSERT OR UPDATE ON public.events
  FOR EACH ROW EXECUTE PROCEDURE update_dow();


-- Trigger: cluster_vectorupdate

-- DROP TRIGGER cluster_vectorupdate ON public.events;

CREATE TRIGGER cluster_vectorupdate
  BEFORE INSERT OR UPDATE
  ON public.events
  FOR EACH ROW
EXECUTE PROCEDURE tsvector_update_trigger('cluster_key', 'pg_catalog.english', 'title', 'short_description', 'org_group', 'event_type', 'game_system', 'rules_edition');

-- Trigger: desc_vectorupdate

-- DROP TRIGGER desc_vectorupdate ON public.events;

CREATE TRIGGER desc_vectorupdate
  BEFORE INSERT OR UPDATE
  ON public.events
  FOR EACH ROW
EXECUTE PROCEDURE tsvector_update_trigger('desc_tsv', 'pg_catalog.english', 'short_description', 'long_description');

-- Trigger: title_vectorupdate

-- DROP TRIGGER title_vectorupdate ON public.events;

CREATE TRIGGER title_vectorupdate
  BEFORE INSERT OR UPDATE
  ON public.events
  FOR EACH ROW
EXECUTE PROCEDURE tsvector_update_trigger('title_tsv', 'pg_catalog.english', 'title');

-- Trigger: search_vectorupdate

-- DROP TRIGGER search_vectorupdate ON public.events;

CREATE TRIGGER search_vectorupdate
  BEFORE INSERT OR UPDATE
  ON public.events
  FOR EACH ROW
EXECUTE PROCEDURE tsvector_update_trigger('search_key', 'pg_catalog.english', 'title', 'short_description', 'long_description', 'org_group', 'event_type', 'event_id', 'game_system');