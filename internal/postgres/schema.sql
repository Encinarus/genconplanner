-- Enable pgcrypto for sha256 support
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Table: public.users
CREATE TABLE public.users
(
  email text COLLATE pg_catalog."default" NOT NULL,
  display_name text COLLATE pg_catalog."default",
  gencon_name text COLLATE pg_catalog."default",
  gencon_id text COLLATE pg_catalog."default",
  gencon_email text COLLATE pg_catalog."default",
  wishlist_constraints_initialized boolean NOT NULL DEFAULT false,
  wishlist_dirty boolean NOT NULL DEFAULT true,
  wishlist_updated_at timestamp without time zone NOT NULL DEFAULT now(),
  CONSTRAINT users_pkey PRIMARY KEY (email)
)
WITH (
  OIDS = FALSE
)
TABLESPACE pg_default;

ALTER TABLE public.users
  OWNER to postgres;

-- Table: public.parties
CREATE TABLE public.parties
(
    party_id SERIAL PRIMARY KEY,
    name     text COLLATE pg_catalog."default" NOT NULL,
    year     integer                           NOT NULL,
    leader_email text REFERENCES public.users(email),
    short_code text UNIQUE
)
WITH (
    OIDS = FALSE
)
TABLESPACE pg_default;

ALTER TABLE public.parties
    OWNER to postgres;

-- Table: public.party_members
CREATE TABLE public.party_members
(
    party_id integer NOT NULL,
    email text COLLATE pg_catalog."default" NOT NULL,
    display_name text COLLATE pg_catalog."default",
    gencon_name text COLLATE pg_catalog."default",
    gencon_id text COLLATE pg_catalog."default",
    gencon_email text COLLATE pg_catalog."default",
    CONSTRAINT party_members_pkey PRIMARY KEY (party_id, email)
)
WITH (
    OIDS = FALSE
)
TABLESPACE pg_default;

ALTER TABLE public.party_members
    OWNER to postgres;

