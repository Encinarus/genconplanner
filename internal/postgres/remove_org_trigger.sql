-- Remove per-row organizer trigger and function to eliminate lock contention during bulk updates
DROP TRIGGER IF EXISTS update_org ON public.events;
DROP FUNCTION IF EXISTS public.update_org();
