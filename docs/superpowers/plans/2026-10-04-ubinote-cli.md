# ubinote CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a hybrid Go CLI (`ubinote`) that talks to the personal_note API with scriptable commands and a Charm TUI for browsing/editing Markdown notes.

**Architecture:** `cobra` commands wrap a shared HTTP client and config/credentials store. The interactive app is a Bubble Tea master–detail UI (list + glamour view + raw editor). Markdown format actions are a pure Go port of the web `markdownFormat.ts` helper, wired to editor key chords and a `ctrl+p` palette.

**Tech Stack:** Go 1.27, cobra, bubbletea, bubbles, lipgloss, glamour, net/http, encoding/json

## Global Constraints

- Module path: `ubinote-cli` (Go 1.27)
- Binary name: `ubinote`
- Default API URL: `http://localhost:8000`
- Auth header: `Authorization: Bearer <token>` (no cookies)
- Credentials file mode: `0600` at `~/.config/ubinote/credentials.json`
- Config at `~/.config/ubinote/config.json`; env override `UBINOTE_API_URL`
- No database access; HTTP API only
- No `$EDITOR` integration in v1
- No live split preview in v1

## File Structure

| Path | Responsibility |
| --- | --- |
| `cmd/ubinote/main.go` | Entrypoint; calls `cli.Execute()` |
| `internal/config/config.go` | Load/save config + credentials paths |
| `internal/config/config_test.go` | Config/credentials tests |
| `internal/api/types.go` | Note, auth request/response types |
| `internal/api/client.go` | HTTP client methods |
| `internal/api/client_test.go` | httptest client tests |
| `internal/api/errors.go` | Typed API errors (`ErrUnauthorized`, etc.) |
| `internal/format/format.go` | `Apply(action, value, start, end)` |
| `internal/format/format_test.go` | Format action unit tests |
| `internal/ui/styles.go` | Shared lipgloss styles |
| `internal/tui/app.go` | Root Bubble Tea model |
| `internal/tui/login.go` | Login form model |
| `internal/tui/editor.go` | Title + textarea editor + save/cancel |
| `internal/tui/format_keys.go` | Key chord → format action map |
| `internal/tui/palette.go` | `ctrl+p` format action list |
| `internal/tui/keys_test.go` | Format key map tests |
| `internal/cli/root.go` | Cobra root + dependency wiring |
| `internal/cli/auth.go` | login/logout/register |
| `internal/cli/notes.go` | list/view/create/edit/delete |
| `internal/cli/config.go` | config set-url / show |
| `internal/cli/tui.go` | `tui` command + default run |
| `README.md` | Setup and usage |

---

### Task 1: Module scaffold + config/credentials

**Files:**
- Create: `go.mod`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `cmd/ubinote/main.go` (stub)

**Interfaces:**
- Produces:
  - `config.Config` with field `APIURL string`
  - `config.Credentials` with field `Token string`
  - `func Dir() (string, error)` → `~/.config/ubinote`
  - `func Load() (Config, error)`
  - `func Save(Config) error`
  - `func LoadCredentials() (Credentials, error)`
  - `func SaveCredentials(Credentials) error`
  - `func ClearCredentials() error`
  - `func DefaultAPIURL() string` → `"http://localhost:8000"`
  - `UBINOTE_API_URL` overrides `APIURL` when set after load

- [ ] **Step 1: Init module**

```bash
cd /home/popey/Personal/Project/goTraining/ubinote_cli
go mod init ubinote-cli
```

Expected: `go.mod` with `module ubinote-cli` and `go 1.27`

- [ ] **Step 2: Write failing config tests**

```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"ubinote-cli/internal/config"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("UBINOTE_API_URL", "")

	cfg := config.Config{APIURL: "http://example.com:9000"}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.APIURL != "http://example.com:9000" {
		t.Fatalf("got %q", got.APIURL)
	}
}

func TestEnvOverridesAPIURL(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("UBINOTE_API_URL", "http://override:1")
	_ = config.Save(config.Config{APIURL: "http://file:2"})
	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.APIURL != "http://override:1" {
		t.Fatalf("got %q", got.APIURL)
	}
}

func TestCredentialsMode0600(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.SaveCredentials(config.Credentials{Token: "abc"}); err != nil {
		t.Fatal(err)
	}
	dir, _ := config.Dir()
	info, err := os.Stat(filepath.Join(dir, "credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
}
```

- [ ] **Step 3: Run tests — expect FAIL**

```bash
go test ./internal/config/ -v
```

