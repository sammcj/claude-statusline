# claude-statusline Design

## Summary

Go CLI that generates configurable status lines for Claude Code.
Starship-like TOML config with modules, styles, palettes, and powerline support.
Distributed via Homebrew tap (`felipeelias/homebrew-tap`).

## How It Works

1. Claude Code invokes `claude-statusline` as a status line command
2. Claude Code pipes JSON session data to stdin
3. `claude-statusline` reads config, parses JSON, runs modules, renders styled output
4. Outputs a single line of ANSI-styled text to stdout

Integration in `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "claude-statusline"
  }
}
```

## Config

Location: `~/.config/claude-statusline/config.toml`

Zero-config works out of the box with sensible defaults.

```toml
format = "$model | $directory | $cost | $context"
palette = "default"

[palettes.default]
accent = "cyan"
cost_ok = "green"
cost_warn = "yellow"
cost_high = "red"
ctx_ok = "green"
ctx_warn = "yellow"
ctx_high = "red"

[model]
format = "{{.DisplayName}}"
style = "bold"

[directory]
format = "{{.Dir}}"
style = "palette:accent"
truncation_length = 3

[cost]
format = "${{printf \"%.2f\" .TotalCostUSD}}"
style = "palette:cost_ok"
thresholds = [
  { above = 1.0, style = "palette:cost_warn" },
  { above = 5.0, style = "palette:cost_high" },
]

[context]
format = "{{.Bar}} {{printf \"%.0f\" .UsedPct}}%"
style = "palette:ctx_ok"
bar_width = 5
bar_fill = "█"
bar_empty = "░"
thresholds = [
  { above = 50, style = "palette:ctx_warn" },
  { above = 70, style = "208" },
  { above = 90, style = "palette:ctx_high" },
]

[git_branch]
format = " {{.Branch}}"
style = "palette:accent"
disabled = true

[session_timer]
format = "{{.Elapsed}}"
style = "dim"
disabled = true

[lines_changed]
format = "+{{.Added}} -{{.Removed}}"
added_style = "green"
removed_style = "red"
disabled = true
```

## Format String

The top-level `format` controls which modules appear and in what order.
Module variables use `$module_name`. Everything else is literal text.

```toml
# Plain (default)
format = "$model | $directory | $cost | $context"

# Powerline (user opts in, requires Nerd Font)
format = "[](bg:blue)$model[](fg:blue bg:cyan)$directory[](fg:cyan bg:green)$cost[](fg:green)"
```

No separate `separator` or `preset` config. The format string is the
single source of truth for layout, exactly like starship.

## Style System

Each module has a `style` field. Supports:

- Named colors: `red`, `green`, `cyan`, `blue`, `white`, `black`
- Attributes: `bold`, `dim`, `italic`, `underline`
- 256-color: `"208"` (orange)
- Hex: `"#ff5500"`
- Combined: `"fg:#aaa bg:#333 bold"`
- Palette refs: `"palette:accent"` (resolved from active palette)

Styled text groups in the format string use starship syntax:
`[text](style)` for literal styled text (used for powerline separators).

## Palettes

Named color palettes. The `palette` key selects the active one.
Only `default` ships in the binary. Additional palettes (Tokyo Night,
Gruvbox, Catppuccin, etc.) documented in the README as copy-paste examples.

```toml
palette = "tokyo-night"

[palettes.tokyo-night]
accent = "#769ff0"
cost_ok = "#73daca"
cost_warn = "#e0af68"
cost_high = "#f7768e"
ctx_ok = "#73daca"
ctx_warn = "#e0af68"
ctx_high = "#f7768e"
```

## Modules

7 built-in modules. Each has `format`, `style`, `disabled` fields.
Module-specific fields where relevant.

| Module | Default on | Data source |
|--------|-----------|-------------|
| `model` | yes | stdin `.model.display_name` |
| `directory` | yes | stdin `.cwd` (tilde-collapsed) |
| `cost` | yes | stdin `.cost.total_cost_usd` |
| `context` | yes | stdin `.context_window.used_percentage` |
| `git_branch` | no | `git rev-parse --abbrev-ref HEAD` |
| `session_timer` | no | stdin `.cost.total_duration_ms` |
| `lines_changed` | no | stdin `.cost.total_lines_added/removed` |

### Thresholds

`cost` and `context` modules support thresholds that change style
based on value:

```toml
[cost]
style = "green"
thresholds = [
  { above = 1.0, style = "yellow" },
  { above = 5.0, style = "red" },
]
```

Thresholds are evaluated in order. The last matching threshold wins.

### Context Bar

The `context` module renders a visual progress bar:

```
██░░░ 42%
```

Configurable via `bar_width`, `bar_fill`, `bar_empty`.

### Directory Truncation

`truncation_length` controls how many path segments to show:

```
~/a/very/deep/nested/path → ~/n/path  (truncation_length = 1)
~/a/very/deep/nested/path → ~/d/n/path  (truncation_length = 2)
```

### Git Branch

Runs `git rev-parse --abbrev-ref HEAD`. Returns empty if not in a git repo.
Disabled by default.

### Session Timer

Formats `cost.total_duration_ms` as `HH:MM:SS` or `MM:SS`.

### Lines Changed

Shows `+N -M` with separate styles for added/removed.

## JSON Input

Claude Code pipes this JSON to stdin (relevant fields):

```json
{
  "model": { "id": "...", "display_name": "Claude Opus 4" },
  "cwd": "/home/user/project",
  "cost": {
    "total_cost_usd": 1.23,
    "total_duration_ms": 300000,
    "total_lines_added": 42,
    "total_lines_removed": 7
  },
  "context_window": {
    "used_percentage": 42.5,
    "remaining_percentage": 57.5,
    "context_window_size": 200000
  },
  "session_id": "...",
  "version": "..."
}
```

Fields may be absent or null. Modules handle missing data gracefully
(render empty string, skip segment).

## Default Output (zero config)

```
Claude Opus 4 | ~/project | $0.42 | ██░░░ 42%
```

With ANSI colors: model bold, directory cyan, cost green, context bar
green/yellow/red based on thresholds.

## Project Structure

```
claude-statusline/
├── main.go
├── go.mod
├── internal/
│   ├── config/       # TOML parsing, defaults, palettes
│   ├── input/        # JSON stdin parsing
│   ├── modules/      # One file per module
│   ├── render/       # Format string parsing, module composition
│   └── style/        # Color/style/palette → ANSI escape codes
├── config.example.toml
├── .goreleaser.yml
└── .github/workflows/
    ├── ci.yml
    └── release.yml
```

## Distribution

- GoReleaser builds cross-platform binaries (darwin/linux, amd64/arm64)
- GitHub releases on tag push
- Homebrew formula auto-published to `felipeelias/homebrew-tap`
- `brew install felipeelias/tap/claude-statusline`

## Future (not v1)

- More modules (vim mode, worktree, token counts, cache efficiency)
- `claude-statusline init` command to generate config interactively
- `claude-statusline explain` to preview output with sample data
