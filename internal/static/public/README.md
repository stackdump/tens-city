# Tens City - JSON-LD Object Manager

A single-page application for managing JSON-LD objects with GitHub authentication and Supabase backend.

## Features

- **GitHub OAuth Login**: Secure authentication via GitHub
- **Object Management**: Query, view, and post JSON-LD objects
- **ACE Editor**: Syntax-highlighted JSON editing
- **Ownership Tracking**: All posted objects are tracked by owner
- **CID Generation**: Automatic content-addressed identifiers for objects

## Setup

### 1. Configure Supabase

You need a Supabase project with:

1. **GitHub OAuth enabled** in Authentication > Providers
2. **Database tables** created (see migrations in `/migrations`)
3. **Row Level Security (RLS)** policies enabled

### 2. Add Supabase Configuration

Update `index.html` to include your Supabase credentials as HTML attributes:

```html
<tens-city 
  supabase-url="https://your-project.supabase.co" 
  supabase-key="your-anon-key">
</tens-city>
```

You can also copy `index.example.html` as a starting template.

**Note**: The default configuration in the code uses local Supabase development server settings. For production, always configure via HTML attributes.

### 3. Configure GitHub OAuth

In your Supabase project settings:

1. Go to Authentication > Providers > GitHub
2. Enable GitHub provider
3. Add your GitHub OAuth app credentials
4. Set the callback URL to: `https://your-project.supabase.co/auth/v1/callback`
5. In your GitHub OAuth app settings, set the Authorization callback URL to the same

### 4. Serve the Application

Simply serve the `public` directory with any static file server:

```bash
# Using Python
cd public
python3 -m http.server 8080

# Using Node.js http-server
npx http-server public -p 8080

# Using PHP
cd public
php -S localhost:8080
```

Then open http://localhost:8080 in your browser.

## Usage

### Login
1. Click "Login with GitHub" button
2. Authorize the application via GitHub OAuth
3. You'll be redirected back to the application

### Load Objects
- Click "📋 Load Objects" to fetch the 10 most recent objects from the database
- Results are displayed in JSON format in the editor

### Post Object
1. Edit the JSON in the editor to create a valid JSON-LD object
2. Ensure it has a `@context` field (required for JSON-LD)
3. Click "📤 Post Object" to save it to the database
4. A CID (Content Identifier) will be automatically generated

### Clear Editor
- Click "🗑️ Clear" to reset the editor to a blank JSON-LD template

## JSON-LD Format

All objects must be valid JSON-LD with at minimum:

```json
{
  "@context": "https://pflow.xyz/schema",
  "@type": "YourType",
  "your": "data"
}
```

## Database Schema

The application uses the following table structure:

```sql
CREATE TABLE public.objects (
  cid text PRIMARY KEY,
  owner_uuid uuid NOT NULL,
  raw jsonb NOT NULL,
  canonical text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
```

See `/migrations` for complete schema and RLS policies.

## Security Notes

- All database operations require authentication
- Row Level Security (RLS) ensures users can only modify their own objects
- All users can read all objects (public read)
- OAuth tokens are managed by Supabase Auth
- Never commit your Supabase anon key to public repositories (use environment variables or config files in production)

## Development

The application is built as a Web Component using:
- Custom Elements API
- ES6 Modules
- Supabase JS Client (via CDN)
- ACE Editor (via CDN)

No build step is required - it runs directly in modern browsers.

## Troubleshooting

**"Failed to load Supabase"**: Check that your browser can access cdn.jsdelivr.net and that your Supabase URL/key are correct.

**"Login failed"**: Verify GitHub OAuth is properly configured in Supabase and your GitHub OAuth app.

**"Failed to post object"**: Ensure the JSON is valid JSON-LD with a `@context` field, and that database migrations have been run.

**CORS errors**: Make sure your site URL is added to Supabase's allowed origins in Authentication > URL Configuration.
