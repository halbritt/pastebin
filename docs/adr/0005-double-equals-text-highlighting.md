# Double-Equals Text Highlighting

## Decision

The browser Paste View recognizes text enclosed by paired double-equals
delimiters as highlighted text. `==important==` renders as
`<mark>important</mark>`.

The syntax is implemented as a local Goldmark inline extension. It uses
Goldmark's delimiter processing so inline Markdown inside a highlight is parsed
normally. Highlight delimiters inside inline code, fenced code blocks, or
escaped input remain literal. Raw HTML remains disabled.

The existing Bluemonday UGC policy permits attribute-free `mark` elements, so
this decision does not broaden the sanitizer policy. The renderer emits no
attributes on `mark`.

## Scope

This decision extends ADR 0004's browser presentation dialect. Highlighting is
applied only while rendering `GET /p/{code}`. Stored content, Raw Paste bytes,
CLI and JSON retrieval, routes, and expiration behavior are unchanged. Existing
Pastes receive the syntax when rendered.

## Alternatives

Adding another Markdown dependency was rejected because Goldmark already
provides the delimiter and rendering extension points needed for this small
syntax addition.

Preprocessing the source text was rejected because it would have to reproduce
Markdown's code-span, escaping, nesting, and delimiter rules outside the parser.

Code syntax highlighting remains outside this decision.
