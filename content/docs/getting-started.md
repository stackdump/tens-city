---
title: Getting Started with Tens City
description: A comprehensive guide to getting started with the Tens City JSON-LD platform
datePublished: 2025-11-02T00:00:00Z
dateModified: 2025-11-02T00:00:00Z
author:
  name: Tens City Team
  type: Organization
  url: https://tens.city
tags:
  - getting-started
  - tutorial
  - introduction
collection: guides
lang: en
slug: getting-started
draft: false
---

# Getting Started with Tens City

Welcome to **Tens City** — a minimal, filesystem-based platform for managing JSON-LD objects with dignity and simplicity.

## What is Tens City?

Tens City provides stable, non-transient spots for your data. Think of it as a tent city rather than skyscrapers:

- **Anyone can stake a corner** — no permission required
- **Minimal structure that belongs to you** — dignity through shelter
- **No HOA, no governance tokens** — just a tarp and a hook to keep your stuff dry

## Core Principles

1. **Stable, addressable directories**: Each person/object gets a directory with a stable URL/CID
2. **Static file serving**: Content is served straight off disk with no app server overhead
3. **Filesystem-based**: The city is literally a filesystem tree
4. **Minimal runtime**: The absolute minimum needed to let your data simply live

## Quick Start

### 1. Build the Tools

```bash
go build -o seal ./cmd/seal
go build -o webserver ./cmd/webserver
```

### 2. Create a JSON-LD Document

Create a file `example.jsonld`:

```json
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "My First Article",
  "author": {
    "@type": "Person",
    "name": "Your Name"
  }
}
```

### 3. Seal and Store

```bash
./seal -in example.jsonld -store data
```

This computes a Content Identifier (CID) using URDNA2015 canonicalization.

### 4. Start the Web Server

```bash
export SUPABASE_JWT_SECRET="your-secret"
./webserver -addr :8080 -store data -public public
```

### 5. Access Your Data

Open your browser to `http://localhost:8080` and start managing your JSON-LD objects!

## Features

- **GitHub Authentication**: Secure authentication via Supabase JWT
- **Ownership Tracking**: Every saved object tracks its author
- **ACE Editor**: Syntax highlighting and validation
- **Automatic Permalinks**: Share and load data via URL parameters
- **CID-based Storage**: Content-addressable storage using CIDs

## Next Steps

- Read about [JSON-LD Script Tags](jsonld-script-tag.md)
- Learn about [Authentication](authentication.md)
- Explore [Ethereum Signing](ethereum-signing.md)
- Understand [Canonical JSON](canonical-json.md)

## Philosophy

The "lean-to of sticks" is the absolute minimum runtime that lets something simply live. We provide:

- Each person/object gets a directory
- The directory is static (only files)
- It is addressable (stable URL/CID)
- Served straight off disk — no app server trying to own content

Welcome to your corner of the city!
