# claude-statusline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI that generates configurable, styled status lines for Claude Code with starship-like TOML config.

**Architecture:** Single binary reads JSON from stdin, loads TOML config from `~/.config/claude-statusline/config.toml`, evaluates enabled modules against a format string, renders ANSI-styled output to stdout. Modules are Go interfaces registered in a map. Style system parses starship-like style strings into ANSI escape codes.

**Tech Stack:** Go 1.26, BurntSushi/toml, stretchr/testify, text/template (stdlib). No CLI framework — stdin/stdout only.

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/input/input.go`
- Create: `internal/config/config.go`
- Create: `internal/style/style.go`
- Create: `internal/modules/module.go`
- Create: `internal/render/render.go`

**Step 1: Initialize Go module and install dependencies**

```bash
go mod init github.com/felipeelias/claude-statusline
go get github.com/BurntSushi/toml@latest
go get github.com/stretchr/testify@latest
```

**Step 2: Create directory structure**

```bash
mkdir -p internal/{input,config,style,modules,render}
```

**Step 3: Create minimal main.go**

```go
package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	fmt.Fprintln(os.Stderr, "claude-statusline", version)
}
```

**Step 4: Verify it compiles**

Run: `go build -o claude-statusline .`
Expected: binary created, no errors

**Step 5: Commit**

```bash
git init
git add go.mod go.sum main.go internal/
git commit -m "feat: project scaffolding"
```

---

### Task 2: JSON Input Parsing

**Files:**
- Create: `internal/input/input.go`
- Create: `internal/input/input_test.go`

**Step 1: Write failing tests**

```go
package input_test

import (
	"strings"
	"testing"

	"github.com/felipeelias/claude-statusline/internal/input"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	json := `{
		"model": {"id": "claude-opus-4", "display_name": "Claude Opus 4"},
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
		}
	}`

	data, err := input.Parse(strings.NewReader(json))
	require.NoError(t, err)

	assert.Equal(t, "claude-opus-4", data.Model.ID)
	assert.Equal(t, "Claude Opus 4", data.Model.DisplayName)
	assert.Equal(t, "/home/user/project", data.Cwd)
	assert.InDelta(t, 1.23, data.Cost.TotalCostUSD, 0.001)
	assert.Equal(t, 300000, data.Cost.TotalDurationMs)
	assert.Equal(t, 42, data.Cost.TotalLinesAdded)
	assert.Equal(t, 7, data.Cost.TotalLinesRemoved)
	assert.InDelta(t, 42.5, data.ContextWindow.UsedPercentage, 0.01)
	assert.InDelta(t, 57.5, data.ContextWindow.RemainingPercentage, 0.01)
	assert.Equal(t, 200000, data.ContextWindow.ContextWindowSize)
}

func TestParseEmpty(t *testing.T) {
	data, err := input.Parse(strings.NewReader("{}"))
	require.NoError(t, err)

	assert.Equal(t, "", data.Model.DisplayName)
	assert.Equal(t, "", data.Cwd)
	assert.InDelta(t, 0.0, data.Cost.TotalCostUSD, 0.001)
}

