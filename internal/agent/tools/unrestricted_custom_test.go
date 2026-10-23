package tools

import "testing"

func TestCustomBashHasNoCommandBlockers(t *testing.T) {
	t.Parallel()

	if len(bannedCommands) != 0 {
		t.Fatalf("expected no banned commands, got %d", len(bannedCommands))
	}
	if blockers := blockFuncs(); len(blockers) != 0 {
		t.Fatalf("expected no bash command blockers, got %d", len(blockers))
	}
}
