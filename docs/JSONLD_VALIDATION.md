# JSON-LD Validation with pg_jsonschema

This document describes the JSON-LD validation features implemented using PostgreSQL's `pg_jsonschema` extension.

## Overview

The tens-city database enforces JSON-LD correctness at the database layer using the `pg_jsonschema` PostgreSQL extension. This provides automatic validation that ensures all stored objects meet basic JSON-LD requirements.

## What is Validated

The validation schema enforces the following requirements:

1. **@context field is required**: Every JSON-LD document must have a `@context` field
2. **@context can be multiple types**: The `@context` field can be:
   - A string (e.g., `"https://schema.org"`)
   - An object (e.g., `{"name": "http://schema.org/name"}`)
   - An array (for multiple contexts)
   - null (per JSON-LD spec)

## How It Works

### Migration

The validation is implemented in the migration file `migrations/migrations_20251030_add_jsonld_validation.sql`:

```sql
-- Enable the pg_jsonschema extension
CREATE EXTENSION IF NOT EXISTS pg_jsonschema;

-- Define a JSON-LD validation schema
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

-- Add validation constraint to objects table
ALTER TABLE public.objects
  ADD CONSTRAINT objects_raw_is_valid_jsonld
  CHECK (jsonb_matches_schema(get_jsonld_base_schema(), raw));
```

### Database-Level Enforcement

When you try to insert an object without a `@context` field, the database will reject it:

```sql
-- This will fail
INSERT INTO public.objects (cid, owner_uuid, raw, canonical)
VALUES ('test-cid', '00000000-0000-0000-0000-000000000000', 
        '{"name": "Missing context"}', 'canonical data');
-- ERROR: new row for relation "objects" violates check constraint "objects_raw_is_valid_jsonld"

-- This will succeed
INSERT INTO public.objects (cid, owner_uuid, raw, canonical)
VALUES ('test-cid', '00000000-0000-0000-0000-000000000000', 
        '{"@context": "https://schema.org", "name": "Valid JSON-LD"}', 'canonical data');
```

## Examples

### Valid JSON-LD Documents

All of these are valid and will be accepted:

1. **String context:**
```json
{
  "@context": "https://schema.org",
  "name": "Example"
}
```

2. **Object context:**
```json
{
  "@context": {
    "name": "http://schema.org/name",
    "description": "http://schema.org/description"
  },
  "name": "Example"
}
```

3. **Array context:**
```json
{
  "@context": [
    "https://schema.org",
    {"custom": "http://example.com/custom"}
  ],
  "name": "Example"
}
```

4. **Null context:**
```json
{
  "@context": null,
  "name": "Example"
}
```

### Invalid JSON-LD Documents

These will be rejected:

1. **Missing @context:**
```json
{
  "name": "Example"
}
```

2. **Wrong @context type:**
```json
{
  "@context": 123,
  "name": "Example"
}
```

## Testing

The validation is thoroughly tested in `cmd/edge/validation_test.go`. Run the tests with:

```bash
go test -v ./cmd/edge -run TestJSONLDValidation
```

## Benefits

1. **Data Integrity**: Ensures all objects are valid JSON-LD at the database level
2. **No Application Changes**: Existing code continues to work, but invalid data is rejected
3. **Clear Error Messages**: When validation fails, you get a clear database constraint error
4. **Performance**: Validation happens at insert/update time, not on every read

## Extending the Schema

To add more validation rules, modify the `get_jsonld_base_schema()` function. For example, to also require a `@type` field:

```sql
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
      },
      "@type": {
        "oneOf": [
          {"type": "string"},
          {"type": "array"}
        ]
      }
    },
    "required": ["@context", "@type"]
  }'::json;
$$ LANGUAGE SQL IMMUTABLE;
```

Note: After modifying the schema function, you may need to drop and recreate the constraint for existing tables.

## Resources

- [pg_jsonschema GitHub](https://github.com/supabase/pg_jsonschema)
- [JSON-LD Specification](https://www.w3.org/TR/json-ld11/)
- [JSON Schema Documentation](https://json-schema.org/)
