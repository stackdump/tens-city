---
title: "tens-city v0.8: A Federated Blog in 500 Lines"
description: A minimal blog platform with ActivityPub federation. Markdown files, a Go binary, JSON for state. No database required.
datePublished: 2026-01-25T12:00:00Z
author:
  name: stackdump
  type: Person
  url: https://github.com/stackdump
tags:
  - tens-city
  - activitypub
  - federation
  - fediverse
  - go
collection: about
lang: en
slug: tens-city-release
image: https://blog.stackdump.com/images/tens-city-release/federation-flow.svg
draft: false
---

# tens-city v0.8: A Federated Blog in 500 Lines

We released [tens-city v0.8.0](https://github.com/stackdump/tens-city/releases/tag/v0.8.0) this week. It's a minimal blog platform with full ActivityPub federation support. You can follow this blog from Mastodon at `@myork@blog.stackdump.com`.

## The Minimal Stack

Most blog platforms grow into content management systems. They accumulate features: admin panels, user management, plugin systems, database migrations. We went the other direction.

![The Minimal Stack](/images/tens-city-release/stack-architecture.svg)

The entire stack:

| Component | Purpose |
|-----------|---------|
| Markdown files | Content with YAML frontmatter |
| Go binary | Renders markdown, handles HTTP, signs ActivityPub requests |
| JSON files | Followers, published posts, RSA keys |
| nginx | TLS termination, reverse proxy |

That's it. No database server. No background job queue. No cache layer. The webserver reads markdown files from disk and renders them on demand.

## ActivityPub Federation

The interesting part of this release is ActivityPub support. We can now federate with Mastodon, Misskey, Pleroma, and any other ActivityPub-compatible platform.

![ActivityPub Federation](/images/tens-city-release/federation-flow.svg)

### How It Works

**Discovery**: When someone searches for `@myork@blog.stackdump.com` on Mastodon, their server queries our WebFinger endpoint (`/.well-known/webfinger`) to find the actor URL. Then it fetches the actor profile to get the inbox, outbox, and public key.

**Following**: When a user clicks "Follow", their server sends a signed `Follow` activity to our inbox. We verify the HTTP signature, store the follower URL, and send back an `Accept` activity.

**Publishing**: When we publish a new post, we wrap it in a `Create` activity with an `Article` object and POST it to each follower's inbox. The request is signed with our RSA private key so their server can verify it came from us.

### HTTP Signatures

Every ActivityPub request between servers is signed. This prevents spoofing—a malicious server can't pretend to be us because they don't have our private key.

```
Signature: keyId="https://blog.stackdump.com/users/myork#main-key",
           algorithm="rsa-sha256",
           headers="(request-target) host date digest",
           signature="base64..."
```

The receiving server fetches our public key from our actor profile and verifies the signature matches the request body.

## State Without a Database

Where does the follower list go? Where do we track which posts have been federated? Most platforms would reach for PostgreSQL here. We use JSON files.

![State Files](/images/tens-city-release/state-files.svg)

```json
// followers.json
[
  "https://mastodon.social/users/someone",
  "https://hachyderm.io/users/another"
]

// published.json
[
  "https://blog.stackdump.com/posts/tic-tac-toe-model",
  "https://blog.stackdump.com/posts/token-language"
]
```

This scales to thousands of followers and hundreds of posts with no performance issues. JSON parsing is fast. File reads are fast. We don't need transactions or complex queries—just append to a list and write it back.

The tradeoff: we can't efficiently query "who followed after date X" or "which posts got the most boosts". We don't need those features for a personal blog.

## The Publish Workflow

Publishing a post is one command:

![Publish Workflow](/images/tens-city-release/publish-workflow.svg)

```bash
./publish.sh "Add new blog post"
```

This script:

1. Commits changes to git
2. Pushes to GitHub
3. SSHs to the server and pulls
4. Restarts the webserver
5. Calls `/publish` to federate new posts

The federation step is idempotent—it checks `published.json` and only sends posts that haven't been sent before. We can run it repeatedly without spamming followers.

## What We Didn't Build

The interesting design decisions are what we left out:

**No admin panel**: Edit markdown files directly. Use git for version control.

**No media uploads**: Put images in `content/images/` and commit them.

**No comments**: The fediverse is the comment system. Reply to a post on Mastodon.

**No analytics**: We don't track readers. If we wanted analytics, we'd add a lightweight script.

**No scheduled posts**: Write when ready, publish when ready.

**No themes**: The HTML/CSS is in the Go binary. Fork and modify if needed.

Each missing feature is a maintenance burden we don't carry.

## Running Your Own

```bash
# Clone and build
git clone https://github.com/stackdump/tens-city
cd tens-city && make build

# Configure ActivityPub
export ACTIVITYPUB_DOMAIN=blog.example.com
export ACTIVITYPUB_USERNAME=author
export ACTIVITYPUB_PUBLISH_TOKEN=$(openssl rand -base64 32)

# Start server
./webserver -addr :8080 -content content/posts
```

Add markdown files to `content/posts/`, put nginx in front with TLS, and you have a federated blog.

## Philosophy

The name "tens city" evokes tent cities—minimal structures, easily moved, no bureaucracy. The software embodies this: a single binary, files on disk, no dependencies beyond the operating system.

We could add features. User accounts, comment moderation, post scheduling, theme customization. Each feature makes the system harder to understand, harder to maintain, harder to trust.

Instead, we keep it small. The entire ActivityPub implementation is ~500 lines of Go. We can read it, understand it, debug it. When something breaks, we know where to look.

Small models beat large models. This applies to software too.

## Links

- [tens-city on GitHub](https://github.com/stackdump/tens-city)
- [Follow @myork@blog.stackdump.com](https://blog.stackdump.com/users/myork)
- [ActivityPub Specification](https://www.w3.org/TR/activitypub/)
