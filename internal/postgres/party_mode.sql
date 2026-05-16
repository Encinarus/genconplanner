-- Phase 1: Basic Party Management & Visibility

-- Add leader_email to parties table
ALTER TABLE public.parties ADD COLUMN leader_email TEXT REFERENCES public.users(email);

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

-- Define the interest tier enum
CREATE TYPE public.interest_tier AS ENUM ('must_have', 'very_interested', 'somewhat_interested');

-- Add the tier column to starred_events
ALTER TABLE public.starred_events ADD COLUMN tier public.interest_tier NOT NULL DEFAULT 'very_interested';

-- Phase 2.6: Wishlist Constraints
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS wishlist_constraints_initialized BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS wishlist_dirty BOOLEAN NOT NULL DEFAULT TRUE;

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
