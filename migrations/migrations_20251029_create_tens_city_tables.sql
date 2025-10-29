-- Create tables for storing sealed objects and optional normalized quads
-- Designed for Supabase (Postgres). Run via psql or Supabase migrations.
-- Requires superuser to create extensions in some setups; Supabase typically allows pgcrypto.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Primary object store (sealed objects)
CREATE TABLE IF NOT EXISTS public.objects (
  cid text PRIMARY KEY,                -- computed by the app (SealJSONLD)
  owner_uuid uuid NOT NULL,            -- Supabase auth.uid() stored as UUID for ownership
  raw jsonb NOT NULL,                  -- original JSON-LD
  canonical text NOT NULL,             -- canonical N-Quads (UTF-8)
  canonical_sha256 bytea GENERATED ALWAYS AS (digest(canonical, 'sha256')) STORED,
  storage_path text NULL,              -- optional Supabase Storage path if you store files in Storage
  created_at timestamptz NOT NULL DEFAULT now()
);

-- JSONB GIN index for fast containment queries
CREATE INDEX IF NOT EXISTS objects_raw_gin ON public.objects USING GIN (raw);

-- Quick lookup by @type (common for object queries)
CREATE INDEX IF NOT EXISTS objects_type_idx ON public.objects ((raw->>'@type'));

-- Signature metadata (one-to-many)
CREATE TABLE IF NOT EXISTS public.signatures (
  id bigserial PRIMARY KEY,
  cid text NOT NULL REFERENCES public.objects(cid) ON DELETE CASCADE,
  signer_address text NOT NULL,
  signature text NOT NULL,
  use_personal_sign boolean DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- Optional normalized quads table for SQL-based graph queries
CREATE TABLE IF NOT EXISTS public.quads (
  id bigserial PRIMARY KEY,
  cid text NOT NULL REFERENCES public.objects(cid) ON DELETE CASCADE,
  subject text NOT NULL,
  predicate text NOT NULL,
  object text NOT NULL,                 -- store full lexical form; for literals include datatype/lang
  object_is_literal boolean NOT NULL DEFAULT true,
  graph text NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- Indexes for efficient graph lookups
CREATE INDEX IF NOT EXISTS quads_sp_idx ON public.quads (subject, predicate);
CREATE INDEX IF NOT EXISTS quads_p_idx ON public.quads (predicate);
CREATE INDEX IF NOT EXISTS quads_g_idx ON public.quads (graph);
CREATE UNIQUE INDEX IF NOT EXISTS quads_unique ON public.quads (subject, predicate, object, coalesce(graph, ''));

-- A convenience view
CREATE OR REPLACE VIEW public.object_metadata AS
SELECT cid, owner_uuid, raw->>'@type' AS type, raw->>'@version' AS version, created_at
FROM public.objects;