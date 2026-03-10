package modules

import (
	"testing"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/input"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectoryModule_Name(t *testing.T) {
	m := DirectoryModule{}
	assert.Equal(t, "directory", m.Name())
}

func TestDirectoryModule_Render(t *testing.T) {
	cfg := config.Default()

	t.Run("happy path with tilde substitution and truncation", func(t *testing.T) {
		data := input.Data{
			Cwd: "/home/user/a/very/deep/nested/path",
		}

		// TruncationLength default is 3
		result, err := DirectoryModule{homeDir: "/home/user"}.Render(data, cfg)
		require.NoError(t, err)
		// With truncation length 3, keep last 3 segments fully, abbreviate earlier ones
		// segments: [a, very, deep, nested, path], keep last 3: [deep, nested, path]
		// abbreviate [a, very] → [a, v]
		// Result: ~/a/v/deep/nested/path
		assert.Contains(t, result, "~/a/v/deep/nested/path")
	})

	t.Run("empty cwd returns empty string", func(t *testing.T) {
		data := input.Data{Cwd: ""}

		result, err := DirectoryModule{}.Render(data, cfg)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("home directory alone becomes tilde", func(t *testing.T) {
		data := input.Data{Cwd: "/home/user"}

		result, err := DirectoryModule{homeDir: "/home/user"}.Render(data, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, "~")
	})

	t.Run("short path no truncation needed", func(t *testing.T) {
		data := input.Data{Cwd: "/home/user/projects"}

		result, err := DirectoryModule{homeDir: "/home/user"}.Render(data, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, "~/projects")
	})

	t.Run("truncation length 2", func(t *testing.T) {
		customCfg := config.Default()
		customCfg.Directory.TruncationLength = 2

		data := input.Data{
			Cwd: "/home/user/a/very/deep/nested/path",
		}

		result, err := DirectoryModule{homeDir: "/home/user"}.Render(data, customCfg)
		require.NoError(t, err)
		// Keep last 2 segments, abbreviate the rest
		// ~/a/very/deep/nested/path → ~/a/v/d/n/nested/path
		// Wait: segments after ~ are [a, very, deep, nested, path], keep last 2: [nested, path]
		// abbreviate [a, very, deep] to [a, v, d]
		assert.Contains(t, result, "~/a/v/d/nested/path")
	})

	t.Run("path outside home directory", func(t *testing.T) {
		data := input.Data{Cwd: "/var/log"}

		result, err := DirectoryModule{homeDir: "/home/user"}.Render(data, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, "/var/log")
	})

	t.Run("style is applied", func(t *testing.T) {
		data := input.Data{Cwd: "/home/user/project"}

		result, err := DirectoryModule{homeDir: "/home/user"}.Render(data, cfg)
		require.NoError(t, err)
		// Default style is "palette:accent" -> "cyan" -> ANSI 36
		assert.Contains(t, result, "\033[36m")
	})
}
