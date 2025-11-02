---
title: Authentication with GitHub
description: Understanding authentication and author tracking in Tens City
datePublished: 2025-11-02T00:00:00Z
author:
  name: Tens City Team
  type: Organization
  url: https://tens.city
tags:
  - authentication
  - security
  - github
collection: guides
lang: en
slug: authentication
---

# Authentication and Author Information

## Overview

The `/api/save` endpoint requires authentication using Supabase JWT tokens obtained via GitHub OAuth. When a user saves a JSON-LD object, their GitHub identity is automatically recorded.

## How It Works

1. **GitHub OAuth**: Users authenticate via GitHub through Supabase
2. **JWT Token**: Supabase issues a JWT token containing user information
3. **Server Validation**: The server validates the token using the `SUPABASE_JWT_SECRET`
4. **Author Tracking**: GitHub username and ID are stored with each saved object

## Security Features

- **Cryptographic JWT verification** using Supabase JWT secret
- **Server-side validation** of JSON-LD structure
- **Content size limits** to prevent abuse
- **Author verification** for deletions - only the author can delete their objects

## Setting Up

1. Set the Supabase JWT secret:
   ```bash
   export SUPABASE_JWT_SECRET="your-supabase-jwt-secret"
   ```

2. Start the server:
   ```bash
   ./webserver -addr :8080 -store data -public public
   ```

## Ownership

Objects are tied to their creators:
- Only the author can delete an object
- Author information is publicly visible in the object metadata
- GitHub ID provides stronger verification than username alone
