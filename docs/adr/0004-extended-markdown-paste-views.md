# Extended Markdown for Paste Views

## Decision

The browser Paste View renders Markdown with Goldmark's built-in GitHub
Flavored Markdown (GFM) extension bundle. The accepted extended dialect is
limited to tables, strikethrough, literal URL and email autolinks, and task
lists.

Task-list markers render as disabled checkboxes. Sanitization permits only the
renderer-generated checkbox shape and does not allow a reader to change task
state. Raw HTML in Paste text is not rendered as HTML.

## Scope

Rendering is a browser presentation concern for `GET /p/{code}`. It runs when a
Paste View is requested, so existing and new pastes use the same dialect. The
Raw Paste bytes, CLI and JSON retrieval, stored representation, HTTP routes,
and expiration behavior are unchanged.

## Exclusions

This decision does not add raw HTML, footnotes, definition lists, emoji
expansion, math or diagram syntax, syntax highlighting, or client-side
interactive task state. Adding any of these features requires a separate
decision because each one changes the rendering or sanitization contract beyond
Goldmark's built-in GFM bundle.
