# ubinote CLI

Terminal client for the [personal_note](../personal_note) API (same notes as [ubinote_web](../ubinote_web)).

Browse and edit Markdown notes in an interactive TUI, or use scriptable commands.

## Features

- Welcome menu with mouse + keyboard
- Login / register (JWT stored locally)
- Master–detail notes browser (list + rendered Markdown)
- Built-in editor with format shortcuts and palette (`ctrl+p`)
- Slow-load warning when the API is waking up
- CLI commands for list / view / create / edit / delete

## Install

**With Go**

```bash
go install github.com/popey17/Ubinote_cli/cmd/ubinote@latest
```

Ensure `$(go env GOPATH)/bin` is on your `PATH`, then run `ubinote`.

**From source**

```bash
git clone https://github.com/popey17/Ubinote_cli.git
cd Ubinote_cli
go build -o bin/ubinote ./cmd/ubinote
./bin/ubinote
```

## Requirements

- Go 1.27+ (for `go install` / building from source)
- Reachable ubinote API (default URL is baked into the binary; override with `ubinote config set-url` or `UBINOTE_API_URL`)

## Configuration

Files live under `~/.config/ubinote/` (or `$XDG_CONFIG_HOME/ubinote/`):

| File | Purpose |
| --- | --- |
| `config.json` | `{ "api_url": "http://localhost:8000" }` |
| `credentials.json` | `{ "token": "…" }` (mode `0600`) |

```bash
./bin/ubinote config show
./bin/ubinote config set-url http://localhost:8000
```

Environment override:

```bash
export UBINOTE_API_URL=http://host:port
```

## Commands

| Command | Description |
| --- | --- |
| `ubinote` / `ubinote tui` | Open the interactive app |
| `ubinote register` | Create an account (CLI prompts) |
| `ubinote login` | Log in and store JWT |
| `ubinote logout` | Clear local token |
| `ubinote list` | List note ids and titles |
| `ubinote view <id>` | Render Markdown to the terminal |
| `ubinote create` | Create a note in the TUI editor |
| `ubinote edit <id>` | Edit a note in the TUI editor |
| `ubinote delete <id>` | Delete a note (asks for confirmation) |
| `ubinote config show` | Show API URL and login status |
| `ubinote config set-url <url>` | Set API base URL |

## TUI

### Flow

**Welcome → Login → Notes**

If a session already exists, the app opens the notes browser directly.

### Welcome

| Action | How |
| --- | --- |
| Login | Click or ↑↓ + Enter |
| Help / Info | Click or ↑↓ + Enter |
| Quit | `q` or click quit hint |

### Login

| Action | How |
| --- | --- |
| Focus fields | Click or Tab |
| Submit | Click **Sign in** / Enter |
| Toggle register | Click toggle or `ctrl+r` |
| Back | Click **Back** or Esc |

If connect takes longer than ~3s, a wake-up warning is shown.

### Notes

| Key | Action |
| --- | --- |
| ↑↓ | Move between notes |
| `e` | Edit selected note |
| `n` | New note |
| `d` | Delete selected note |
| `r` | Refresh from API |
| `L` | Log out (confirm with `y`) |
| `/` | Filter list |
| `q` | Quit |

Selected note title is underlined. Preview is rendered Markdown (glamour).

### Editor

| Key | Action |
| --- | --- |
| `ctrl+s` | Save |
| `esc` | Cancel (confirm if dirty) |
| `tab` | Title ↔ body |
| `ctrl+p` | Format palette |

#### Format shortcuts (body focused)

| Key | Format |
| --- | --- |
| `ctrl+b` | Bold |
| `ctrl+i` | Italic |
| `ctrl+k` | Link |
| `alt+1` / `2` / `3` | Headings |
| `alt+c` | Inline code |
| `ctrl+shift+c` | Code block |
| `alt+u` / `alt+o` | Bullet / numbered list |
| `alt+t` | Task checklist |
| `ctrl+]` / `ctrl+[` | Indent / outdent |
| `alt+q` | Quote |

If a chord is swallowed by the terminal, use `ctrl+p`.

## Project layout

```
cmd/ubinote/          entrypoint
internal/api/         HTTP client for personal_note
internal/cli/         cobra commands
internal/config/      config + credentials
internal/format/      Markdown format actions (web toolbar parity)
internal/tui/         Bubble Tea app (welcome, login, browse, editor)
internal/ui/          shared lipgloss theme
```

## Stack

- [cobra](https://github.com/spf13/cobra) — commands
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [bubbles](https://github.com/charmbracelet/bubbles) + [lipgloss](https://github.com/charmbracelet/lipgloss) — TUI
- [glamour](https://github.com/charmbracelet/glamour) — Markdown rendering

## Develop

```bash
go test ./...
go run ./cmd/ubinote --help
go build -o bin/ubinote ./cmd/ubinote
```

## Related projects

| Path | Role |
| --- | --- |
| [`../personal_note`](../personal_note) | Go REST API |
| [`../ubinote_web`](../ubinote_web) | Web UI (Vite + React) |
