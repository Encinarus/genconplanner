-- Table: public.gencon_locations
CREATE TABLE IF NOT EXISTS public.gencon_locations (
    id integer NOT NULL,
    searchable_name text COLLATE pg_catalog."default" NOT NULL,
    location_label text COLLATE pg_catalog."default",
    map_location text COLLATE pg_catalog."default" NOT NULL,
    category character varying(50) COLLATE pg_catalog."default" NOT NULL,
    convention_id integer NOT NULL DEFAULT 0,
    CONSTRAINT gencon_locations_pkey PRIMARY KEY (id)
)
WITH (
    OIDS = FALSE
)
TABLESPACE pg_default;

ALTER TABLE public.gencon_locations
    OWNER to postgres;

-- Indexes for efficient lookup
CREATE INDEX IF NOT EXISTS idx_gencon_locations_category
    ON public.gencon_locations USING btree
    (category COLLATE pg_catalog."default");

CREATE INDEX IF NOT EXISTS idx_gencon_locations_searchable_name
    ON public.gencon_locations USING btree
    (searchable_name COLLATE pg_catalog."default");

CREATE INDEX IF NOT EXISTS idx_gencon_locations_label
    ON public.gencon_locations USING btree
    (location_label COLLATE pg_catalog."default");
