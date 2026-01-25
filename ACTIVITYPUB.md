# ActivityPub Federation

tens-city supports ActivityPub federation, allowing your blog to be followed from Mastodon, Misskey, Pleroma, and other fediverse platforms.

## Features

- **Discovery**: WebFinger, NodeInfo, Actor profile
- **Follow/Unfollow**: Accept follow requests, track followers
- **Publish**: Push new posts to all followers' timelines
- **Article format**: Blog posts are published as ActivityPub Article objects

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ACTIVITYPUB_DOMAIN` | Yes | Your blog's domain (e.g., `blog.example.com`) |
| `ACTIVITYPUB_USERNAME` | Yes | Fediverse username (e.g., `myork`) |
| `ACTIVITYPUB_DISPLAY_NAME` | No | Display name shown on profiles |
| `ACTIVITYPUB_SUMMARY` | No | Bio/description for the actor profile |
| `ACTIVITYPUB_PROFILE_URL` | No | URL to your profile page |
| `ACTIVITYPUB_ICON_URL` | No | URL to avatar image |
| `ACTIVITYPUB_KEY_PATH` | No | Path to RSA private key (default: `data/activitypub.key`) |
| `ACTIVITYPUB_PUBLISH_TOKEN` | No | Secret token for the publish endpoint |

### Example Setup

```bash
export ACTIVITYPUB_DOMAIN=blog.example.com
export ACTIVITYPUB_USERNAME=blogger
export ACTIVITYPUB_DISPLAY_NAME="My Blog"
export ACTIVITYPUB_SUMMARY="Writing about interesting things"
export ACTIVITYPUB_KEY_PATH=data/activitypub.key
export ACTIVITYPUB_PUBLISH_TOKEN=$(head -c 32 /dev/urandom | base64 | tr -d '/+=' | head -c 32)

./webserver
```

On first run, an RSA keypair is automatically generated at `ACTIVITYPUB_KEY_PATH`.

## Endpoints

### Discovery

| Endpoint | Description |
|----------|-------------|
| `/.well-known/webfinger?resource=acct:user@domain` | WebFinger discovery |
| `/.well-known/nodeinfo` | NodeInfo discovery |
| `/nodeinfo/2.0` | NodeInfo 2.0 metadata |
| `/users/{username}` | Actor profile (JSON-LD) |

### Collections

| Endpoint | Description |
|----------|-------------|
| `/users/{username}/outbox` | Published posts (Article activities) |
| `/users/{username}/followers` | Followers collection |
| `/users/{username}/following` | Following collection (empty) |

### Inbox

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/users/{username}/inbox` | POST | Receives activities (Follow, Undo, etc.) |

### Publishing

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/publish?token=<secret>` | POST | Push new posts to followers |
| `/publish?token=<secret>&slug=<post-slug>` | POST | Push specific post |

## Usage

### Following from Mastodon

Search for `@username@yourdomain.com` in Mastodon's search box, then click Follow.

### Publishing Posts

When you add a new blog post to `content/posts/`, call the publish endpoint:

```bash
curl -X POST "https://yourdomain.com/publish?token=YOUR_TOKEN"
```

Response:
```json
{
  "message": "Publish complete",
  "published": 1,
  "total": 7
}
```

Posts are tracked in `data/published.json` to avoid duplicate delivery.

### Publish a Specific Post

```bash
curl -X POST "https://yourdomain.com/publish?token=YOUR_TOKEN&slug=my-new-post"
```

## Data Files

All ActivityPub state is stored in the data directory:

| File | Description |
|------|-------------|
| `activitypub.key` | RSA private key for signing requests |
| `followers.json` | List of follower actor URLs |
| `published.json` | List of published post IDs |

## Legacy Compatibility

For blogs migrating from WriteFreely/Write.as, legacy endpoints are supported:

- `/api/collections/{username}` → Actor profile
- `/api/collections/{username}/inbox` → Inbox

## How It Works

### Follow Flow

1. Remote user searches for `@user@domain`
2. Their server fetches `/.well-known/webfinger`
3. Their server fetches `/users/{username}` (Actor profile)
4. Their server POSTs a Follow activity to `/users/{username}/inbox`
5. tens-city saves the follower and sends an Accept activity back
6. Follow is complete

### Publish Flow

1. You add a new post to `content/posts/`
2. You call `POST /publish?token=...`
3. tens-city creates a Create activity with the Article
4. tens-city fetches each follower's inbox URL
5. tens-city POSTs the signed activity to each inbox
6. Post appears in followers' timelines

## HTTP Signatures

All outgoing requests are signed using HTTP Signatures (draft-cavage-http-signatures) with:

- Algorithm: `rsa-sha256`
- Headers signed: `(request-target)`, `host`, `date`, `digest` (POST), `content-type` (POST)

## Disabling Federation

To run without ActivityPub:

```bash
./webserver --no-federation
```
