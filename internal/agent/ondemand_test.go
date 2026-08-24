package agent

import (
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

type fakeMCPTool struct {
	*fakeTool
	mcpName string
}

func (f *fakeMCPTool) MCP() string {
	return f.mcpName
}

func TestFilterOnDemandTools(t *testing.T) {
	t.Parallel()

	builtin := &fakeTool{name: "view"}
	always := &fakeMCPTool{fakeTool: &fakeTool{name: "mcp_always_search"}, mcpName: "always"}
	sistrix := &fakeMCPTool{fakeTool: &fakeTool{name: "mcp_sistrix_search"}, mcpName: "sistrix"}
	rybbit := &fakeMCPTool{fakeTool: &fakeTool{name: "mcp_rybbit_query"}, mcpName: "rybbit"}

	got := filterOnDemandTools(
		[]fantasy.AgentTool{builtin, always, sistrix, rybbit},
		[]string{"rybbit", "sistrix"},
		[]string{"sistrix"},
	)

	require.Equal(t, []fantasy.AgentTool{builtin, always, sistrix}, got)
}

func TestHookedToolPreservesMCPIdentity(t *testing.T) {
	t.Parallel()

	inner := &fakeMCPTool{fakeTool: &fakeTool{name: "mcp_sistrix_search"}, mcpName: "sistrix"}
	wrapped := newHookedTool(inner, nil)
	require.Equal(t, "sistrix", wrapped.MCP())
}

func TestDrainQueueKeepsOnDemandTurnSeparate(t *testing.T) {
	t.Parallel()

	env := testEnv(t)
	a := NewSessionAgent(SessionAgentOptions{
		Sessions: env.sessions,
		Messages: env.messages,
	}).(*sessionAgent)

	const sessionID = "on-demand-queue"
	a.messageQueue.Set(sessionID, []SessionAgentCall{
		{SessionID: sessionID, Prompt: "plain follow-up"},
		{SessionID: sessionID, Prompt: "use sistrix", ActiveOnDemand: []string{"sistrix"}},
	})

	fold, canceled := a.drainQueueForStep(sessionID)
	require.Len(t, fold, 1)
	require.Equal(t, "plain follow-up", fold[0].Prompt)
	require.Empty(t, canceled)
	queued, ok := a.messageQueue.Get(sessionID)
	require.True(t, ok)
	require.Len(t, queued, 1)
	require.Equal(t, "use sistrix", queued[0].Prompt)
}

func TestCanceledQueuedTurnReleasesOnDemandLease(t *testing.T) {
	t.Parallel()

	env := testEnv(t)
	a := NewSessionAgent(SessionAgentOptions{
		Sessions: env.sessions,
		Messages: env.messages,
	}).(*sessionAgent)

	const sessionID = "on-demand-canceled"
	released := 0
	a.messageQueue.Set(sessionID, []SessionAgentCall{{
		SessionID:       sessionID,
		Prompt:          "use sistrix",
		ActiveOnDemand:  []string{"sistrix"},
		ReleaseOnDemand: func() { released++ },
		acceptSeq:       1,
	}})
	a.cancelMark.Set(sessionID, 2)

	fold, canceled := a.drainQueueForStep(sessionID)
	require.Empty(t, fold)
	require.Empty(t, canceled)
	require.Equal(t, 1, released)
}
