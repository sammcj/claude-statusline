package modules

import (
	"strings"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/input"
)

// ContextModule renders the context window usage with a progress bar.
type ContextModule struct{}

func (ContextModule) Name() string { return "context" }

func (ContextModule) Render(data input.Data, cfg config.Config) (string, error) {
	pct := data.ContextWindow.UsedPercentage

	barWidth := cfg.Context.BarWidth
	filled := int(pct / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}
	empty := barWidth - filled

	bar := strings.Repeat(cfg.Context.BarFill, filled) + strings.Repeat(cfg.Context.BarEmpty, empty)

	templateData := struct {
		Bar     string
		UsedPct float64
	}{
		Bar:     bar,
		UsedPct: pct,
	}

	result, err := renderTemplate("context", cfg.Context.Format, templateData)
	if err != nil {
		return "", err
	}

	winningStyle := resolveThresholdStyle(pct, cfg.Context.Thresholds, cfg.Context.Style)

	return wrapStyle(result, winningStyle, cfg), nil
}
