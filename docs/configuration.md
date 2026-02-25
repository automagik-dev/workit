# Configuration

> Back to [README](../README.md)

## Config File (JSON5)

Find the actual config path with `wk config path` or `wk auth keyring`.

### Typical paths

- **macOS**: `~/Library/Application Support/workit/config.json`
- **Linux**: `~/.config/workit/config.json` (or `$XDG_CONFIG_HOME/workit/config.json`)
- **Windows**: `%AppData%\workit\config.json`

### Example

JSON5 supports comments and trailing commas:

```json5
{
  // Avoid macOS Keychain prompts
  keyring_backend: "file",
  // Default output timezone for Calendar/Gmail (IANA, UTC, or local)
  default_timezone: "UTC",
  // Optional account aliases
  account_aliases: {
    work: "work@company.com",
    personal: "me@gmail.com",
  },
  // Optional per-account OAuth client selection
  account_clients: {
    "work@company.com": "work",
  },
  // Optional domain -> client mapping
  client_domains: {
    "example.com": "work",
  },
}
```

## Config Commands

```bash
wk config path                          # Show config file path
wk config list                          # List all config values
wk config keys                          # List all known config keys
wk config get default_timezone          # Get a specific value
wk config set default_timezone UTC      # Set a value
wk config unset default_timezone        # Remove a value
```

## Account Selection

Specify the account using either a flag or environment variable:

```bash
# Via flag
wk gmail search 'newer_than:7d' --account you@gmail.com

# Via alias
wk auth alias set work work@company.com
wk gmail search 'newer_than:7d' --account work

# Via environment
export WK_ACCOUNT=you@gmail.com
wk gmail search 'newer_than:7d'

# Auto-select (default account or the single stored token)
wk gmail labels list --account auto
```

## Account Aliases

```bash
wk auth alias set work work@company.com
wk auth alias list
wk auth alias unset work
```

Aliases work anywhere you pass `--account` or `WK_ACCOUNT` (reserved names: `auto`, `default`).

## Command Allowlist (Sandboxing)

Restrict which top-level commands are available:

```bash
# Only allow calendar + tasks commands for an agent
wk --enable-commands calendar,tasks calendar events --today

# Same via env
export WK_ENABLE_COMMANDS=calendar,tasks
wk tasks list <tasklistId>
```

## Output Modes

### Default (human-friendly)

Human-readable tables on stdout with colors (when connected to a TTY).

```bash
wk gmail search 'newer_than:7d' --max 3
```

### Plain (`--plain`)

Stable TSV on stdout (tabs preserved; best for piping to tools that expect `\t`). No colors.

```bash
wk gmail search 'newer_than:7d' --max 3 --plain
```

### JSON (`--json`)

Machine-readable output for scripting and automation:

```bash
wk gmail search 'newer_than:7d' --max 3 --json
```

Data goes to stdout, errors and progress to stderr for clean piping:

```bash
wk --json drive ls --max 5 | jq '.files[] | select(.mimeType=="application/pdf")'
```

Colors are enabled only in rich TTY output and are disabled automatically for `--json` and `--plain`.

### JSON Field Selection (`--select`)

Select specific fields from JSON output:

```bash
wk drive ls --json --select "name,id,size"
```

Pass an empty string to discover available fields:

```bash
wk drive ls --json --select ""
```

### JQ Expressions (`--jq`)

Apply jq expressions directly:

```bash
wk drive ls --json --jq '.files[] | .name'
```

### Results Only (`--results-only`)

In JSON mode, emit only the primary result (drops envelope fields like `nextPageToken`):

```bash
wk drive ls --json --results-only
```

## Environment Variables Reference

| Variable | Description |
|---|---|
| `WK_ACCOUNT` | Default account email or alias to use (avoids repeating `--account`; otherwise uses keyring default or a single stored token) |
| `WK_CLIENT` | OAuth client name (selects stored credentials + token bucket) |
| `WK_JSON` | Default JSON output |
| `WK_PLAIN` | Default plain output |
| `WK_COLOR` | Color mode: `auto` (default), `always`, or `never` |
| `WK_TIMEZONE` | Default output timezone for Calendar/Gmail (IANA name, `UTC`, or `local`) |
| `WK_ENABLE_COMMANDS` | Comma-separated allowlist of top-level commands (e.g., `calendar,tasks`) |
| `WK_KEYRING_BACKEND` | Force keyring backend: `auto`, `keychain`, or `file` (overrides config) |
| `WK_KEYRING_PASSWORD` | Password for the encrypted on-disk keyring (file backend; avoids interactive prompt) |
| `WK_READ_ONLY` | Set to `true` to hide write commands and request read-only OAuth scopes |
| `WK_COMMAND_TIER` | Command visibility tier: `core`, `extended`, or `complete` (default: `complete`) |
| `WK_CALENDAR_WEEKDAY` | Set to `1` to default `--weekday` for calendar events output |
| `WK_CONFIG_DIR` | Override config directory path (useful for isolated headless sessions) |

## Shell Completions

Generate shell completions for your preferred shell:

### Bash

```bash
# macOS (with Homebrew)
wk completion bash > $(brew --prefix)/etc/bash_completion.d/wk

# Linux
wk completion bash > /etc/bash_completion.d/wk

# Or load directly in your current session
source <(wk completion bash)
```

### Zsh

```zsh
wk completion zsh > "${fpath[1]}/_wk"

# Or add to .zshrc for automatic loading
echo 'eval "$(wk completion zsh)"' >> ~/.zshrc
```

### Fish

```fish
wk completion fish > ~/.config/fish/completions/wk.fish
```

### PowerShell

```powershell
wk completion powershell | Out-String | Invoke-Expression

# Or add to profile for all sessions
wk completion powershell >> $PROFILE
```

After installing completions, start a new shell session for changes to take effect.