Expected: FAIL (package/types missing)

- [ ] **Step 4: Implement config**

```go
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const defaultAPIURL = "http://localhost:8000"

type Config struct {
	APIURL string `json:"api_url"`
}

type Credentials struct {
	Token string `json:"token"`
}

func DefaultAPIURL() string { return defaultAPIURL }

func Dir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "ubinote"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "ubinote"), nil
}

func Load() (Config, error) {
	cfg := Config{APIURL: defaultAPIURL}
	dir, err := Dir()
	if err != nil {
		return cfg, err
	}
	b, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if v := os.Getenv("UBINOTE_API_URL"); v != "" {
				cfg.APIURL = v
			}
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.APIURL == "" {
		cfg.APIURL = defaultAPIURL
	}
	if v := os.Getenv("UBINOTE_API_URL"); v != "" {
		cfg.APIURL = v
	}
	return cfg, nil
}

func Save(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), b, 0o644)
}

func LoadCredentials() (Credentials, error) {
	var c Credentials
	dir, err := Dir()
	if err != nil {
		return c, err
	}
	b, err := os.ReadFile(filepath.Join(dir, "credentials.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c, nil
		}
		return c, err
	}
	return c, json.Unmarshal(b, &c)
}

func SaveCredentials(c Credentials) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "credentials.json")
	return os.WriteFile(path, b, 0o600)
}

func ClearCredentials() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(dir, "credentials.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
```

Stub main:

```go
package main

import "fmt"

func main() {
	fmt.Println("ubinote")
}
```

- [ ] **Step 5: Run tests — expect PASS**

```bash
go test ./internal/config/ -v
```

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum cmd/ubinote/main.go internal/config/
git commit -m "feat: add config and credentials storage"
```

---

### Task 2: Markdown format package (web toolbar parity)

**Files:**
- Create: `internal/format/format.go`
- Create: `internal/format/format_test.go`

**Interfaces:**
- Produces:
  - `type Action string` constants: `Bold`, `Italic`, `H1`, `H2`, `H3`, `UL`, `OL`, `Check`, `Quote`, `Code`, `CodeBlock`, `Link`, `Indent`, `Outdent`
  - `type Result struct { Value string; SelectionStart, SelectionEnd int }`
  - `func Apply(action Action, value string, start, end int) Result`
- Consumes: nothing

Behavior must match `ubinote_web/src/lib/markdownFormat.ts` (4-space indent, placeholders, strip list markers).

- [ ] **Step 1: Write failing tests**

```go
package format_test

import (
	"testing"

	"ubinote-cli/internal/format"
)

func TestBoldWrapsSelection(t *testing.T) {
	r := format.Apply(format.Bold, "hello world", 6, 11)
	if r.Value != "hello **world**" {
		t.Fatalf("got %q", r.Value)
	}
	if r.SelectionStart != 8 || r.SelectionEnd != 13 {
		t.Fatalf("selection %d-%d", r.SelectionStart, r.SelectionEnd)
	}
}

func TestBoldPlaceholder(t *testing.T) {
	r := format.Apply(format.Bold, "", 0, 0)
	if r.Value != "**bold text**" {
		t.Fatalf("got %q", r.Value)
	}
}

func TestHeadingPrefixesLine(t *testing.T) {
	r := format.Apply(format.H2, "Title", 0, 5)
	if r.Value != "## Title" {
		t.Fatalf("got %q", r.Value)
	}
}

func TestULAndIndent(t *testing.T) {
	r := format.Apply(format.UL, "item", 0, 4)
	if r.Value != "- item" {
		t.Fatalf("got %q", r.Value)
	}
	r = format.Apply(format.Indent, r.Value, 0, len(r.Value))
	if r.Value != "    - item" {
		t.Fatalf("got %q", r.Value)
	}
}

func TestCheckAndCodeBlock(t *testing.T) {
	r := format.Apply(format.Check, "task", 0, 4)
	if r.Value != "- [ ] task" {
		t.Fatalf("got %q", r.Value)
	}
	r = format.Apply(format.CodeBlock, "x", 0, 1)
	if r.Value != "\n```\nx\n```\n" {
		t.Fatalf("got %q", r.Value)
	}
}

