package mcp

import (
	"slices"
	"strings"
	"unicode"

	"github.com/charmbracelet/crush/internal/config"
)

var activationWords = map[string]struct{}{
	"analyse":    {},
	"analyze":    {},
	"analysiere": {},
	"ask":        {},
	"benutze":    {},
	"benutzen":   {},
	"check":      {},
	"checke":     {},
	"durchsuche": {},
	"frage":      {},
	"mit":        {},
	"mcp":        {},
	"nutz":       {},
	"nutze":      {},
	"nutzen":     {},
	"prüfe":      {},
	"prüfen":     {},
	"query":      {},
	"search":     {},
	"suche":      {},
	"through":    {},
	"use":        {},
	"using":      {},
	"verwende":   {},
	"verwenden":  {},
	"via":        {},
	"with":       {},
	"überprüfe":  {},
	"überprüfen": {},
}

var negationWords = map[string]struct{}{
	"kein":    {},
	"keine":   {},
	"keinen":  {},
	"nicht":   {},
	"no":      {},
	"not":     {},
	"ohne":    {},
	"without": {},
}

var negationResetWords = map[string]struct{}{
	"aber":    {},
	"but":     {},
	"instead": {},
	"sondern": {},
}

// MatchOnDemand returns the configured on-demand MCP servers explicitly
// requested by a natural-language prompt. Exact aliases and small typos are
// accepted, but only alongside an activation word such as "use", "nutze",
// "with", or "MCP" so an incidental mention does not start a server.
func MatchOnDemand(prompt string, configured config.MCPs) []string {
	tokens := promptTokens(prompt)
	if !containsActivationWord(tokens) {
		return nil
	}

	var matched []string
	for name, m := range configured {
		if m.Disabled || !m.OnDemand {
			continue
		}

		aliases := append([]string{name}, m.Aliases...)
		if matchesAlias(tokens, aliases) {
			matched = append(matched, name)
		}
	}
	slices.Sort(matched)
	return matched
}

// OnDemandNames returns every enabled MCP configured for cold loading.
func OnDemandNames(configured config.MCPs) []string {
	names := make([]string, 0)
	for name, m := range configured {
		if m.OnDemand && !m.Disabled {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

func promptTokens(value string) []string {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, "don't", "do not")
	value = strings.ReplaceAll(value, "dont", "do not")

	return strings.FieldsFunc(value, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

func containsActivationWord(tokens []string) bool {
	for _, token := range tokens {
		if _, ok := activationWords[token]; ok {
			return true
		}
	}
	return false
}

func matchesAlias(tokens, aliases []string) bool {
	for _, alias := range aliases {
		aliasTokens := promptTokens(alias)
		if len(aliasTokens) == 0 || len(aliasTokens) > len(tokens) {
			continue
		}
		for start := 0; start <= len(tokens)-len(aliasTokens); start++ {
			if isNegated(tokens, start) {
				continue
			}
			if fuzzyTokenSequence(tokens[start:start+len(aliasTokens)], aliasTokens) {
				return true
			}
		}
	}
	return false
}

func isNegated(tokens []string, start int) bool {
	from := max(0, start-3)
	for i := start - 1; i >= from; i-- {
		token := tokens[i]
		if _, ok := negationResetWords[token]; ok {
			return false
		}
		if _, ok := negationWords[token]; ok {
			return true
		}
	}
	return false
}

func fuzzyTokenSequence(actual, expected []string) bool {
	for i := range expected {
		if !fuzzyTokenEqual(actual[i], expected[i]) {
			return false
		}
	}
	return true
}

func fuzzyTokenEqual(actual, expected string) bool {
	if actual == expected {
		return true
	}

	expectedRunes := []rune(expected)
	actualRunes := []rune(actual)
	maxDistance := 0
	switch {
	case len(expectedRunes) >= 9:
		maxDistance = 2
	case len(expectedRunes) >= 5:
		maxDistance = 1
	}
	if maxDistance == 0 || len(actualRunes) < 2 || len(expectedRunes) < 2 ||
		actualRunes[0] != expectedRunes[0] || actualRunes[1] != expectedRunes[1] {
		return false
	}
	return damerauLevenshtein(actualRunes, expectedRunes) <= maxDistance
}

// damerauLevenshtein computes optimal-string-alignment distance so a single
// transposition, such as "sisrtix", counts as one typo.
func damerauLevenshtein(a, b []rune) int {
	rows := len(a) + 1
	cols := len(b) + 1
	distance := make([][]int, rows)
	for i := range rows {
		distance[i] = make([]int, cols)
		distance[i][0] = i
	}
	for j := range cols {
		distance[0][j] = j
	}

	for i := 1; i < rows; i++ {
		for j := 1; j < cols; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			distance[i][j] = min(
				distance[i-1][j]+1,
				distance[i][j-1]+1,
				distance[i-1][j-1]+cost,
			)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				distance[i][j] = min(distance[i][j], distance[i-2][j-2]+1)
			}
		}
	}
	return distance[len(a)][len(b)]
}
