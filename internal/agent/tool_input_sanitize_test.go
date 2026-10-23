package agent

import (
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

func TestSanitizeMalformedToolCallMessages(t *testing.T) {
	t.Parallel()

	malformedInput := `{"content":"unfinished`
	messages := []fantasy.Message{
		{
			Role: fantasy.MessageRoleAssistant,
			Content: []fantasy.MessagePart{
				fantasy.ToolCallPart{
					ToolCallID: "bad-call",
					ToolName:   "write",
					Input:      malformedInput,
				},
				fantasy.ToolCallPart{
					ToolCallID: "good-call",
					ToolName:   "bash",
					Input:      `{"command":"pwd"}`,
				},
			},
		},
	}

	sanitized, count := sanitizeMalformedToolCallMessages(messages)

	require.Equal(t, 1, count)
	badCall, ok := fantasy.AsMessagePart[fantasy.ToolCallPart](sanitized[0].Content[0])
	require.True(t, ok)
	require.Equal(t, "{}", badCall.Input)
	goodCall, ok := fantasy.AsMessagePart[fantasy.ToolCallPart](sanitized[0].Content[1])
	require.True(t, ok)
	require.Equal(t, `{"command":"pwd"}`, goodCall.Input)

	// Provider preparation must not mutate Fantasy's retained step messages.
	originalBadCall, ok := fantasy.AsMessagePart[fantasy.ToolCallPart](messages[0].Content[0])
	require.True(t, ok)
	require.Equal(t, malformedInput, originalBadCall.Input)
}

func TestSanitizeMalformedToolCallMessagesHandlesPointerParts(t *testing.T) {
	t.Parallel()

	messages := []fantasy.Message{
		{
			Role: fantasy.MessageRoleAssistant,
			Content: []fantasy.MessagePart{
				&fantasy.ToolCallPart{
					ToolCallID: "pointer-call",
					ToolName:   "write",
					Input:      `{"content":`,
				},
			},
		},
	}

	sanitized, count := sanitizeMalformedToolCallMessages(messages)

	require.Equal(t, 1, count)
	toolCall, ok := fantasy.AsMessagePart[fantasy.ToolCallPart](sanitized[0].Content[0])
	require.True(t, ok)
	require.Equal(t, "{}", toolCall.Input)
}
