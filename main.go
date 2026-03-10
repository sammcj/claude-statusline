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
	cfg, err := config.Load(config.DefaultPath())
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
