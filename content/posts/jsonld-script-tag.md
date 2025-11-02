---
title: JSON-LD Script Tags
description: Learn how to embed and load JSON-LD data using script tags in Tens City
datePublished: 2025-11-02T00:00:00Z
author:
  name: Tens City Team
  type: Organization
  url: https://tens.city
tags:
  - json-ld
  - script-tags
  - embedding
collection: guides
lang: en
slug: jsonld-script-tag
---

# JSON-LD Script Tag and Permalink Feature

The tens-city web component supports loading and embedding JSON-LD data in multiple ways.

## Embedding JSON-LD

You can embed JSON-LD directly in your HTML using script tags:

```html
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "Example Article"
}
</script>
```

## Loading from URL Parameters

Load JSON-LD data via URL parameters:

- `?data=<json>` - Load JSON directly from the URL
- `?cid=<CID>` - Load JSON-LD by Content Identifier

## Auto-update Feature

When you edit JSON-LD in the ACE editor, the embedded script tag automatically updates to reflect your changes.

## Permalink Generation

The permalink anchor dynamically updates with the editor content, making it easy to share your work.
