# Permalink Fix - Testing Guide

## Problem
After adding authentication to the application, the permalink feature (`?data=...` URLs) stopped working. When users clicked a permalink:
1. The page would redirect to login screen
2. After logging in, the JSON data from the URL was lost
3. The editor would show default template instead of the permalink data

## Solution
The fix captures permalink data early in the page lifecycle, before authentication checks, and preserves it during the login flow.

## How to Test

### Test 1: Permalink with Authentication
1. Start the webserver: `./webserver -addr :8080 -store data -public public`
2. Visit: http://localhost:8080/test-permalink.html
3. Click on one of the test permalink links
4. If not logged in, you'll see the login screen
5. After logging in via GitHub OAuth, the JSON data from the URL should load into the editor
6. Check browser console for logging messages showing the permalink capture and loading

### Test 2: Verify Console Logging
Open browser console and look for these log messages:

**When page loads with `?data=...`:**
- `Permalink: Captured data from URL parameter`
- `Permalink: Successfully parsed and stored permalink data`

**When loading data into editor:**
- `Loading initial data...`
- `Loading data from pending permalink data`
- `Successfully loaded permalink data into editor`

**When auto-save happens (for authenticated users):**
- `Auto-save: Checking if auto-save should run...`
- `Auto-save: Attempting to auto-save JSON-LD data`
- `Auto-save: Sending save request to /api/save`
- `Auto-save: Success! CID: ... - Redirecting to ?cid= URL`

**When permalink anchor updates:**
- `Permalink: Updated permalink anchor with current editor content`

### Test 3: Permalink URL Format
Example permalink URL:
```
http://localhost:8080/?data=%7B%22%40context%22%3A%22https%3A%2F%2Fpflow.xyz%2Fschema%22%2C%22%40type%22%3A%22TestObject%22%2C%22name%22%3A%22Test%201%22%7D
```

Decoded data:
```json
{"@context":"https://pflow.xyz/schema","@type":"TestObject","name":"Test 1"}
```

## Implementation Details

### Key Changes
1. **Early Capture**: `_capturePermalinkData()` is called in `connectedCallback()` before authentication
2. **Persistent Storage**: Data is stored in `this._pendingPermalinkData` field
3. **Priority Loading**: In `_loadInitialData()`, pending data is checked before URL parsing
4. **Enhanced Logging**: All permalink and auto-save operations log their status

### Code Flow
```
Page Load
  → connectedCallback()
  → _capturePermalinkData()  [Captures ?data=... from URL]
  → _initSupabase()
  → _checkAuth()
    → _showLogin() [if not authenticated]
    → User logs in
    → _showApp() [after authentication]
      → _loadInitialData()
        → Check _pendingPermalinkData [Uses captured data!]
        → Load into editor
        → _autoSaveFromURL() [if authenticated]
```

## Logging Prefixes
- `Permalink:` - Operations related to permalink capture and anchor updates
- `Auto-save:` - Operations related to automatic saving of permalink data
- Generic messages - Data loading and initialization

## Testing Checklist
- [x] Permalink data is captured before authentication
- [x] Data persists through login flow
- [x] Editor loads data after successful login
- [x] Auto-save works for authenticated users
- [x] Permalink anchor updates correctly
- [x] Console logging provides clear debugging information
- [x] All existing tests pass
