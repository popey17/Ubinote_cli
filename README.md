# ubinote (CLI)

Terminal client for the [personal_note](../personal_note) API. Notes are Markdown. Use scriptable commands or an interactive master–detail TUI.

## Requirements

- Go 1.27+
- Running API (default `http://localhost:8000`)

## Install

```bash
cd ubinote_cli
go build -o bin/ubinote ./cmd/ubinote
```

## Config

Stored under `~/.config/ubinote/` (or `$XDG_CONFIG_HOME/ubinote/`):

| File | Purpose |
| --- | --- |
| `config.json` | `{ "api_url": "http://localhost:8000" }` |
| `credentials.json` | `{ "token": "…" }` (mode `0600`) |

Override URL: `UBINOTE_API_URL=http://host:port`

```bash
./bin/ubinote config set-url http://localhost:8000
./bin/ubinote config show
```

## Auth

```bash
./bin/ubinote register
./bin/ubinote login
./bin/ubinote logout
```

## Commands

| Command | Description |
| --- | --- |
| `ubinote` / `ubinote tui` | Interactive browser |
| `ubinote list` | List notes |
| `ubinote view <id>` | Render Markdown to stdout |
| `ubinote create` | Create note in TUI editor |
| `ubinote edit <id>` | Edit note in TUI editor |
| `ubinote delete <id>` | Delete with confirmation |

## TUI keys

| Key | Action |
| --- | --- |
| ↑↓ / mouse | Navigate list |
| Enter | Open note |
| `e` | Edit |
| `n` | New note |
| `d` | Delete |
| `/` | Filter list |
| `q` | Quit |

### Editor

| Key | Action |
| --- | --- |
| `ctrl+s` | Save |
| `esc` | Cancel |
| `tab` | Title ↔ body |
| `ctrl+p` | Format palette |

### Format shortcuts (body focused)

| Key | Format |
| --- | --- |
| `ctrl+b` | Bold |
| `ctrl+i` | Italic |
| `ctrl+k` | Link |
| `alt+1` / `2` / `3` | Headings |
| `alt+c` | Inline code |
| `ctrl+shift+c` | Code block |
| `alt+u` / `alt+o` | Bullet / numbered list |
| `alt+t` | Task |
| `ctrl+]` / `ctrl+[` | Indent / outdent |
| `alt+q` | Quote |

If a chord does not reach the terminal, use `ctrl+p`.

## Develop

```bash
go test ./...
go run ./cmd/ubinote --help
```
