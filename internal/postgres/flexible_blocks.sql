-- Add min_duration_minutes to user_wishlist_constraints
-- 0 means hard block, > 0 means flexible block requiring a gap of that size.
ALTER TABLE public.user_wishlist_constraints 
ADD COLUMN IF NOT EXISTS min_duration_minutes INTEGER NOT NULL DEFAULT 0;
