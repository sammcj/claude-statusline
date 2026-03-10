package modules

import (
	"os"
	"strings"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/input"
)

// DirectoryModule renders the current working directory with tilde substitution and truncation.
type DirectoryModule struct {
	// homeDir overrides the home directory for testing. If empty, os.UserHomeDir() is used.
	homeDir string
}

// NewDirectoryModule creates a DirectoryModule that uses the real home directory.
func NewDirectoryModule() DirectoryModule {
	home, _ := os.UserHomeDir()
	return DirectoryModule{homeDir: home}
}

func (DirectoryModule) Name() string { return "directory" }

func (m DirectoryModule) Render(data input.Data, cfg config.Config) (string, error) {
	cwd := data.Cwd
	if cwd == "" {
		return "", nil
	}

	home := m.homeDir
	if home == "" {
		home, _ = os.UserHomeDir()
	}

	// Tilde substitution
	dir := cwd
	if home != "" {
		if dir == home {
			dir = "~"
		} else if strings.HasPrefix(dir, home+"/") {
			dir = "~" + dir[len(home):]
		}
	}

	// Truncation
	dir = truncatePath(dir, cfg.Directory.TruncationLength)

	templateData := struct{ Dir string }{Dir: dir}

	result, err := renderTemplate("directory", cfg.Directory.Format, templateData)
	if err != nil {
		return "", err
	}

	return wrapStyle(result, cfg.Directory.Style, cfg), nil
}

// truncatePath keeps the last n path segments fully and abbreviates earlier ones
// to their first character. The leading "/" or "~/" prefix is preserved.
func truncatePath(path string, n int) string {
	if n <= 0 {
		return path
	}

	// Determine prefix and segments
	var prefix string
	var segmentStr string

	if strings.HasPrefix(path, "~/") {
		prefix = "~/"
		segmentStr = path[2:]
	} else if path == "~" {
		return "~"
	} else if strings.HasPrefix(path, "/") {
		prefix = "/"
		segmentStr = path[1:]
	} else {
		prefix = ""
		segmentStr = path
	}

	if segmentStr == "" {
		return prefix
	}

	segments := strings.Split(segmentStr, "/")

	if len(segments) <= n {
		return path
	}

	// Abbreviate segments before the last n
	cutoff := len(segments) - n
	for i := 0; i < cutoff; i++ {
		if len(segments[i]) > 0 {
			// Keep only the first character (first rune)
			r := []rune(segments[i])
			segments[i] = string(r[0])
		}
	}

	return prefix + strings.Join(segments, "/")
}
