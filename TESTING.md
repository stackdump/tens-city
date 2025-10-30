# Testing Checklist for Tens City

This checklist covers manual testing steps for the new tens-city application.

## Prerequisites
- [ ] Supabase project created
- [ ] GitHub OAuth configured in Supabase
- [ ] Database migrations run
- [ ] Supabase URL and anon key obtained

## Configuration Tests
- [ ] Application shows error message when no configuration provided
- [ ] Error message displays helpful information about required attributes
- [ ] Link to README.md works from error page

## Authentication Tests
- [ ] Login button appears on initial load
- [ ] Clicking "Login with GitHub" redirects to GitHub OAuth
- [ ] After GitHub authorization, user is redirected back to app
- [ ] User email/username is displayed in header
- [ ] Logout button appears when logged in
- [ ] Clicking logout successfully logs out user
- [ ] After logout, login screen reappears

## Load Objects Tests
- [ ] "Load Objects" button appears in toolbar
- [ ] Clicking button fetches objects from database
- [ ] Results display in ACE editor as JSON
- [ ] JSON is properly formatted and syntax highlighted
- [ ] Empty result shows count: 0 when no objects exist
- [ ] Error handling works if database is unreachable

## Post Object Tests
- [ ] ACE editor displays default JSON-LD template on load
- [ ] Editing JSON in ACE editor works smoothly
- [ ] Posting valid JSON-LD succeeds
- [ ] Success message shows generated CID
- [ ] Posted object includes owner_uuid
- [ ] Posted object is visible in database
- [ ] Posting invalid JSON shows error message
- [ ] Posting JSON without @context shows validation error

## Editor Tests
- [ ] ACE editor loads successfully
- [ ] Syntax highlighting works for JSON
- [ ] Code can be edited freely
- [ ] Clear button resets editor to template
- [ ] Editor is responsive and fills available space

## UI/UX Tests
- [ ] Application is responsive on different screen sizes
- [ ] All buttons have hover states
- [ ] All buttons are clearly labeled
- [ ] Toolbar buttons are accessible and functional
- [ ] Header displays correctly with user info

## Security Tests
- [ ] Unauthenticated users cannot access app features
- [ ] Authentication token is managed by Supabase
- [ ] No sensitive credentials are exposed in client code
- [ ] CID generation produces consistent results
- [ ] Row-level security prevents unauthorized access

## Integration Tests
- [ ] CDN resources (Supabase JS, ACE editor) load correctly
- [ ] Network errors are handled gracefully
- [ ] Browser console shows no errors
- [ ] CORS is properly configured for Supabase

## Notes
- Test with both local Supabase (docker-compose) and cloud Supabase
- Test with different browsers (Chrome, Firefox, Safari)
- Test with network throttling to simulate slow connections
- Check browser developer tools for console errors
- Verify database entries match what's shown in UI
