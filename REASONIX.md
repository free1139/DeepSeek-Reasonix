# Reasonix project memory

This file is loaded into every session's system prompt (the cache-stable prefix),
so keep it concise and durable — it is the project's standing instructions to the
agent. It is the Reasonix agent of Claude Code's CLAUDE.md.

## Conventions

- Go kernel under `internal/`; each package owns one concern and documents it in a
  package comment. Match the surrounding comment density and idiom when editing.
- One transport-agnostic `control.Controller` sits behind every frontend (chat
  TUI, HTTP/SSE serve, Wails desktop). Add behavior to the controller, not a
  frontend, so all three inherit it.
- Cache-first: the system-prompt prefix (base prompt + tools + memory) must stay
  byte-stable across turns so DeepSeek's automatic prefix cache stays warm. Never
  mutate it mid-session — ride the turn tail instead (see `control.Compose`).

## Memory

- Hierarchical docs: `REASONIX.md` (this file, committed/shared), `REASONIX.local.md`
  (personal, git-ignored), user-global `~/.config/reasonix/REASONIX.md`, and any
  `REASONIX.md` in an ancestor dir. `AGENTS.md` is accepted as a fallback name.
- `@path` on its own line imports another file's contents.
- `#<note>` in chat quick-adds a line here. The `remember` tool saves durable
  facts to the per-project auto-memory store (frontmatter files + `MEMORY.md`
  index), which loads into the prefix on the next session.

## Go Files: Always Use gopls for Validation

When working with Go source files (.go):

1. **Detect .go file** → Call `lsp_diagnostics(file)` to check for errors,
   warnings, and misspellings. This is the first action when you plan to
   edit a Go file.

2. **Auto-fix simple issues**: For clear-cut problems like:
   - Misspelled words (e.g., "variabale" → "variable")
   - Unused imports
   - Missing punctuation
   
   Apply fixes directly using `edit_file` or `multi_edit`.

3. **Verify after edits**: Always re-run `lsp_diagnostics(file)` after making
   changes to confirm the fix is correct and no new issues were introduced.

## Notes
