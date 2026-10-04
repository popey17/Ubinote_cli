# ubinote CLI Design

**Date:** 2026-10-04  
**Status:** Approved (conversation)  
**Workspace:** `/home/popey/Personal/Project/goTraining/ubinote_cli`  
**API:** `/home/popey/Personal/Project/goTraining/personal_note`  
**Web reference:** `/home/popey/Personal/Project/goTraining/ubinote_web`

## Goal

Build a Go CLI for the ubinote personal-notes API. Notes store Markdown bodies. The CLI must support scriptable commands and an interactive TUI with keyboard/mouse navigation, styled Markdown viewing, and a built-in editor with formatting shortcuts equivalent to the web Markdown toolbar.

## Decisions

| Topic | Choice |
| --- | --- |
| Interaction | Hybrid: `cobra` commands + Bubble Tea TUI |
| Editor | Built-in TUI editor (not `$EDITOR`) |
| Auth | Login once; store JWT under `~/.config/ubinote/` |
| v1 scope | Full CRUD + auth + TUI (parity with web) |
| TUI layout | Master–detail: list left, content right |
| Edit mode | Raw Markdown editor in the right pane (preview after save / exit edit) |
| Stack | Charm: cobra, bubbletea, bubbles, lipgloss, glamour |
| Format UX | Key chords + `ctrl+p` format palette (web toolbar parity) |

## Architecture

**Binary name:** `ubinote`

```
cmd/ubinote/          main entry
internal/api/         HTTP client (auth + notes)
internal/config/      API URL + credential paths
internal/auth/        login/logout/register helpers (CLI-facing)
internal/format/      Markdown format actions (port of web markdownFormat.ts)
internal/tui/         Bubble Tea app (list + view + edit + login)
internal/ui/          shared lipgloss styles / theme
```

### External dependencies (planned)

- `github.com/spf13/cobra`
- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/bubbles`
- `github.com/charmbracelet/lipgloss`
- `github.com/charmbracelet/glamour`
- `github.com/charmbracelet/huh` (optional for login prompts outside TUI)

### API contract (existing)

Base URL default: `http://localhost:8000` (from personal_note `PORT`; overridable via config).

Auth: `Authorization: Bearer <token>` (CLI does not use cookies).

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/api/v1/auth/register` | `{email, password}` → `{id, email}` |
| POST | `/api/v1/auth/login` | `{email, password}` → `{token}` |
| POST | `/api/v1/auth/logout` | no body; CLI also deletes local token |
| GET | `/api/v1/me` | auth required |
| GET/POST | `/api/v1/notes` | list / create `{title, body}` |
| GET/PUT/DELETE | `/api/v1/notes/{id}` | get / update / delete |

Note shape: `id`, `user_id`, `title`, `body`, `created_at`, `updated_at`.

## Commands

| Command | Behavior |
| --- | --- |
| `ubinote login` | Prompt email/password; save JWT |
| `ubinote logout` | Clear local token; call API logout if reachable |
| `ubinote register` | Register then optionally login |
| `ubinote list` | Print note id + title (+ updated_at) |
| `ubinote view <id>` | Render Markdown with glamour to stdout |
| `ubinote create` | Open TUI editor for new note; POST on save |
| `ubinote edit <id>` | Open TUI editor; PUT on save |
| `ubinote delete <id>` | Confirm then DELETE |
| `ubinote` / `ubinote tui` | Launch master–detail TUI |
| `ubinote config set-url <url>` | Persist API base URL |
| `ubinote config show` | Show config + whether a token is stored |

Missing token on protected commands → message to run `ubinote login`. TUI shows an inline login form instead.

## Config & credentials

- `~/.config/ubinote/config.json` — `{ "api_url": "http://localhost:8000" }`
- `~/.config/ubinote/credentials.json` — `{ "token": "..." }` with file mode `0600`

Override env (optional convenience, not required for v1): `UBINOTE_API_URL`.

## TUI design

### Master–detail (browse)

- **Left:** note list (`bubbles/list`), mouse + ↑↓, Enter focuses view
- **Right:** title + glamour-rendered body in a viewport (scroll)
- **Footer:** key help — navigate, `e` edit, `n` new, `d` delete, `q` quit

### Edit mode

- List stays visible (dimmed / unfocused)
- Right pane becomes:
  - Title field
  - Body textarea (`bubbles/textarea`) with Markdown-oriented editing
- `ctrl+s` → save via API → return to view mode with refreshed content
- `esc` → discard unsaved changes (confirm if dirty) → view mode
- No live preview pane in v1; rendering returns after save

### Format toolbar parity

Port web `FormatAction` set into `internal/format`:

`h1`, `h2`, `h3`, `bold`, `italic`, `ul`, `ol`, `indent`, `outdent`, `check`, `quote`, `code`, `codeblock`, `link`

Apply via:

1. **Key chords** while the body editor is focused (v1 defaults):
   - `ctrl+b` bold · `ctrl+i` italic · `ctrl+k` link
   - `alt+1` / `alt+2` / `alt+3` headings
   - `ctrl+shift+c` code block · `alt+c` inline code
   - `alt+u` bullet list · `alt+o` ordered · `alt+t` task
   - `ctrl+]` indent · `ctrl+[` outdent · `alt+q` quote
2. **`ctrl+p` format palette** — fuzzy list of all actions (terminal stand-in for the click toolbar)

If a terminal cannot deliver a chord, the palette remains the fallback for every action.

Selection semantics match the web helper: wrap selection or insert placeholder; line-prefix for headings/lists/quote; indent uses 4 spaces.

### Login form (TUI)

Shown when launching TUI without a valid token: email + password → login → load notes.

## Components & data flow

```
login → save token
list notes → App.notes
select note → GET note → NoteView (glamour)
edit → NoteEditor (local buffer)
format key/palette → format.Apply → update buffer + cursor
ctrl+s → PUT/POST note → refresh list/view
```

**Models**

| Model | Role |
| --- | --- |
| `App` | Root focus, note loading, key routing |
| `NoteList` | List of notes |
| `NoteView` | Viewport + glamour |
| `NoteEditor` | Title + body + format bindings/palette |
| `LoginForm` | Credentials when unauthenticated |

## Error handling

- **401:** clear local token; prompt re-login (command or TUI form)
- **Network failure:** status-line / stderr message; keep editor buffer intact
- **Save failure:** stay in edit mode; show error; do not discard buffer
- **404:** clear message; return to list if viewing deleted note

## Testing

- Unit tests for `internal/format` covering bold, italic, headings, lists, indent/outdent, checklist, quote, code, codeblock, link (aligned with web behavior)
- API client tests with `net/http/httptest`
- Focused TUI tests for format shortcut wiring and save/cancel paths where practical without flaky full-screen integration

## Out of scope (v1)

- Offline / local file sync
- Opening `$EDITOR`
- Cookie-based auth
- Live split preview while editing
- Search/filter beyond what `bubbles/list` provides by default (basic filter OK if free)
- Sharing, tags, folders

## Success criteria

1. User can register/login from CLI and persist a token.
2. User can list/view/create/edit/delete notes via commands and via TUI.
3. View mode styles Markdown distinctly (headings, bold, lists, tasks, quotes, code, links).
4. Edit mode supports web-equivalent format actions via keys and palette.
5. CLI talks only to the existing personal_note HTTP API (no DB access).
