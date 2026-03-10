package modules

import (
	"testing"

	"github.com/felipeelias/claude-statusline/internal/config"
	"github.com/felipeelias/claude-statusline/internal/input"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCostModule_Name(t *testing.T) {
	m := CostModule{}
	assert.Equal(t, "cost", m.Name())
}

func TestCostModule_Render(t *testing.T) {
	cfg := config.Default()

	t.Run("happy path with cost", func(t *testing.T) {
		data := input.Data{
			Cost: input.Cost{TotalCostUSD: 0.42},
		}

		result, err := CostModule{}.Render(data, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, "$0.42")
		// Default style is palette:cost_ok -> green -> ANSI 32
		assert.Contains(t, result, "\033[32m")
	})

	t.Run("zero cost", func(t *testing.T) {
		data := input.Data{
			Cost: input.Cost{TotalCostUSD: 0},
		}

		result, err := CostModule{}.Render(data, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, "$0.00")
	})

	t.Run("threshold above 1.0 uses warning style", func(t *testing.T) {
		data := input.Data{
			Cost: input.Cost{TotalCostUSD: 2.50},
		}

		result, err := CostModule{}.Render(data, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, "$2.50")
		// palette:cost_warn -> yellow -> ANSI 33
		assert.Contains(t, result, "\033[33m")
	})

	t.Run("threshold above 5.0 uses high style", func(t *testing.T) {
		data := input.Data{
			Cost: input.Cost{TotalCostUSD: 10.00},
		}

		result, err := CostModule{}.Render(data, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, "$10.00")
		// palette:cost_high -> red -> ANSI 31
		assert.Contains(t, result, "\033[31m")
	})

	t.Run("no threshold matches uses base style", func(t *testing.T) {
		data := input.Data{
			Cost: input.Cost{TotalCostUSD: 0.50},
		}

		result, err := CostModule{}.Render(data, cfg)
		require.NoError(t, err)
		// palette:cost_ok -> green -> ANSI 32
		assert.Contains(t, result, "\033[32m")
	})
}