func TestParseInvalid(t *testing.T) {
	_, err := input.Parse(strings.NewReader("not json"))
	assert.Error(t, err)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/input/ -v`
Expected: FAIL — package doesn't exist yet

**Step 3: Implement input parsing**

```go
package input

import (
	"encoding/json"
	"io"
)

type Data struct {
	Model         Model         `json:"model"`
	Cwd           string        `json:"cwd"`
	Cost          Cost          `json:"cost"`
	ContextWindow ContextWindow `json:"context_window"`
	SessionID     string        `json:"session_id"`
	Version       string        `json:"version"`
}

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Cost struct {
	TotalCostUSD      float64 `json:"total_cost_usd"`
	TotalDurationMs   int     `json:"total_duration_ms"`
	TotalLinesAdded   int     `json:"total_lines_added"`
	TotalLinesRemoved int     `json:"total_lines_removed"`
}

type ContextWindow struct {
	UsedPercentage      float64 `json:"used_percentage"`
	RemainingPercentage float64 `json:"remaining_percentage"`
	ContextWindowSize   int     `json:"context_window_size"`
}

func Parse(r io.Reader) (Data, error) {
	var data Data
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return Data{}, err
	}
	return data, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/input/ -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/input/
git commit -m "feat: add JSON input parsing from stdin"
```

---

### Task 3: Style System

**Files:**
- Create: `internal/style/style.go`
- Create: `internal/style/style_test.go`

Parse starship-like style strings (`"bold green"`, `"fg:#ff5500 bg:blue"`, `"208"`) into ANSI escape sequences.

**Step 1: Write failing tests**

```go
package style_test

import (
	"testing"

	"github.com/felipeelias/claude-statusline/internal/style"
	"github.com/stretchr/testify/assert"
)

func TestWrapNamed(t *testing.T) {
	tests := []struct {
		name     string
		style    string
		text     string
		expected string
	}{
		{"bold", "bold", "hi", "\033[1mhi\033[0m"},
		{"green fg", "green", "hi", "\033[32mhi\033[0m"},
		{"bold green", "bold green", "hi", "\033[1;32mhi\033[0m"},
		{"dim", "dim", "hi", "\033[2mhi\033[0m"},
		{"italic", "italic", "hi", "\033[3mhi\033[0m"},
		{"underline", "underline", "hi", "\033[4mhi\033[0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := style.Parse(tt.style)
			assert.Equal(t, tt.expected, s.Wrap(tt.text))
		})
	}
}

func TestWrapHex(t *testing.T) {
	s := style.Parse("fg:#ff5500")
	assert.Equal(t, "\033[38;2;255;85;0mhi\033[0m", s.Wrap("hi"))
}

func TestWrapBg(t *testing.T) {
	s := style.Parse("bg:blue")
	assert.Equal(t, "\033[44mhi\033[0m", s.Wrap("hi"))
}

func TestWrapBgHex(t *testing.T) {
	s := style.Parse("bg:#333333")
	assert.Equal(t, "\033[48;2;51;51;51mhi\033[0m", s.Wrap("hi"))
}

func TestWrap256Color(t *testing.T) {
	s := style.Parse("208")
	assert.Equal(t, "\033[38;5;208mhi\033[0m", s.Wrap("hi"))
}

func TestWrapCombined(t *testing.T) {
	s := style.Parse("fg:#aaaaaa bg:#333333 bold")
	result := s.Wrap("hi")
	assert.Contains(t, result, "\033[")
	assert.Contains(t, result, "1")  // bold
	assert.Contains(t, result, "38;2;170;170;170") // fg
	assert.Contains(t, result, "48;2;51;51;51")    // bg
	assert.True(t, result[len(result)-4:] == "\033[0m")
}

