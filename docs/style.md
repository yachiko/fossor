# Documentation Style Guide

Purpose: Shared conventions for all Fossor docs.

## Voice & Tone

- Concise, task-oriented.
- Assume reader knows `git` fundamentals (branches, remotes, stash, rebase).
- No marketing language. No emoji.

## Headings

| Level    | Use                                       |
| -------- | ----------------------------------------- |
| H1 (`#`)  | Page title (one per file)                 |
| H2 (`##`) | Major sections                            |
| H3 (`###`)| Subsections when necessary (avoid >3 levels) |

## Language Conventions

- Use "manage view" for the four-tab per-repo screen.
- Use "main screen" for the repo table.
- "Repo" and "repository" are interchangeable; prefer "repo" inline, "repository" in titles.
- Refer to keys in backticks: `q`, `Enter`, `Esc`, `Ctrl+C`.
- Numbers: spell out one–nine; use numerals ≥10.

## Code & Examples

- Use fenced code blocks with a language: `bash`, `go`, `yaml`.
- Keep examples minimal; link to reference for exhaustive listings.
- Prefer `./fossor` over `fossor` in examples that follow a local `make build`.

## Keys, Flags, and States

- Always surround keys, flags, and state names with backticks: `--no-fetch`, `Status`, `Up to date`.
- Single canonical keybinding table only in `reference/keybindings.md`. README excerpts are allowed; other docs link.
- Single canonical flag table only in `reference/cli.md`.

## Stability Markers

Add at the top of each page after the title. Legend:

- **Stable** — Backward compatible, monitored.
- **Provisional** — May change; gathering feedback.
- **Experimental** — Likely to change.

## Related Docs Footer

Each page ends with:

```
## See Also
- <link 1>
- <link 2>
```

Minimum 2 links where relevant.

## Admonitions

Emulate with bold prefixes: **Note:**, **Warning:**, **Tip:**.

## Formatting Anti-Patterns

- Wall-of-text paragraphs >6 lines.
- Mixing rationale into reference tables (rationale goes in `explanation/`).
- Repeating identical key/flag tables across pages.
- ASCII art of UI elements (link to the screenshot in the README instead).

## Changelog Awareness

When changing user-visible behavior, update the matching reference page in the same PR and add a `CHANGELOG.md` entry under `## [Unreleased]`.
