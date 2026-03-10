package modules

import (
	"bytes"
	"text/template"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/style"
)

// renderTemplate executes a Go text/template with the given data and returns the result.
func renderTemplate(name, format string, data any) (string, error) {
	tmpl, err := template.New(name).Parse(format)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// wrapStyle resolves a style string via the config palette, parses it, and wraps text.
func wrapStyle(text, styleStr string, cfg config.Config) string {
	resolved := cfg.ResolveStyle(styleStr)
	return style.Parse(resolved).Wrap(text)
}

// resolveThresholdStyle evaluates thresholds in order. The last threshold whose
// Above value is less than the given value wins. If none match, the base style is used.
func resolveThresholdStyle(value float64, thresholds []config.Threshold, baseStyle string) string {
	winner := baseStyle
	for _, t := range thresholds {
		if value > t.Above {
			winner = t.Style
		}
	}
	return winner
}
