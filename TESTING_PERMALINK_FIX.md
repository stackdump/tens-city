# Manual Testing Guide for Permalink Fix

## Issue
When a logged-in user clicks a permalink URL with `?data=...`, the Petri Net data skeleton does not appear in the editor.

## Root Cause
The `_showApp()` method was being called twice due to a race condition:
1. Once from the `onAuthStateChange` callback
2. Once from the `_checkAuth()` async completion

The second call would wipe out the editor (via `innerHTML = ''`) before the permalink data could be loaded, causing the editor to fall back to the default template.

## Fix Applied
Added an `_appShown` flag that:
- Prevents duplicate initialization when `_showApp()` is called multiple times
- Preserves the permalink data loaded during the first call
- Only toggles visibility on subsequent calls
- Resets when showing login screen (to allow re-initialization after logout)

## How to Test

### Prerequisites
1. Start the webserver: `./webserver -addr :8080 -store data -public public`
2. Have a GitHub account set up with the Supabase OAuth app

### Test Case 1: User Already Logged In
1. Open browser and go to `http://127.0.0.1:8080/`
2. Log in with GitHub OAuth
3. **Important**: Keep the tab open and logged in
4. In the same tab (or a new tab with the same session), navigate to:
   ```
   http://127.0.0.1:8080/?data=%257B%2522%2540context%2522%253A%2522https%253A%252F%252Fpflow.xyz%252Fschema%2522%252C%2522%2540type%2522%253A%2522PetriNet%2522%252C%2522%2540version%2522%253A%25221.1%2522%252C%2522arcs%2522%253A%255B%255D%252C%2522places%2522%253A%257B%257D%252C%2522token%2522%253A%255B%2522https%253A%252F%252Fpflow.xyz%252Ftokens%252Fblack%2522%255D%252C%2522transitions%2522%253A%257B%257D%257D
   ```
5. **Expected Result**: The editor should display the PetriNet JSON with:
   - `@context`: "https://pflow.xyz/schema"
   - `@type`: "PetriNet"
   - `@version`: "1.1"
   - `arcs`: []
   - `places`: {}
   - `token`: ["https://pflow.xyz/tokens/black"]
   - `transitions`: {}

### Test Case 2: User Not Logged In
1. Open browser in incognito/private mode or after logging out
2. Navigate to the same permalink URL
3. You should see the login screen
4. Click "Login with GitHub"
5. Complete OAuth flow
6. **Expected Result**: After login redirect, the editor should display the PetriNet JSON (same as above)

### Test Case 3: Browser Console Verification
1. Open Developer Tools Console (F12)
2. Navigate to permalink URL (logged in or not)
3. **Expected Console Messages**:
   - `Permalink: Captured data from URL parameter`
   - `Permalink: Successfully parsed and stored permalink data (in memory and sessionStorage)`
   - (After login) `App already initialized, skipping duplicate _showApp() call` (if already logged in)
   - `Loading data from pending permalink data`
   - `Successfully loaded permalink data into editor`

### Test Case 4: Verify No Duplicate Initialization
1. Set a breakpoint or add a counter in `_showApp()` method
2. Navigate to permalink URL while logged in
3. **Expected**: `_showApp()` should execute fully only once
4. The second call should return early with the message "App already initialized..."

## Decoded Permalink Data

The test URL contains double-encoded JSON. When decoded, it produces:

```json
{
  "@context": "https://pflow.xyz/schema",
  "@type": "PetriNet",
  "@version": "1.1",
  "arcs": [],
  "places": {},
  "token": [
    "https://pflow.xyz/tokens/black"
  ],
  "transitions": {}
}
```

## Additional Test Links

Simple test cases from `test-permalink.html`:

1. **Test 1: Simple Object**
   ```
   http://127.0.0.1:8080/?data=%7B%22%40context%22%3A%22https%3A%2F%2Fpflow.xyz%2Fschema%22%2C%22%40type%22%3A%22TestObject%22%2C%22name%22%3A%22Test%201%22%7D
   ```

2. **Test 2: Petri Net**
   ```
   http://127.0.0.1:8080/?data=%7B%22%40context%22%3A%22https%3A%2F%2Fpflow.xyz%2Fschema%22%2C%22%40type%22%3A%22PetriNet%22%2C%22arcs%22%3A%5B%5D%2C%22name%22%3A%22Example%20Petri%20Net%22%2C%22places%22%3A%7B%7D%2C%22transitions%22%3A%7B%7D%7D
   ```

## Success Criteria

✅ Permalink data loads correctly when user is already logged in
✅ Permalink data loads correctly after completing login flow
✅ No duplicate `_showApp()` initialization
✅ Console logs show proper permalink capture and loading
✅ Editor displays the correct JSON from the URL
✅ All existing Go tests continue to pass
