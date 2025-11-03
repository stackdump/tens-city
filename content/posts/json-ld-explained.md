---
title: JSON-LD Explained
description: Understanding structured data and why it matters for your blog
datePublished: 2025-10-30T00:00:00Z
author:
  name: Data Team
  type: Organization
  url: https://tens.city
tags:
  - json-ld
  - structured-data
  - seo
  - schema.org
collection: guides
lang: en
slug: json-ld-explained
draft: false
---

# JSON-LD Explained

JSON-LD (JSON for Linking Data) is a lightweight way to add structured data to your web pages. Let's explore why it matters and how it works.

## What is JSON-LD?

JSON-LD is a format for encoding linked data using JSON. It's designed to be easy to read and write by humans and machines.

```json
{
  "@context": "https://schema.org",
  "@type": "BlogPosting",
  "headline": "JSON-LD Explained",
  "author": {
    "@type": "Organization",
    "name": "Data Team"
  },
  "datePublished": "2025-10-30T00:00:00Z"
}
```

## Why Use JSON-LD?

### 1. **Better Search Results**

Search engines use structured data to create rich snippets:

- Star ratings
- Recipe cards
- Event details
- Product information

### 2. **Semantic Web**

Help machines understand your content's meaning, not just its presentation.

### 3. **No Markup Pollution**

Unlike microdata, JSON-LD lives in a `<script>` tag, keeping your HTML clean:

```html
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "My Article"
}
</script>
```

## Common Schema.org Types

### BlogPosting

```json
{
  "@context": "https://schema.org",
  "@type": "BlogPosting",
  "headline": "My Blog Post",
  "author": {
    "@type": "Person",
    "name": "John Doe"
  },
  "datePublished": "2025-10-30",
  "image": "https://example.com/image.jpg"
}
```

### Person

```json
{
  "@context": "https://schema.org",
  "@type": "Person",
  "name": "Jane Smith",
  "jobTitle": "Software Engineer",
  "url": "https://janesmith.com",
  "sameAs": [
    "https://twitter.com/janesmith",
    "https://github.com/janesmith"
  ]
}
```

### Organization

```json
{
  "@context": "https://schema.org",
  "@type": "Organization",
  "name": "Tens City",
  "url": "https://tens.city",
  "logo": "https://tens.city/logo.png"
}
```

### Recipe (Bonus!)

```json
{
  "@context": "https://schema.org",
  "@type": "Recipe",
  "name": "Chocolate Chip Cookies",
  "recipeIngredient": [
    "2 cups flour",
    "1 cup sugar",
    "1 cup chocolate chips"
  ],
  "recipeInstructions": "Mix and bake at 350°F for 12 minutes"
}
```

## How Tens City Uses JSON-LD

Every blog post in Tens City automatically generates JSON-LD from its frontmatter:

**Markdown Frontmatter:**
```yaml
---
title: My Post
author:
  name: John Doe
  type: Person
datePublished: 2025-11-03T00:00:00Z
tags:
  - example
---
```

**Generated JSON-LD:**
```json
{
  "@context": "https://schema.org",
  "@type": "BlogPosting",
  "headline": "My Post",
  "author": {
    "@type": "Person",
    "name": "John Doe"
  },
  "datePublished": "2025-11-03T00:00:00Z",
  "keywords": ["example"]
}
```

## Testing Your JSON-LD

Use these tools to validate your structured data:

1. [Google's Rich Results Test](https://search.google.com/test/rich-results)
2. [Schema.org Validator](https://validator.schema.org/)
3. [JSON-LD Playground](https://json-ld.org/playground/)

## Best Practices

### ✅ Do

- Use official schema.org types when possible
- Include all required properties
- Validate your JSON-LD
- Keep data consistent with page content

### ❌ Don't

- Add misleading information
- Stuff keywords
- Use JSON-LD for spam
- Include hidden content

## The Future of Structured Data

JSON-LD is becoming the standard for structured data on the web:

- **Voice Assistants** use it to answer questions
- **Search Engines** create rich snippets
- **Social Media** previews content better
- **AI Systems** understand context

## Conclusion

JSON-LD is a simple, powerful way to make your content more discoverable and useful. Tens City automatically generates it for you, but understanding it helps you create better content.

Start thinking semantically, and the machines will thank you! 🤖