func TestWrapEmpty(t *testing.T) {
	s := style.Parse("")
	assert.Equal(t, "hi", s.Wrap("hi"))
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/style/ -v`
Expected: FAIL

**Step 3: Implement style parsing**

`internal/style/style.go` — Parse style string tokens, map named colors to ANSI codes, support hex via 24-bit escape sequences, support 256-color, combine into a single ANSI prefix string. `Wrap(text)` returns `\033[<codes>m<text>\033[0m`.

Named colors map:
- black=30, red=31, green=32, yellow=33, blue=34, magenta=35, cyan=36, white=37
- bg variants: +10 (40-47)

Attributes: bold=1, dim=2, italic=3, underline=4

Hex: `fg:#RRGGBB` → `38;2;R;G;B`, `bg:#RRGGBB` → `48;2;R;G;B`

256-color: bare number like `"208"` → `38;5;208`

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/style/ -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/style/
git commit -m "feat: add ANSI style parsing system"
```

---

### Task 4: Config Loading with Defaults and Palettes

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write failing tests**

```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	cfg := config.Default()

	assert.Equal(t, "$model | $directory | $cost | $context", cfg.Format)
	assert.Equal(t, "default", cfg.Palette)
	assert.False(t, cfg.Model.Disabled)
	assert.True(t, cfg.GitBranch.Disabled)
	assert.True(t, cfg.SessionTimer.Disabled)
	assert.True(t, cfg.LinesChanged.Disabled)
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	err := os.WriteFile(path, []byte(`
format = "$model | $cost"
palette = "custom"

[palettes.custom]
accent = "#ff0000"

[model]
format = "M: {{.DisplayName}}"
style = "bold"

[git_branch]
disabled = false
`), 0o644)
	require.NoError(t, err)

	cfg, err := config.Load(path)
	require.NoError(t, err)

	assert.Equal(t, "$model | $cost", cfg.Format)
	assert.Equal(t, "custom", cfg.Palette)
	assert.Equal(t, "M: {{.DisplayName}}", cfg.Model.Format)
	assert.False(t, cfg.GitBranch.Disabled)
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := config.Load("/nonexistent/config.toml")
	require.NoError(t, err) // missing file is OK, use defaults
	assert.Equal(t, "$model | $directory | $cost | $context", cfg.Format)
}

func TestResolvePaletteColor(t *testing.T) {
	cfg := config.Default()
	assert.Equal(t, "cyan", cfg.ResolveStyle("palette:accent"))
	assert.Equal(t, "bold green", cfg.ResolveStyle("bold green"))
}

func TestThresholds(t *testing.T) {
	cfg := config.Default()
	assert.Equal(t, 5, len(cfg.Context.Thresholds))
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -v`
Expected: FAIL

**Step 3: Implement config loading**

`internal/config/config.go`:
- `Config` struct with `Format`, `Palette`, `Palettes` map, and per-module config structs
- `Default()` returns hardcoded defaults (format, palette "default" with colors, module defaults)
- `Load(path)` reads TOML file, merges with defaults. Missing file returns defaults (no error).
- `ResolveStyle(s)` — if s starts with `"palette:"`, look up the color name in the active palette. Otherwise return s unchanged.
- Module configs: `ModelConfig`, `DirectoryConfig`, `CostConfig`, `ContextConfig`, `GitBranchConfig`, `SessionTimerConfig`, `LinesChangedConfig`
- Each module config has `Format`, `Style`, `Disabled` fields
- Cost and Context have `Thresholds []Threshold` where `Threshold` is `{Above float64, Style string}`
- Context also has `BarWidth`, `BarFill`, `BarEmpty`
- Directory has `TruncationLength`

Default palette:

```go
"default": {
    "accent":   "cyan",
    "cost_ok":  "green",
    "cost_warn": "yellow",
    "cost_high": "red",
    "ctx_ok":   "green",
    "ctx_warn": "yellow",
    "ctx_high": "red",
}
```

Default thresholds for context: `[{50, "palette:ctx_warn"}, {70, "208"}, {90, "palette:ctx_high"}]`

Default thresholds for cost: `[{1.0, "palette:cost_warn"}, {5.0, "palette:cost_high"}]`

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add TOML config with defaults and palettes"
```

---

### Task 5: Module Interface and Model Module

**Files:**
- Create: `internal/modules/module.go`
- Create: `internal/modules/model.go`
- Create: `internal/modules/model_test.go`

**Step 1: Write failing tests**

```go
package modules_test

import (
	"testing"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/input"
	"github.com/felipeelias/claude-statusline/internal/modules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelModule(t *testing.T) {
	cfg := config.Default()
	data := input.Data{
		Model: input.Model{DisplayName: "Claude Opus 4"},
	}

	m := modules.NewModel(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "Claude Opus 4")
}

func TestModelModuleEmpty(t *testing.T) {
	cfg := config.Default()
	data := input.Data{}

	m := modules.NewModel(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Equal(t, "", result)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/modules/ -v`
Expected: FAIL

**Step 3: Implement module interface and model module**

`internal/modules/module.go`:

```go
package modules

import (
	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/input"
)

type Module interface {
	Name() string
	Render(data input.Data, cfg config.Config) (string, error)
}
```

`internal/modules/model.go`:
- Struct with format and style from config
- `Render` executes Go template with `DisplayName` field
- Returns empty string if DisplayName is empty
- Wraps result in resolved ANSI style

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/ -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/modules/
git commit -m "feat: add module interface and model module"
```

---

### Task 6: Directory Module

**Files:**
- Create: `internal/modules/directory.go`
- Create: `internal/modules/directory_test.go`

**Step 1: Write failing tests**

```go
package modules_test

import (
	"testing"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/input"
	"github.com/felipeelias/claude-statusline/internal/modules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectoryModule(t *testing.T) {
	cfg := config.Default()
	data := input.Data{Cwd: "/home/testuser/projects/myapp"}

	m := modules.NewDirectory(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	// Default truncation_length = 3, tilde substitution
	assert.Contains(t, result, "myapp")
}

func TestDirectoryTildeSubstitution(t *testing.T) {
	cfg := config.Default()
	home := "/home/testuser"
	t.Setenv("HOME", home)
	data := input.Data{Cwd: home + "/projects/myapp"}

	m := modules.NewDirectory(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "~")
	assert.NotContains(t, result, home)
}

func TestDirectoryTruncation(t *testing.T) {
	cfg := config.Default()
	cfg.Directory.TruncationLength = 2
	home := "/home/testuser"
	t.Setenv("HOME", home)
	data := input.Data{Cwd: home + "/a/very/deep/nested/path"}

	m := modules.NewDirectory(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "n/path")
	assert.NotContains(t, result, "a/very")
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/modules/ -v -run TestDirectory`
Expected: FAIL

**Step 3: Implement directory module**

- Tilde substitution: replace `$HOME` prefix with `~`
- Truncation: keep last N path segments, abbreviate earlier ones to first char
- Execute Go template with `Dir` field
- Wrap in resolved style

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/ -v -run TestDirectory`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/modules/directory.go internal/modules/directory_test.go
git commit -m "feat: add directory module with tilde substitution and truncation"
```

---

### Task 7: Cost Module with Thresholds

**Files:**
- Create: `internal/modules/cost.go`
- Create: `internal/modules/cost_test.go`

**Step 1: Write failing tests**

```go
package modules_test

import (
	"testing"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/input"
	"github.com/felipeelias/claude-statusline/internal/modules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCostModule(t *testing.T) {
	cfg := config.Default()
	data := input.Data{Cost: input.Cost{TotalCostUSD: 1.234}}

	m := modules.NewCost(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "$1.23")
}

func TestCostModuleZero(t *testing.T) {
	cfg := config.Default()
	data := input.Data{}

	m := modules.NewCost(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "$0.00")
}

func TestCostModuleThresholdStyle(t *testing.T) {
	cfg := config.Default()

	tests := []struct {
		cost          float64
		expectContain string // ANSI code fragment
	}{
		{0.50, "\033[32m"},  // green (default, below 1.0)
		{2.00, "\033[33m"},  // yellow (above 1.0)
		{6.00, "\033[31m"},  // red (above 5.0)
	}
	for _, tt := range tests {
		data := input.Data{Cost: input.Cost{TotalCostUSD: tt.cost}}
		m := modules.NewCost(cfg)
		result, err := m.Render(data, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, tt.expectContain, "cost=%.2f", tt.cost)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/modules/ -v -run TestCost`
Expected: FAIL

**Step 3: Implement cost module**

- Execute Go template with `TotalCostUSD` (via `printf "%.2f"`)
- Evaluate thresholds: iterate in order, last match wins
- Resolve threshold style via config palette
- Wrap result in winning style

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/ -v -run TestCost`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/modules/cost.go internal/modules/cost_test.go
git commit -m "feat: add cost module with threshold-based styling"
```

---

### Task 8: Context Module with Progress Bar

**Files:**
- Create: `internal/modules/context.go`
- Create: `internal/modules/context_test.go`

**Step 1: Write failing tests**

```go
package modules_test

import (
	"testing"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/input"
	"github.com/felipeelias/claude-statusline/internal/modules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextModule(t *testing.T) {
	cfg := config.Default()
	data := input.Data{
		ContextWindow: input.ContextWindow{UsedPercentage: 42.5},
	}

	m := modules.NewContext(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "██░░░")
	assert.Contains(t, result, "42%")
}

func TestContextBar100Percent(t *testing.T) {
	cfg := config.Default()
	data := input.Data{
		ContextWindow: input.ContextWindow{UsedPercentage: 100.0},
	}

	m := modules.NewContext(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "█████")
	assert.Contains(t, result, "100%")
}

func TestContextBarZero(t *testing.T) {
	cfg := config.Default()
	data := input.Data{
		ContextWindow: input.ContextWindow{UsedPercentage: 0},
	}

	m := modules.NewContext(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "░░░░░")
	assert.Contains(t, result, "0%")
}

func TestContextThresholds(t *testing.T) {
	cfg := config.Default()

	tests := []struct {
		pct           float64
		expectContain string
	}{
		{30.0, "\033[32m"},  // green
		{55.0, "\033[33m"},  // yellow
		{95.0, "\033[31m"},  // red
	}
	for _, tt := range tests {
		data := input.Data{
			ContextWindow: input.ContextWindow{UsedPercentage: tt.pct},
		}
		m := modules.NewContext(cfg)
		result, err := m.Render(data, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, tt.expectContain, "pct=%.0f", tt.pct)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/modules/ -v -run TestContext`
Expected: FAIL

**Step 3: Implement context module**

- Build progress bar: `filled = int(pct / 100 * barWidth)`, fill with `bar_fill`, rest with `bar_empty`
- Execute Go template with `Bar`, `UsedPct` fields
- Evaluate thresholds same as cost module
- Wrap in resolved style

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/ -v -run TestContext`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/modules/context.go internal/modules/context_test.go
git commit -m "feat: add context module with progress bar and thresholds"
```

---

### Task 9: Git Branch, Session Timer, Lines Changed Modules

**Files:**
- Create: `internal/modules/gitbranch.go`
- Create: `internal/modules/gitbranch_test.go`
- Create: `internal/modules/session.go`
- Create: `internal/modules/session_test.go`
- Create: `internal/modules/lines.go`
- Create: `internal/modules/lines_test.go`

**Step 1: Write failing tests**

`gitbranch_test.go`:

```go
func TestGitBranchInRepo(t *testing.T) {
	// Create a temp git repo
	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()
	exec.Command("git", "-C", dir, "commit", "--allow-empty", "-m", "init").Run()

	cfg := config.Default()
	cfg.GitBranch.Disabled = false
	data := input.Data{Cwd: dir}

	m := modules.NewGitBranch(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	// Default branch is main or master
	assert.True(t, result != "", "expected branch name")
}

func TestGitBranchNotInRepo(t *testing.T) {
	cfg := config.Default()
	cfg.GitBranch.Disabled = false
	data := input.Data{Cwd: t.TempDir()}

	m := modules.NewGitBranch(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}
```

`session_test.go`:

```go
func TestSessionTimer(t *testing.T) {
	cfg := config.Default()
	cfg.SessionTimer.Disabled = false
	data := input.Data{Cost: input.Cost{TotalDurationMs: 3661000}} // 1h 1m 1s

	m := modules.NewSessionTimer(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "1:01:01")
}

func TestSessionTimerMinutes(t *testing.T) {
	cfg := config.Default()
	cfg.SessionTimer.Disabled = false
	data := input.Data{Cost: input.Cost{TotalDurationMs: 125000}} // 2m 5s

	m := modules.NewSessionTimer(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "2:05")
}
```

`lines_test.go`:

```go
func TestLinesChanged(t *testing.T) {
	cfg := config.Default()
	cfg.LinesChanged.Disabled = false
	data := input.Data{Cost: input.Cost{TotalLinesAdded: 42, TotalLinesRemoved: 7}}

	m := modules.NewLinesChanged(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "+42")
	assert.Contains(t, result, "-7")
}

func TestLinesChangedZero(t *testing.T) {
	cfg := config.Default()
	cfg.LinesChanged.Disabled = false
	data := input.Data{}

	m := modules.NewLinesChanged(cfg)
	result, err := m.Render(data, cfg)
	require.NoError(t, err)

	assert.Equal(t, "", result) // hide when no changes
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/modules/ -v -run "TestGitBranch|TestSession|TestLines"`
Expected: FAIL

**Step 3: Implement all three modules**

- `gitbranch.go`: Run `git -C <cwd> rev-parse --abbrev-ref HEAD`, return empty on error
- `session.go`: Convert ms to `H:MM:SS` or `M:SS`, execute template with `Elapsed`
- `lines.go`: Render `+Added` in added_style and `-Removed` in removed_style, return empty if both zero

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/ -v -run "TestGitBranch|TestSession|TestLines"`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/modules/gitbranch.go internal/modules/gitbranch_test.go
git add internal/modules/session.go internal/modules/session_test.go
git add internal/modules/lines.go internal/modules/lines_test.go
git commit -m "feat: add git_branch, session_timer, lines_changed modules"
```

---

### Task 10: Format String Renderer

**Files:**
- Create: `internal/render/render.go`
- Create: `internal/render/render_test.go`

Parses the format string, replaces `$module_name` with module output, handles `[text](style)` for inline styled text (powerline separators).

**Step 1: Write failing tests**

```go
package render_test

import (
	"testing"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/input"
	"github.com/felipeelias/claude-statusline/internal/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderPlain(t *testing.T) {
	cfg := config.Default()
	data := input.Data{
		Model:         input.Model{DisplayName: "Claude Opus 4"},
		Cwd:           "/tmp/test",
		Cost:          input.Cost{TotalCostUSD: 0.42},
		ContextWindow: input.ContextWindow{UsedPercentage: 42.5},
	}

	result, err := render.Render(cfg, data)
	require.NoError(t, err)

	// Should contain all 4 default module outputs separated by |
	assert.Contains(t, result, "Claude Opus 4")
	assert.Contains(t, result, "/tmp/test")
	assert.Contains(t, result, "$0.42")
	assert.Contains(t, result, "42%")
	assert.Contains(t, result, " | ")
}

func TestRenderDisabledModule(t *testing.T) {
	cfg := config.Default()
	cfg.Format = "$model | $git_branch | $cost"
	// git_branch is disabled by default
	data := input.Data{
		Model: input.Model{DisplayName: "Opus"},
		Cost:  input.Cost{TotalCostUSD: 1.0},
	}

	result, err := render.Render(cfg, data)
	require.NoError(t, err)

	assert.Contains(t, result, "Opus")
	assert.Contains(t, result, "$1.00")
	// Disabled module should produce empty string; surrounding separators remain
	// (user controls layout via format string)
}

func TestRenderStyledText(t *testing.T) {
	cfg := config.Default()
	cfg.Format = "[hello](bold green)"

	result, err := render.Render(cfg, input.Data{})
	require.NoError(t, err)

	assert.Contains(t, result, "\033[1;32m")
	assert.Contains(t, result, "hello")
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/render/ -v`
Expected: FAIL

**Step 3: Implement renderer**

`internal/render/render.go`:
- Parse format string into tokens: literal text, `$module_name` refs, `[text](style)` groups
- Build module registry map: `{"model": modelModule, "directory": dirModule, ...}`
- For each `$module_name` token: if module is disabled, replace with empty string. Otherwise call `module.Render(data, cfg)`.
- For each `[text](style)` token: parse style and wrap text.
- Concatenate all tokens.

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/render/ -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/render/
git commit -m "feat: add format string renderer with module composition"
```

---

### Task 11: Wire main.go + Integration Test

**Files:**
- Modify: `main.go`
- Create: `main_test.go`

**Step 1: Write failing integration test**

```go
package main_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	// Build the binary
	build := exec.Command("go", "build", "-o", "claude-statusline-test", ".")
	build.Dir = "."
	require.NoError(t, build.Run())
	defer os.Remove("claude-statusline-test")

	input := `{
		"model": {"display_name": "Claude Opus 4"},
		"cwd": "/home/testuser/projects/myapp",
		"cost": {"total_cost_usd": 0.42},
		"context_window": {"used_percentage": 42.5}
	}`

	cmd := exec.Command("./claude-statusline-test")
	cmd.Stdin = strings.NewReader(input)
	cmd.Env = append(os.Environ(), "HOME=/home/testuser")
	out, err := cmd.Output()
	require.NoError(t, err)

	result := string(out)
	assert.Contains(t, result, "Claude Opus 4")
	assert.Contains(t, result, "myapp")
	assert.Contains(t, result, "$0.42")
	assert.Contains(t, result, "42%")
	assert.Contains(t, result, "|")
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestIntegration`
Expected: FAIL

**Step 3: Implement main.go**

```go
package main

import (
	"fmt"
	"os"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/input"
	"github.com/felipeelias/claude-statusline/internal/render"
)

var version = "dev"

func main() {
	configPath := config.DefaultPath()

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	data, err := input.Parse(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "input error:", err)
		os.Exit(1)
	}

	output, err := render.Render(cfg, data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render error:", err)
		os.Exit(1)
	}

	fmt.Print(output)
}
```

**Step 4: Run all tests**

Run: `go test ./... -v -race`
Expected: all PASS

**Step 5: Commit**

```bash
git add main.go main_test.go
git commit -m "feat: wire main.go with full rendering pipeline"
```

---

### Task 12: Release Infrastructure

**Files:**
- Create: `.goreleaser.yml`
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/release.yml`
- Create: `.golangci.yml`
- Create: `config.example.toml`
- Create: `LICENSE`

**Step 1: Create .goreleaser.yml**

Same pattern as claude-notifier, with `brews` section targeting `felipeelias/homebrew-tap`.

```yaml
version: 2

builds:
  - main: .
    binary: claude-statusline
    ldflags:
      - -s -w -X main.version={{.Version}}
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64

brews:
  - name: claude-statusline
    repository:
      owner: felipeelias
      name: homebrew-tap
      token: "{{ .Env.TAP_GITHUB_TOKEN }}"
    homepage: https://github.com/felipeelias/claude-statusline
    description: Configurable status line for Claude Code
    license: MIT

checksum:
  name_template: checksums.txt

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^ci:"
```

**Step 2: Create CI workflow**

Same as claude-notifier: test on ubuntu+macos, lint Go, lint markdown, lint YAML.

**Step 3: Create release workflow**

Same as claude-notifier: trigger on `v*` tags, goreleaser + attestation.

**Step 4: Create .golangci.yml**

Copy from claude-notifier.

**Step 5: Create config.example.toml**

```toml
# claude-statusline configuration
# Place this file at ~/.config/claude-statusline/config.toml

# Module order and separators
format = "$model | $directory | $cost | $context"

# Active palette name
palette = "default"

# Color palettes (define your own or override default)
[palettes.default]
accent = "cyan"
cost_ok = "green"
cost_warn = "yellow"
cost_high = "red"
ctx_ok = "green"
ctx_warn = "yellow"
ctx_high = "red"

[model]
# format = "{{.DisplayName}}"
# style = "bold"

[directory]
# format = "{{.Dir}}"
# style = "palette:accent"
# truncation_length = 3

[cost]
# format = "${{printf \"%.2f\" .TotalCostUSD}}"
# style = "palette:cost_ok"
# thresholds = [
#   { above = 1.0, style = "palette:cost_warn" },
#   { above = 5.0, style = "palette:cost_high" },
# ]

[context]
# format = "{{.Bar}} {{printf \"%.0f\" .UsedPct}}%"
# style = "palette:ctx_ok"
# bar_width = 5
# bar_fill = "█"
# bar_empty = "░"
# thresholds = [
#   { above = 50, style = "palette:ctx_warn" },
#   { above = 70, style = "208" },
#   { above = 90, style = "palette:ctx_high" },
# ]

[git_branch]
disabled = true
# format = " {{.Branch}}"
# style = "palette:accent"

[session_timer]
disabled = true
# format = "{{.Elapsed}}"
# style = "dim"

[lines_changed]
disabled = true
# format = "+{{.Added}} -{{.Removed}}"
# added_style = "green"
# removed_style = "red"
```

**Step 6: Create LICENSE (MIT)**

**Step 7: Commit**

```bash
git add .goreleaser.yml .github/ .golangci.yml config.example.toml LICENSE
git commit -m "feat: add release infrastructure and example config"
```

---

### Task 13: README

**Files:**
- Create: `README.md`

Contents:
- What it is (one-liner)
- Screenshot/example output
- Installation (`brew install felipeelias/tap/claude-statusline` + `go install`)
- Configuration (Claude Code settings.json + TOML config)
- Modules table (name, description, default enabled)
- Style system reference
- Theme examples section (Tokyo Night, Gruvbox, Catppuccin powerline configs as copy-paste TOML)
- Powerline setup example

**Step 1: Write README**

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README with installation and configuration guide"
```
