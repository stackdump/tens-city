-- Add pg_jsonschema extension for JSON-LD validation
-- This provides guarantees of JSON-LD correctness from the database layer

-- Enable the pg_jsonschema extension
CREATE EXTENSION IF NOT EXISTS pg_jsonschema;

-- Define a basic JSON-LD schema that ensures:
-- 1. The document has a @context field (required for JSON-LD)
-- 2. The @context can be a string, object, array, or null
-- 3. Additional properties are allowed (for the actual JSON-LD content)
CREATE OR REPLACE FUNCTION get_jsonld_base_schema() RETURNS json AS $$
  SELECT '{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
      "@context": {
        "oneOf": [
          {"type": "string"},
          {"type": "object"},
          {"type": "array"},
          {"type": "null"}
        ]
      }
    },
    "required": ["@context"]
  }'::json;
$$ LANGUAGE SQL IMMUTABLE;

-- Add a CHECK constraint to the objects table to validate JSON-LD structure
-- This ensures that all raw JSON-LD documents have at minimum a @context field
ALTER TABLE public.objects
  ADD CONSTRAINT objects_raw_is_valid_jsonld
  CHECK (jsonb_matches_schema(get_jsonld_base_schema(), raw));

-- Add a helpful comment explaining the constraint
COMMENT ON CONSTRAINT objects_raw_is_valid_jsonld ON public.objects IS
  'Validates that the raw JSONB field contains valid JSON-LD with required @context field';
