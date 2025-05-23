-- Table: public.parties

-- DROP TABLE public.parties;

CREATE TABLE public.parties
(
    party_id SERIAL PRIMARY KEY,
    name     text COLLATE pg_catalog."default" NOT NULL,
    year     integer                           NOT NULL
)
    WITH (
        OIDS = FALSE
    )
    TABLESPACE pg_default;

ALTER TABLE public.parties
    OWNER to postgres;

-- Table: public.party_members

-- DROP TABLE public.party_members;

CREATE TABLE public.party_members
(
    party_id integer NOT NULL,
    email text COLLATE pg_catalog."default" NOT NULL,
    CONSTRAINT party_members_pkey PRIMARY KEY (party_id, email)
)
    WITH (
        OIDS = FALSE
    )
    TABLESPACE pg_default;

ALTER TABLE public.party_members
    OWNER to postgres;

-- Table: public.boardgame

-- DROP TABLE public.boardgame;

CREATE TABLE public.boardgame
(
    name text COLLATE pg_catalog."default" NOT NULL,
    bgg_id integer NOT NULL,
    family_ids integer[],
    last_update date,
    num_ratings integer,
    avg_ratings double precision,
    year_published integer,
    type text COLLATE pg_catalog."default",
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

-- Table: public.boardgame_family

-- DROP TABLE public.boardgame_family;

CREATE TABLE public.boardgame_family
(
    name text COLLATE pg_catalog."default" NOT NULL,
    bgg_id integer NOT NULL,
    last_update date,
    CONSTRAINT boardgame_family_pkey PRIMARY KEY (bgg_id)
)
    WITH (
        OIDS = FALSE
    )
    TABLESPACE pg_default;

ALTER TABLE public.boardgame_family
    OWNER to postgres;

-- Index: bgf_name_idx

-- DROP INDEX public.bgf_name_idx;

CREATE INDEX bgf_name_idx
    ON public.boardgame_family USING btree
        (name COLLATE pg_catalog."default")
    TABLESPACE pg_default;

-- Table: public.starred_events

-- DROP TABLE public.starred_events;

CREATE TABLE public.starred_events
(
  email text COLLATE pg_catalog."default" NOT NULL,
  event_id character varying(13) COLLATE pg_catalog."default" NOT NULL,
  level character varying(10) COLLATE pg_catalog."default",
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
    event_id character varying(13) COLLATE pg_catalog."default" NOT NULL,
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
    day_of_week integer,
    search_key tsvector,
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

-- DROP INDEX public.org_group;

CREATE INDEX org_group
    ON public.events USING btree
        (org_group COLLATE pg_catalog."default")
    TABLESPACE pg_default;

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

-- DROP FUNCTION custer_update_trigger ON public.events;

CREATE FUNCTION custer_update_trigger() RETURNS trigger AS $$
begin
  new.cluster_key :=
    to_tsvector('pg_catalog.english', coalesce(new.title, '')) ||
    to_tsvector('pg_catalog.english', coalesce(new.short_description)) ||
    to_tsvector('pg_catalog.english', coalesce(new.org_group)) ||
    to_tsvector('pg_catalog.english', coalesce(new.event_type)) ||
    to_tsvector('pg_catalog.english', coalesce(new.game_system)) ||
    to_tsvector('pg_catalog.english', coalesce(new.rules_edition)) ||
    to_tsvector('pg_catalog.english', CONCAT(new.year, 'eventyear'));
  return new;
end
$$ LANGUAGE plpgsql;

-- DROP TRIGGER cluster_vectorupdate ON public.events;

CREATE TRIGGER cluster_vectorupdate
  BEFORE INSERT OR UPDATE
  ON public.events
  FOR EACH ROW
EXECUTE FUNCTION custer_update_trigger();

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

-- SEQUENCE: public.orgs_id_seq

-- DROP SEQUENCE public.orgs_id_seq;

CREATE SEQUENCE public.orgs_id_seq;

ALTER SEQUENCE public.orgs_id_seq
    OWNER TO postgres;

-- Table: public.orgs

-- DROP TABLE public.orgs;

CREATE TABLE public.orgs
(
    id integer NOT NULL DEFAULT nextval('orgs_id_seq'::regclass),
    alias text COLLATE pg_catalog."default" NOT NULL,
    CONSTRAINT orgs_pkey PRIMARY KEY (id, alias)
)
    WITH (
        OIDS = FALSE
    )
    TABLESPACE pg_default;

ALTER TABLE public.orgs
    OWNER to postgres;

-- Index: alias_idx

-- DROP INDEX public.alias_idx;

CREATE INDEX alias_idx
    ON public.orgs USING btree
        (alias COLLATE pg_catalog."default" text_pattern_ops)
    TABLESPACE pg_default;

-- FUNCTION: public.update_org()

-- DROP FUNCTION public.update_org();

CREATE OR REPLACE FUNCTION public.update_org()
    RETURNS trigger
    LANGUAGE 'plpgsql'
    COST 100
    VOLATILE NOT LEAKPROOF
AS $BODY$
BEGIN
    IF new.org_group = '' OR new.org_group is null THEN
        RETURN NULL;
    END IF;

    INSERT INTO orgs(alias)
    SELECT new.org_group
    WHERE NOT EXISTS (
        SELECT alias FROM orgs WHERE new.org_group = alias
    );

    UPDATE orgs o
    SET id = (SELECT MIN(o2.id) FROM orgs o2
              WHERE TRANSLATE(LOWER(o2.alias), '''.",!:; ', '')
                        = TRANSLATE(LOWER(o.alias), '''.",!:; ', ''))
    WHERE o.alias = new.org_group;
    RETURN NEW;
END
$BODY$;

ALTER FUNCTION public.update_org()
    OWNER TO postgres;

-- Trigger: update_org

-- DROP TRIGGER update_org ON public.events;

CREATE TRIGGER update_org
    BEFORE INSERT OR UPDATE OF org_group
    ON public.events
    FOR EACH ROW
EXECUTE PROCEDURE public.update_org();

-- insert into orgs(alias)
-- (
-- 	select
-- 	  distinct e.org_group
-- 	from events as e
-- )

-- update orgs o
-- set id = (select min(o2.id) from orgs o2
-- 		  where translate(lower(o2.alias), '''.",!:; ', '') = translate(lower(o.alias), '''.",!:; ', '') )
