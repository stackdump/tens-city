# JSON-LD Script Tag and Permalink Feature

## Overview

The tens-city web component now supports loading and embedding JSON-LD data in multiple ways:

1. **Script Tag Embedding**: Load data from `<script type="application/ld+json">` tags
2. **Auto-Update**: Automatically update script tags when editor content changes
3. **Permalink Creation**: Share data via URL parameters

## Features

### 1. Load from Script Tag

You can embed JSON-LD data directly in the HTML by placing a script tag inside the `<tens-city>` element:

```html
<tens-city>
    <script type="application/ld+json">
    {
      "@context": "https://pflow.xyz/schema",
      "@type": "Object",
      "name": "My Data"
    }
    </script>
</tens-city>
```

When the page loads, this data will automatically appear in the ACE editor.

### 2. Auto-Update Script Tag

When you edit JSON in the ACE editor, the script tag is automatically updated with the current content. This happens on every change, but only if the JSON is valid.

Benefits:
- The page DOM always reflects the current editor state
- External scripts can access the current data via `document.querySelector('script[type="application/ld+json"]')`
- Data persists in the page structure

### 3. Permalink Anchor

The "🔗 Permalink" link in the toolbar is automatically updated as you edit:
- The anchor's href is updated in real-time with the current editor content
- Click the link to open a new tab/window with the current data encoded in the URL
- The permalink anchor always reflects the latest valid JSON in the editor
- Share the URL to allow others to load the exact same data

Example permalink:
```
http://localhost:8080/index.html?data=%7B%22%40context%22%3A...
```

When someone visits this URL, the data is automatically loaded into the editor and added as a script tag.

## Priority Loading

The system checks for data in this order:

1. **URL Parameter**: If `?data=...` is present in URL, use that
2. **Script Tag**: If a `<script type="application/ld+json">` tag exists, use that
3. **Database**: Otherwise, load recent objects from the database

## Usage Examples

### Example 1: Embedded Data

```html
<!DOCTYPE html>
<html>
<head>
    <script src="tens-city.js" type="module"></script>
</head>
<body>
    <tens-city>
        <script type="application/ld+json">
        {
          "@context": "http://schema.org/",
          "@type": "Person",
          "name": "Jane Doe"
        }
        </script>
    </tens-city>
</body>
</html>
```

### Example 2: URL Permalink

Share this URL to load specific data:
```
https://tens.city/?data=%7B%22%40context%22%3A%22http%3A%2F%2Fschema.org%2F%22%2C%22%40type%22%3A%22Person%22%2C%22name%22%3A%22Jane%20Doe%22%7D
```

### Example 3: Accessing Data from JavaScript

```javascript
// Get the current JSON-LD data from the page
const scriptTag = document.querySelector('tens-city script[type="application/ld+json"]');
if (scriptTag) {
    const data = JSON.parse(scriptTag.textContent);
    console.log('Current data:', data);
}
```

## Testing

See `test-jsonld.html` for a complete example with embedded data and testing instructions.

## Implementation Details

### Methods Added

- `_loadFromScriptTag()`: Reads JSON-LD from embedded script tag
- `_updateScriptTag()`: Updates/creates script tag with current editor content
- `_loadFromURL()`: Decodes data from URL parameter
- `_updatePermalinkAnchor()`: Updates permalink anchor href with current editor content

### Security Considerations

- JSON is validated before parsing
- Invalid JSON is gracefully ignored
- URL encoding prevents injection attacks
- Script tags are properly escaped
