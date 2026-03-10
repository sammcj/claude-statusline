package modules

import (
	"testing"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/input"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextModule_Name(t *testing.T) {
	m := ContextModule{}
	assert.Equal(t, "context", m.Name())
}

func TestContextModule_Render(t *testing.T) {
	cfg := config.Default()

	t.Run("happy path with usage", func(t *testing.T) {
		data := input.Data{
			ContextWindow: input.ContextWindow{
				UsedPercentage: 40.0,
			},
		}

		result, err := ContextModule{}.Render(data, cfg)
		require.NoError(t, err)
		// 40% of 5 bar width = 2 filled, 3 empty
		assert.Contains(t, result, "40%")
		// Default bar chars: fill=█ empty=░
		assert.Contains(t, result, "\u2588\u2588\u2591\u2591\u2591")
		// palette:ctx_ok -> green -> ANSI 32
		assert.Contains(t, result, "\033[32m")
	})

	t.Run("zero usage", func(t *testing.T) {
		data := input.Data{
			ContextWindow: input.ContextWindow{
				UsedPercentage: 0,
			},
		}

		result, err := ContextModule{}.Render(data, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, "0%")
		// All empty bars
		assert.Contains(t, result, "\u2591\u2591\u2591\u2591\u2591")
	})

	t.Run("full usage", func(t *testing.T) {
		data := input.Data{
			ContextWindow: input.ContextWindow{
				UsedPercentage: 100.0,
			},
		}

		result, err := ContextModule{}.Render(data, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, "100%")
		// All filled bars
		assert.Contains(t, result, "\u2588\u2588\u2588\u2588\u2588")
	})

	t.Run("threshold above 50 uses warning style", func(t *testing.T) {
		data := input.Data{
			ContextWindow: input.ContextWindow{
				UsedPercentage: 60.0,
			},
		}

		result, err := ContextModule{}.Render(data, cfg)
		require.NoError(t, err)
		// palette:ctx_warn -> yellow -> ANSI 33
		assert.Contains(t, result, "\033[33m")
	})

	t.Run("threshold above 70 uses 208 style", func(t *testing.T) {
		data := input.Data{
			ContextWindow: input.ContextWindow{
				UsedPercentage: 75.0,
			},
		}

		result, err := ContextModule{}.Render(data, cfg)
		require.NoError(t, err)
		// 208 is a 256-color: \033[38;5;208m
		assert.Contains(t, result, "\033[38;5;208m")
	})

	t.Run("threshold above 90 uses high style", func(t *testing.T) {
		data := input.Data{
			ContextWindow: input.ContextWindow{
				UsedPercentage: 95.0,
			},
		}

		result, err := ContextModule{}.Render(data, cfg)
		require.NoError(t, err)
		// palette:ctx_high -> red -> ANSI 31
		assert.Contains(t, result, "\033[31m")
	})

	t.Run("no threshold matches uses base style", func(t *testing.T) {
		data := input.Data{
			ContextWindow: input.ContextWindow{
				UsedPercentage: 30.0,
			},
		}

		result, err := ContextModule{}.Render(data, cfg)
		require.NoError(t, err)
		// palette:ctx_ok -> green -> ANSI 32
		assert.Contains(t, result, "\033[32m")
	})
}