-- Table: public.boardgame
CREATE TABLE public.boardgame
(
    name text COLLATE pg_catalog."default" NOT NULL,
    bgg_id integer NOT NULL,
    family_ids integer[],
    last_update date,
    num_ratings integer,
    avg_ratings double precision,
    num_weights integer,
    avg_weight double precision,
    min_players integer,
    max_players integer,
    best_players text COLLATE pg_catalog."default",
    description text COLLATE pg_catalog."default",
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
CREATE INDEX bg_name_idx
    ON public.boardgame USING btree
        (name COLLATE pg_catalog."default")
    TABLESPACE pg_default;

-- Table: public.boardgame_family
CREATE TABLE public.boardgame_family
(
    name text COLLATE pg_catalog."default" NOT NULL,
    bgg_id integer NOT NULL,
    game_ids integer[],
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
CREATE INDEX bgf_name_idx
    ON public.boardgame_family USING btree
        (name COLLATE pg_catalog."default")
    TABLESPACE pg_default;

-- Type: public.interest_tier
DO $$ BEGIN
    CREATE TYPE public.interest_tier AS ENUM ('must_have', 'very_interested', 'somewhat_interested', 'not_interested', 'purchased');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Table: public.starred_events
CREATE TABLE public.starred_events
(
  email text COLLATE pg_catalog."default" NOT NULL,
  event_id character varying(13) COLLATE pg_catalog."default" NOT NULL,
  level character varying(10) COLLATE pg_catalog."default" NOT NULL,
  tier public.interest_tier NOT NULL DEFAULT 'very_interested',
  CONSTRAINT starred_events_pkey PRIMARY KEY (event_id, email, level)
)
WITH (
  OIDS = FALSE
)
TABLESPACE pg_default;

ALTER TABLE public.starred_events
  OWNER to postgres;

-- Table: public.events
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
    cluster_id text,
    CONSTRAINT event_pkey PRIMARY KEY (event_id)
)
WITH (
  OIDS = FALSE
)
TABLESPACE pg_default;

ALTER TABLE public.events
  OWNER to postgres;

-- Index: dow_index
CREATE INDEX dow_index
  ON public.events USING btree
    (day_of_week)
  TABLESPACE pg_default;

-- Index: org_group
CREATE INDEX org_group
    ON public.events USING btree
        (org_group COLLATE pg_catalog."default")
    TABLESPACE pg_default;

-- Index: cat_hash_index
CREATE INDEX cat_hash_index
  ON public.events USING hash
    (short_category COLLATE pg_catalog."default")
  TABLESPACE pg_default;

-- Index: cluster_key_index
CREATE INDEX cluster_key_index
  ON public.events USING gin
    (cluster_key)
  TABLESPACE pg_default;

-- Index: year_hash_index
CREATE INDEX year_hash_index
  ON public.events USING hash
    (year)
  TABLESPACE pg_default;

-- Index: start_time_index
CREATE INDEX start_time_index
  ON public.events USING hash
    (start_time)
  TABLESPACE pg_default;

-- Index: title_index
CREATE INDEX title_index
  ON public.events USING btree
    (title COLLATE pg_catalog."default")
  TABLESPACE pg_default;

-- Index: search_index
CREATE INDEX search_index
  ON public.events USING gin
    (search_key)
  TABLESPACE pg_default;

-- Index: cluster_id_idx
CREATE INDEX cluster_id_idx ON events (cluster_id);

-- Trigger Function: update_dow
CREATE OR REPLACE FUNCTION update_dow() RETURNS trigger AS $update_dow$
BEGIN
  NEW.day_of_week = EXTRACT (DOW FROM new.start_time AT TIME ZONE 'EDT');
  RETURN NEW;
END;
$update_dow$ LANGUAGE plpgsql;

CREATE TRIGGER update_dow BEFORE INSERT OR UPDATE ON public.events
  FOR EACH ROW EXECUTE PROCEDURE update_dow();

-- Trigger Function: custer_update_trigger
CREATE OR REPLACE FUNCTION custer_update_trigger() RETURNS trigger AS $$
BEGIN
  -- 1. Maintain the existing cluster_key for search
  NEW.cluster_key :=
    to_tsvector('pg_catalog.english', coalesce(NEW.title, '')) ||
    to_tsvector('pg_catalog.english', coalesce(NEW.short_description)) ||
    to_tsvector('pg_catalog.english', coalesce(NEW.org_group)) ||
    to_tsvector('pg_catalog.english', coalesce(NEW.event_type)) ||
    to_tsvector('pg_catalog.english', coalesce(NEW.game_system)) ||
    to_tsvector('pg_catalog.english', coalesce(NEW.rules_edition)) ||
    to_tsvector('pg_catalog.english', CONCAT(NEW.year, 'eventyear'));

  -- 2. Generate the cluster_id fingerprint using SHA-256
  NEW.cluster_id := encode(digest(NEW.cluster_key::text || coalesce(NEW.short_category, ''), 'sha256'), 'hex');

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER cluster_vectorupdate
  BEFORE INSERT OR UPDATE
  ON public.events
  FOR EACH ROW
  EXECUTE FUNCTION custer_update_trigger();

-- Trigger: desc_vectorupdate
CREATE TRIGGER desc_vectorupdate
  BEFORE INSERT OR UPDATE
  ON public.events
  FOR EACH ROW
  EXECUTE PROCEDURE tsvector_update_trigger('desc_tsv', 'pg_catalog.english', 'short_description', 'long_description');

-- Trigger: title_vectorupdate
CREATE TRIGGER title_vectorupdate
  BEFORE INSERT OR UPDATE
  ON public.events
  FOR EACH ROW
  EXECUTE PROCEDURE tsvector_update_trigger('title_tsv', 'pg_catalog.english', 'title');

-- Trigger: search_vectorupdate
CREATE TRIGGER search_vectorupdate
  BEFORE INSERT OR UPDATE
  ON public.events
  FOR EACH ROW
  EXECUTE PROCEDURE tsvector_update_trigger('search_key', 'pg_catalog.english', 'title', 'short_description', 'long_description', 'org_group', 'event_type', 'event_id', 'game_system');

-- SEQUENCE: public.orgs_id_seq
CREATE SEQUENCE public.orgs_id_seq;

ALTER SEQUENCE public.orgs_id_seq
    OWNER TO postgres;

-- Table: public.orgs
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
CREATE INDEX alias_idx
    ON public.orgs USING btree
        (alias COLLATE pg_catalog."default" text_pattern_ops)
    TABLESPACE pg_default;

-- Table: public.update_log
CREATE TABLE public.update_log (
    id SERIAL PRIMARY KEY,
    timestamp timestamp with time zone DEFAULT now(),
    success boolean,
    events_seen integer,
    events_inserted integer,
    events_updated integer,
    events_deleted integer,
    events_unchanged integer,
    error_message text
);

-- Table: public.user_wishlist_constraints
CREATE TABLE public.user_wishlist_constraints (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL REFERENCES public.users(email),
    day_of_week INTEGER NOT NULL DEFAULT -1, -- -1 for "Every Day", 0-6 for Sun-Sat
    start_hour INTEGER NOT NULL,
    start_minute INTEGER NOT NULL DEFAULT 0,
    end_hour INTEGER NOT NULL,
    end_minute INTEGER NOT NULL DEFAULT 0,
    min_duration_minutes INTEGER NOT NULL DEFAULT 0
);

-- Table: public.user_wishlist_cache
CREATE TABLE public.user_wishlist_cache (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL REFERENCES public.users(email),
    year INTEGER NOT NULL,
    event_id TEXT NOT NULL,
    rank INTEGER NOT NULL,
    status TEXT NOT NULL,
    reasoning TEXT[] NOT NULL,
    score DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_user_wishlist_cache_email_year ON public.user_wishlist_cache(email, year);

-- Table: public.party_tickets
CREATE TABLE public.party_tickets (
    ticket_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    party_id integer NOT NULL REFERENCES public.parties(party_id) ON DELETE CASCADE,
    event_id character varying(13) NOT NULL REFERENCES public.events(event_id) ON DELETE CASCADE,
    year integer NOT NULL,
    
    purchaser_email text NOT NULL,
    gencon_purchaser_name text DEFAULT '',
    gencon_ticket_id text,               -- Gen Con Transaction/Ticket ID (e.g., "TXN98765-1")
    gencon_recipient_name text NOT NULL, -- e.g., "Alice Smith", "Dave Smith", "Another ticket for me"
    gencon_recipient_id text,            -- Gen Con Account ID if known (e.g., "88341")
    
    holder_email text NOT NULL, -- Defaults to purchaser_email for unmapped/guest passes
    
    ticket_type character varying(20) NOT NULL,       -- 'physical' | 'eticket'
    ticket_status character varying(20) NOT NULL DEFAULT 'active', -- 'active' | 'returned'
    transfer_status character varying(30) NOT NULL DEFAULT 'none', -- 'none' | 'name_only_transfer' | 'pending_gencon_transfer' | 'completed'
    gencon_return_id text,
    
    created_at timestamp with time zone DEFAULT now(),
    last_modified timestamp with time zone DEFAULT now()
);

CREATE INDEX idx_party_tickets_party_year ON public.party_tickets(party_id, year);
CREATE INDEX idx_party_tickets_event ON public.party_tickets(event_id);
CREATE INDEX idx_party_tickets_holder ON public.party_tickets(holder_email, year);

-- Table: public.ticket_transfers
CREATE TABLE public.ticket_transfers (
    transfer_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id uuid NOT NULL REFERENCES public.party_tickets(ticket_id) ON DELETE CASCADE,
    party_id integer NOT NULL REFERENCES public.parties(party_id) ON DELETE CASCADE,
    
    from_email text NOT NULL REFERENCES public.users(email),
    to_email text NOT NULL REFERENCES public.users(email),
    
    transfer_type character varying(20) NOT NULL, -- 'name_only' | 'eticket'
    status character varying(20) NOT NULL,        -- 'pending' | 'accepted' | 'rejected' | 'completed' | 'cancelled'
    
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

CREATE INDEX idx_ticket_transfers_party ON public.ticket_transfers(party_id);

-- Real-time Pub/Sub Notifications Function and Trigger
CREATE OR REPLACE FUNCTION public.notify_party_interest_update()
RETURNS TRIGGER AS $$
DECLARE
    v_party_id INT;
    v_cluster_id TEXT;
    v_max_tier TEXT;
BEGIN
    -- Get cluster_id for the event
    SELECT cluster_id INTO v_cluster_id FROM public.events WHERE event_id = COALESCE(NEW.event_id, OLD.event_id) LIMIT 1;
    
    -- Calculate max tier for this user and cluster
    SELECT 
        CASE 
            WHEN bool_or(se.tier = 'purchased') THEN 'purchased'
            WHEN bool_or(se.tier = 'must_have') THEN 'must_have'
            WHEN bool_or(se.tier = 'very_interested') THEN 'very_interested'
            WHEN bool_or(se.tier = 'somewhat_interested') THEN 'somewhat_interested'
            ELSE ''
        END INTO v_max_tier
    FROM public.starred_events se
    JOIN public.events e ON se.event_id = e.event_id
    WHERE se.email = COALESCE(NEW.email, OLD.email) AND e.cluster_id = v_cluster_id;

    -- Find active parties for the user
    FOR v_party_id IN 
        SELECT party_id FROM public.party_members WHERE email = COALESCE(NEW.email, OLD.email)
    LOOP
        PERFORM pg_notify('party_updates', json_build_object(
            'party_id', v_party_id,
            'cluster_id', v_cluster_id,
            'event_id', COALESCE(NEW.event_id, OLD.event_id),
            'email', COALESCE(NEW.email, OLD.email),
            'tier', COALESCE(v_max_tier, '')
        )::text);
    END LOOP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trig_party_interest_update
AFTER INSERT OR UPDATE OR DELETE ON public.starred_events
FOR EACH ROW EXECUTE FUNCTION public.notify_party_interest_update();

-- Table: public.admin_users
CREATE TABLE public.admin_users (
  email text NOT NULL,
  CONSTRAINT admin_users_pkey PRIMARY KEY (email)
);

ALTER TABLE public.admin_users
  OWNER to postgres;

