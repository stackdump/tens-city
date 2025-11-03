---
title: Markdown Features Guide
description: Learn about the rich markdown features available in Tens City
datePublished: 2025-11-02T00:00:00Z
dateModified: 2025-11-03T00:00:00Z
author:
  name: Content Team
  type: Organization
  url: https://tens.city
tags:
  - markdown
  - formatting
  - tutorial
collection: guides
lang: en
slug: markdown-features
draft: false
---

# Markdown Features Guide

Tens City supports rich markdown formatting to help you create beautiful, engaging content.

## Text Formatting

You can use **bold**, *italic*, ***bold and italic***, ~~strikethrough~~, and `inline code`.

## Headers

Headers create hierarchy in your documents:

### Level 3 Header
#### Level 4 Header
##### Level 5 Header

## Lists

### Unordered Lists

- First item
- Second item
  - Nested item
  - Another nested item
- Third item

### Ordered Lists

1. First step
2. Second step
   1. Sub-step A
   2. Sub-step B
3. Third step

## Code Blocks

```javascript
// JavaScript example
function greet(name) {
    console.log(`Hello, ${name}!`);
}

greet("World");
```

```python
# Python example
def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

print(fibonacci(10))
```

## Blockquotes

> This is a blockquote. You can use it to highlight important information or quotes from other sources.
>
> Blockquotes can span multiple paragraphs.

## Links and Images

Create links with `[text](url)` syntax:

- [Tens City on GitHub](https://github.com/stackdump/tens-city)
- [Markdown Guide](https://www.markdownguide.org/)

## Tables

| Feature | Support | Notes |
|---------|---------|-------|
| Headers | ✅ | H1-H6 supported |
| Lists | ✅ | Ordered and unordered |
| Code | ✅ | Inline and blocks |
| Tables | ✅ | Full support |

## Horizontal Rules

You can add horizontal rules to separate content:

---

## Task Lists

- [x] Support markdown features
- [x] Generate JSON-LD automatically
- [x] Serve content quickly
- [ ] Add more themes (coming soon!)

## Emphasis and Strong Emphasis

Use *asterisks* or _underscores_ for emphasis, and **double asterisks** or __double underscores__ for strong emphasis.

## Conclusion

These markdown features give you powerful tools to create rich, engaging blog posts. Mix and match them to create content that's both readable and visually appealing.

Happy writing! 📝
