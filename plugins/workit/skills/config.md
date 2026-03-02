# config.md

Configuration management. View, set, and remove persistent config values.

## Read
- `wk config path` — print config file location (alias: `where`)
- `wk config list` — list all config values (aliases: `ls`, `all`)
- `wk config keys` — list available config keys (aliases: `list-keys`, `names`)
- `wk config get <key>` — get a config value (alias: `show`)

## Write
- `wk config set <key> <value>` — set a config value (aliases: `add`, `update`)
- `wk config unset <key>` — remove a config value (aliases: `rm`, `del`, `remove`)

## Safety
- Read commands (`path`, `list`, `keys`, `get`) are safe with `--read-only`.
- Write commands (`set`, `unset`) support `--dry-run` to preview changes.

## Examples
```bash
# Find the config file
wk config path

# List all current config values
wk config list
wk config list --json

# List available config keys
wk config keys
wk config keys --json

# Get a specific value
wk config get timezone

# Set config values
wk config set timezone UTC
wk config set keyring_backend file

# Preview a config change
wk config set timezone America/New_York --dry-run

# Remove a config value
wk config unset timezone
```

## Command index
`path` `list` `keys` `get` `set` `unset`
