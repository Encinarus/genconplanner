-- Phase 1: Basic Party Management & Visibility

-- Add leader_email to parties table
ALTER TABLE public.parties ADD COLUMN IF NOT EXISTS leader_email TEXT REFERENCES public.users(email);

-- For any existing parties, we'll need to assign a leader.
-- This query assigns the first member (alphabetically by email) as the leader.
UPDATE public.parties p
SET leader_email = (
    SELECT email 
    FROM public.party_members pm 
    WHERE pm.party_id = p.party_id 
    ORDER BY email ASC 
    LIMIT 1
)
WHERE leader_email IS NULL;

-- Phase 2: Personal Interest Tiers & Solo Optimization

-- Define the interest tier enum safely
DO $$ BEGIN
    CREATE TYPE public.interest_tier AS ENUM ('must_have', 'very_interested', 'somewhat_interested', 'not_interested', 'purchased');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

ALTER TYPE public.interest_tier ADD VALUE IF NOT EXISTS 'not_interested';
ALTER TYPE public.interest_tier ADD VALUE IF NOT EXISTS 'purchased';

-- Add the tier column to starred_events
ALTER TABLE public.starred_events ADD COLUMN IF NOT EXISTS tier public.interest_tier NOT NULL DEFAULT 'very_interested';

-- Update primary key of starred_events to include level
UPDATE public.starred_events SET level = 'event' WHERE level IS NULL;
ALTER TABLE public.starred_events DROP CONSTRAINT IF EXISTS starred_events_pkey;
ALTER TABLE public.starred_events ALTER COLUMN level SET NOT NULL;
ALTER TABLE public.starred_events ADD CONSTRAINT starred_events_pkey PRIMARY KEY (event_id, email, level);

-- Phase 2.6: Wishlist Constraints
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS wishlist_constraints_initialized BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS wishlist_dirty BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS wishlist_updated_at TIMESTAMP NOT NULL DEFAULT NOW();

CREATE TABLE IF NOT EXISTS public.user_wishlist_constraints (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL REFERENCES public.users(email),
    day_of_week INTEGER NOT NULL DEFAULT -1, -- -1 for "Every Day", 0-6 for Sun-Sat
    start_hour INTEGER NOT NULL,
    start_minute INTEGER NOT NULL DEFAULT 0,
    end_hour INTEGER NOT NULL,
    end_minute INTEGER NOT NULL DEFAULT 0
);

-- Phase 2.7: Wishlist Caching
CREATE TABLE IF NOT EXISTS public.user_wishlist_cache (
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

CREATE INDEX IF NOT EXISTS idx_user_wishlist_cache_email_year ON public.user_wishlist_cache(email, year);

-- Phase 3: Party Join Links
ALTER TABLE public.parties ADD COLUMN IF NOT EXISTS short_code TEXT UNIQUE;

-- Generate random 8 character hex strings for any existing parties
UPDATE public.parties 
SET short_code = encode(gen_random_bytes(4), 'hex')
WHERE short_code IS NULL;

-- Phase 4: Collaborative Group View & Shared Interests (Real-time Pub/Sub)
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

-- Phase 5: Member Gen Con Info
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS gencon_name TEXT;
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS gencon_id TEXT;
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS gencon_email TEXT;
ALTER TABLE public.party_members ADD COLUMN IF NOT EXISTS display_name TEXT;
ALTER TABLE public.party_members ADD COLUMN IF NOT EXISTS gencon_name TEXT;
ALTER TABLE public.party_members ADD COLUMN IF NOT EXISTS gencon_id TEXT;
ALTER TABLE public.party_members ADD COLUMN IF NOT EXISTS gencon_email TEXT;
