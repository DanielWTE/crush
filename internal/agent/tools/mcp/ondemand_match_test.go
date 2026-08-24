package mcp

import (
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/stretchr/testify/require"
)

func TestMatchOnDemand(t *testing.T) {
	t.Parallel()

	configured := config.MCPs{
		"gsc": {
			OnDemand: true,
			Aliases:  []string{"google search console"},
		},
		"rybbit": {
			OnDemand: true,
		},
		"sistrix": {
			OnDemand: true,
		},
		"verbandsstoffe": {
			OnDemand: true,
		},
		"disabled": {
			OnDemand: true,
			Disabled: true,
		},
		"always-on": {},
	}

	tests := []struct {
		name   string
		prompt string
		want   []string
	}{
		{name: "German intent", prompt: "Bitte nutze Sistrix dafür", want: []string{"sistrix"}},
		{name: "English intent", prompt: "Use sistrix for this", want: []string{"sistrix"}},
		{name: "MCP suffix", prompt: "Verbandsstoffe MCP", want: []string{"verbandsstoffe"}},
		{name: "missing letter typo", prompt: "nutze bitte sistrx", want: []string{"sistrix"}},
		{name: "transposed typo", prompt: "use sisrtix for this", want: []string{"sistrix"}},
		{name: "long-name typo", prompt: "mit verbandstofe MCP prüfen", want: []string{"verbandsstoffe"}},
		{name: "multi-word alias typo", prompt: "use gogle search console", want: []string{"gsc"}},
		{name: "multiple explicit profiles", prompt: "Nutze Sistrix und GSC MCP", want: []string{"gsc", "sistrix"}},
		{name: "incidental mention", prompt: "Was ist Sistrix?", want: nil},
		{name: "negated German", prompt: "Nutze nicht Sistrix, sondern interne Daten", want: nil},
		{name: "negated English", prompt: "Do not use sistrix for this", want: nil},
		{name: "negation switches profile", prompt: "Nutze nicht Sistrix, sondern GSC MCP", want: []string{"gsc"}},
		{name: "common-word typo collision", prompt: "Use Sistrix to research rabbit rankings", want: []string{"sistrix"}},
		{name: "disabled profile", prompt: "use disabled MCP", want: nil},
		{name: "always-on profile", prompt: "use always-on MCP", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, MatchOnDemand(tt.prompt, configured))
		})
	}
}

func TestOnDemandNames(t *testing.T) {
	t.Parallel()

	configured := config.MCPs{
		"zeta":     {OnDemand: true},
		"alpha":    {OnDemand: true},
		"disabled": {OnDemand: true, Disabled: true},
		"normal":   {},
	}

	require.Equal(t, []string{"alpha", "zeta"}, OnDemandNames(configured))
}
