package style

import (
	"fmt"
	"strconv"
	"strings"
)

var namedColors = map[string]int{
	"black":   30,
	"red":     31,
	"green":   32,
	"yellow":  33,
	"blue":    34,
	"magenta": 35,
	"cyan":    36,
	"white":   37,
}

var attributes = map[string]int{
	"bold":      1,
	"dim":       2,
	"italic":    3,
	"underline": 4,
}

// Style holds parsed ANSI codes and can wrap text with them.
type Style struct {
	codes []string
}

// Parse parses a starship-like style string into a Style.
func Parse(s string) Style {
	s = strings.TrimSpace(s)
	if s == "" {
		return Style{}
	}

	var codes []string
	tokens := strings.Fields(s)

	for _, token := range tokens {
		if c, ok := parseToken(token); ok {
			codes = append(codes, c...)
		}
	}

	return Style{codes: codes}
}

func parseToken(token string) ([]string, bool) {
	// Check attributes (bold, dim, italic, underline)
	if code, ok := attributes[token]; ok {
		return []string{strconv.Itoa(code)}, true
	}

	// Check bare named colors (red, green, etc.)
	if code, ok := namedColors[token]; ok {
		return []string{strconv.Itoa(code)}, true
	}

	// Check fg: prefix
	if strings.HasPrefix(token, "fg:") {
		value := token[3:]
		return parseFg(value)
	}

	// Check bg: prefix
	if strings.HasPrefix(token, "bg:") {
		value := token[3:]
		return parseBg(value)
	}

	// Check 256-color (bare number)
	if n, err := strconv.Atoi(token); err == nil && n >= 0 && n <= 255 {
		return []string{fmt.Sprintf("38;5;%d", n)}, true
	}

	return nil, false
}

func parseFg(value string) ([]string, bool) {
	// fg:#RRGGBB
	if strings.HasPrefix(value, "#") {
		codes, ok := parseHexColor(value, 38)
		return codes, ok
	}
	// fg:red (named)
	if code, ok := namedColors[value]; ok {
		return []string{strconv.Itoa(code)}, true
	}
	return nil, false
}

func parseBg(value string) ([]string, bool) {
	// bg:#RRGGBB
	if strings.HasPrefix(value, "#") {
		codes, ok := parseHexColor(value, 48)
		return codes, ok
	}
	// bg:red (named, fg code + 10)
	if code, ok := namedColors[value]; ok {
		return []string{strconv.Itoa(code + 10)}, true
	}
	return nil, false
}

func parseHexColor(hex string, base int) ([]string, bool) {
	if len(hex) != 7 || hex[0] != '#' {
		return nil, false
	}
	r, err := strconv.ParseUint(hex[1:3], 16, 8)
	if err != nil {
		return nil, false
	}
	g, err := strconv.ParseUint(hex[3:5], 16, 8)
	if err != nil {
		return nil, false
	}
	b, err := strconv.ParseUint(hex[5:7], 16, 8)
	if err != nil {
		return nil, false
	}
	return []string{fmt.Sprintf("%d;2;%d;%d;%d", base, r, g, b)}, true
}

// Wrap wraps text with ANSI escape codes. If no codes are set, returns text unchanged.
func (s Style) Wrap(text string) string {
	if len(s.codes) == 0 {
		return text
	}
	return fmt.Sprintf("\033[%sm%s\033[0m", strings.Join(s.codes, ";"), text)
}
