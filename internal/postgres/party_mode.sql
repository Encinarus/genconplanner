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
