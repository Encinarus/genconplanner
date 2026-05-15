-- Enable pgcrypto for sha256 support
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Migration to add cluster_id for optimized event grouping
ALTER TABLE events ADD COLUMN IF NOT EXISTS cluster_id TEXT;

-- Update the cluster update trigger to populate cluster_id
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

  -- 2. Generate the cluster_id fingerprint
  -- We use SHA-256 for a robust, collision-resistant fingerprint.
  NEW.cluster_id := encode(digest(NEW.cluster_key::text || coalesce(NEW.short_category, ''), 'sha256'), 'hex');

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- Backfill existing rows
UPDATE events 
SET cluster_id = encode(digest(cluster_key::text || coalesce(short_category, ''), 'sha256'), 'hex')
WHERE cluster_id IS NULL;

-- Create index for fast grouping
CREATE INDEX IF NOT EXISTS cluster_id_idx ON events (cluster_id);