func TestLink(t *testing.T) {
	r := format.Apply(format.Link, "docs", 0, 4)
	if r.Value != "[docs](https://)" {
		t.Fatalf("got %q", r.Value)
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
go test ./internal/format/ -v
```

- [ ] **Step 3: Implement `Apply`**

Port logic from `/home/popey/Personal/Project/goTraining/ubinote_web/src/lib/markdownFormat.ts` into Go:

- `wrapInline` for bold/italic/code/link
- `prefixLines` for h1–h3, ul, ol, check, quote
- `changeIndent` with `INDENT = "    "`
- `codeblock` wraps selection in fenced block with leading/trailing newlines
- Export `AllActions() []Action` and `ActionLabel(Action) string` for the palette

Keep helpers unexported in the same file. Full implementation should be a direct translation of the TypeScript file (no behavior drift).

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test ./internal/format/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/format/
git commit -m "feat: port markdown format actions from web toolbar"
```

---

### Task 3: HTTP API client

**Files:**
- Create: `internal/api/types.go`
- Create: `internal/api/errors.go`
- Create: `internal/api/client.go`
- Create: `internal/api/client_test.go`

**Interfaces:**
- Produces:
  - `type Note struct { ID, UserID, Title, Body string; CreatedAt, UpdatedAt time.Time }` with JSON tags matching API (`user_id`, `created_at`, `updated_at`)
  - `type Client struct` constructed by `New(baseURL, token string) *Client`
  - `func (c *Client) SetToken(token string)`
  - `Register(ctx, email, password) (id, email string, err error)`
  - `Login(ctx, email, password) (token string, err error)`
  - `Logout(ctx) error`
  - `Me(ctx) (userID string, err error)`
  - `ListNotes(ctx) ([]Note, error)`
  - `GetNote(ctx, id string) (Note, error)`
  - `CreateNote(ctx, title, body string) (Note, error)`
  - `UpdateNote(ctx, id, title, body string) (Note, error)`
  - `DeleteNote(ctx, id string) error`
  - `var ErrUnauthorized, ErrNotFound error` (sentinel; wrap with `%w`)
- Consumes: none

- [ ] **Step 1: Write failing client tests with httptest**

```go
package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ubinote-cli/internal/api"
)

func TestLoginAndAuthHeader(t *testing.T) {
	var sawAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "tok123"})
		case "/api/v1/notes":
			sawAuth = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode([]any{})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := api.New(srv.URL, "")
	tok, err := c.Login(context.Background(), "a@b.c", "secret")
	if err != nil || tok != "tok123" {
		t.Fatalf("login: %q %v", tok, err)
	}
	c.SetToken(tok)
	if _, err := c.ListNotes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sawAuth != "Bearer tok123" {
		t.Fatalf("auth %q", sawAuth)
	}
}

func TestUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := api.New(srv.URL, "bad")
	_, err := c.ListNotes(context.Background())
	if err == nil || !errors.Is(err, api.ErrUnauthorized) {
		t.Fatalf("err=%v", err)
	}
}
```

Add `errors` import. Also cover `CreateNote` 201 and `GetNote` 404 → `ErrNotFound`.

- [ ] **Step 2: Run tests — expect FAIL**

```bash
go test ./internal/api/ -v
```

- [ ] **Step 3: Implement client**

- Trim trailing `/` from baseURL
- JSON encode/decode; treat empty body on 204 as success
- On status 401 return `fmt.Errorf("%w: %s", ErrUnauthorized, body)`
- On status 404 return `fmt.Errorf("%w: %s", ErrNotFound, body)`
- Other non-2xx: `fmt.Errorf("api %s: %s", status, body)`

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test ./internal/api/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/api/
git commit -m "feat: add personal_note HTTP API client"
```

---

### Task 4: Cobra CLI — auth + config commands

**Files:**
- Create: `internal/cli/root.go`
- Create: `internal/cli/auth.go`
- Create: `internal/cli/config.go`
- Modify: `cmd/ubinote/main.go`
- Dependencies: `go get github.com/spf13/cobra`

**Interfaces:**
- Consumes: `config.*`, `api.New`, `api.Client.Login/Register/Logout`
- Produces: `func Execute() error` wiring root command

- [ ] **Step 1: Add cobra and implement root**

`root.go` loads config + credentials, builds `*api.Client`, attaches subcommands.

`main.go`:

```go
package main

import (
	"os"
	"ubinote-cli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Implement auth commands**

- `ubinote login`: read email/password from stdin prompts (`fmt.Scanln` or `golang.org/x/term` for password); call `Login`; `SaveCredentials`
- `ubinote register`: prompt; `Register`; print success; optionally auto-login (do auto-login)
- `ubinote logout`: `ClearCredentials`; best-effort `Logout` API call

Print clear errors to stderr; return error from `RunE`.

- [ ] **Step 3: Implement config commands**

- `ubinote config show`: print API URL + whether token is non-empty
- `ubinote config set-url <url>`: save config

- [ ] **Step 4: Manual smoke**

```bash
go run ./cmd/ubinote --help
go run ./cmd/ubinote config show
```

Expected: help lists login/register/logout/config; show prints default URL and no token.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum cmd/ubinote/main.go internal/cli/
git commit -m "feat: add auth and config CLI commands"
```

---

### Task 5: Cobra CLI — note commands (list/view/delete)

**Files:**
- Create: `internal/cli/notes.go`
- Dependencies: `go get github.com/charmbracelet/glamour`

**Interfaces:**
- Consumes: `api.Client` note methods, credentials token
- Produces: `list`, `view`, `delete` commands

- [ ] **Step 1: Require auth helper**

```go
func requireClient() (*api.Client, error) {
	cfg, err := config.Load()
	// ...
	creds, err := config.LoadCredentials()
	if creds.Token == "" {
		return nil, fmt.Errorf("not logged in; run: ubinote login")
	}
	return api.New(cfg.APIURL, creds.Token), nil
}
```

On `ErrUnauthorized`, clear credentials and return “session expired; run ubinote login”.

- [ ] **Step 2: Implement list/view/delete**

- `list`: table-ish lines `id\ttitle\tupdated_at`
- `view <id>`: glamour render of `note.Body` with title header; width from terminal if available else 80
- `delete <id>`: prompt `Delete note <title>? [y/N]`; on yes call `DeleteNote`

- [ ] **Step 3: Build**

```bash
go build -o bin/ubinote ./cmd/ubinote
./bin/ubinote list
```

Expected without token: error message about login (non-zero exit).

- [ ] **Step 4: Commit**

```bash
git add internal/cli/notes.go go.mod go.sum
git commit -m "feat: add list, view, and delete note commands"
```

---

### Task 6: UI styles + TUI login + master–detail shell

**Files:**
- Create: `internal/ui/styles.go`
- Create: `internal/tui/app.go`
- Create: `internal/tui/login.go`
- Create: `internal/cli/tui.go`
- Dependencies: `go get github.com/charmbracelet/bubbletea github.com/charmbracelet/bubbles github.com/charmbracelet/lipgloss github.com/charmbracelet/glamour`

**Interfaces:**
- Consumes: `*api.Client`, `config.SaveCredentials`
- Produces: `tui.Run(client *api.Client) error` — full-screen app
- Root with no args runs TUI; also `ubinote tui`

- [ ] **Step 1: Styles**

Define lipgloss styles: title, dim, selected list item, border, status, error. Dark Tokyo-night–like palette consistent with brainstorm mockups (`#7aa2f7` accent, `#c0caf5` text, `#1a1b26` bg awareness via lipgloss adaptive colors where helpful).

- [ ] **Step 2: Login model**

Fields: email, password (masked), focus index, errMsg. Enter on password → `client.Login` → `SaveCredentials` → `client.SetToken` → transition to main app. Esc/q quits.

- [ ] **Step 3: App model browse shell**

- Left: `bubbles/list` of note titles (custom item with ID)
- Right: empty state or selected note title + glamour viewport
- On start: if `client` token empty → show login; else `ListNotes` and select first
- Keys: `q` quit, `j/k` or arrows via list, mouse enabled (`tea.WithMouseCellMotion()`)
- Selecting item → `GetNote` → render body into viewport
- Footer help string

- [ ] **Step 4: Wire `tui` command + default**

```go
// root RunE when no subcommand: return tui.Run(clientOrEmpty)
```

If token missing, TUI still starts at login form (pass client with empty token + base URL from config).

- [ ] **Step 5: Manual smoke**

```bash
go run ./cmd/ubinote tui
```

Expected: login form or note list if already logged in.

- [ ] **Step 6: Commit**

```bash
git add internal/ui/ internal/tui/ internal/cli/tui.go internal/cli/root.go go.mod go.sum
git commit -m "feat: add TUI login and master-detail browse shell"
```

---

### Task 7: TUI editor + create/edit/delete in TUI

**Files:**
- Create: `internal/tui/editor.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/cli/notes.go` (create/edit launch editor-only mode)

**Interfaces:**
- Produces:
  - `type EditorDone struct { Title, Body string; Saved bool }`
  - Editor as a mode inside `App` OR `tui.RunEditor(client, id string, title, body string) (EditorDone, error)` for CLI create/edit
- Keys: `e` edit selected, `n` new note, `d` delete with confirm, `ctrl+s` save, `esc` cancel (confirm if dirty)

- [ ] **Step 1: Implement editor model**

- Title `textinput.Model`
- Body `textarea.Model` (soft wrap, high height)
- Track `dirty`, `saving`, `errMsg`, `isNew`, `noteID`
- `ctrl+s`: validate non-empty title+body (API requires both); `CreateNote` or `UpdateNote`; on success set Saved and exit edit mode
- `esc`: if dirty ask confirm in status (`y`/`n`); else exit

- [ ] **Step 2: Integrate into App**

Mode enum: `modeLogin | modeBrowse | modeEdit | modeConfirmDelete`

Browse keys:
- `e` → edit current note
- `n` → edit blank new note
- `d` → confirm delete → `DeleteNote` → refresh list

- [ ] **Step 3: CLI create/edit**

- `ubinote create` → `tui.RunEditor` new
- `ubinote edit <id>` → fetch note → `RunEditor`

- [ ] **Step 4: Manual smoke**

Create a note in TUI, save, verify appears in list and `ubinote list`.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/ internal/cli/notes.go
git commit -m "feat: add note editor and TUI/CLI create-edit-delete"
```

---

### Task 8: Format key chords + ctrl+p palette

**Files:**
- Create: `internal/tui/format_keys.go`
- Create: `internal/tui/palette.go`
- Create: `internal/tui/keys_test.go`
- Modify: `internal/tui/editor.go`

**Interfaces:**
- Consumes: `format.Apply`, textarea value + cursor (selection: use cursor as both start/end if bubbles textarea lacks selection API; if selection available, pass real range)
- Produces: `func BindingFor(msg tea.KeyMsg) (format.Action, bool)`
- Palette: `bubbles/list` overlay of all actions; Enter applies

Default map (from spec):

| Key | Action |
| --- | --- |
| ctrl+b | bold |
| ctrl+i | italic |
| ctrl+k | link |
| alt+1/2/3 | h1/h2/h3 |
| ctrl+shift+c | codeblock |
| alt+c | code |
| alt+u | ul |
| alt+o | ol |
| alt+t | check |
| ctrl+] | indent |
| ctrl+[ | outdent |
| alt+q | quote |
| ctrl+p | open palette |

- [ ] **Step 1: Write key map unit test**

```go
func TestBindingForBold(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyCtrlB}
	a, ok := tui.BindingFor(msg)
	if !ok || a != format.Bold {
		t.Fatalf("got %v %v", a, ok)
	}
}
```

Cover several chords.

- [ ] **Step 2: Implement BindingFor + palette model**

When applying format: read textarea value; determine start/end (document limitation: if no selection API, apply at cursor with `start==end`); set value from `Result`; move cursor to `SelectionEnd`.

Show brief status: `applied bold`.

- [ ] **Step 3: Wire into editor Update**

If palette open, it consumes keys first. Else try `BindingFor`. `ctrl+p` toggles palette.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/tui/ ./internal/format/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/tui/
git commit -m "feat: add markdown format shortcuts and palette"
```

---

### Task 9: README + polish + end-to-end verification

**Files:**
- Create: `README.md`
- Modify: help text / footer strings as needed

- [ ] **Step 1: Write README**

Cover: prerequisites (API running), install `go install`/`go build`, config path, command table, TUI keys, format shortcuts table, env `UBINOTE_API_URL`.

- [ ] **Step 2: Run full test suite**

```bash
go test ./...
go build -o bin/ubinote ./cmd/ubinote
```

Expected: all tests pass; binary builds.

- [ ] **Step 3: Manual E2E against local API** (if API available)

```bash
./bin/ubinote register
./bin/ubinote login
./bin/ubinote create
./bin/ubinote list
./bin/ubinote tui
```

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: add ubinote CLI usage README"
```

---

## Spec coverage check

| Spec requirement | Task |
| --- | --- |
| Hybrid cobra + TUI | 4–7 |
| Built-in editor | 7 |
| Stored JWT | 1, 4 |
| Full CRUD | 5, 7 |
| Master–detail | 6 |
| Raw editor right pane | 7 |
| Glamour view | 5, 6 |
| Format actions + keys + palette | 2, 8 |
| Config URL | 1, 4 |
| 401 clears token | 5 |
| format + api tests | 2, 3 |
| README | 9 |

## Placeholder / consistency review

- Types aligned: `api.Note`, `format.Action`, `config.Config`/`Credentials`
- Default URL consistently `http://localhost:8000`
- No TBD steps remaining
