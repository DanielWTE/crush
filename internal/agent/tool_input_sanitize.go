package agent

import (
	"encoding/json"

	"charm.land/fantasy"
)

// sanitizeMalformedToolCallMessages replaces malformed tool call arguments in
// the provider-bound copy of a conversation. Fantasy retains the original
// ToolCallPart between steps, so sanitizing only Crush's persisted message is
// not enough to keep invalid JSON out of the next provider request.
func sanitizeMalformedToolCallMessages(messages []fantasy.Message) ([]fantasy.Message, int) {
	sanitized := cloneFantasyMessages(messages)
	sanitizedCount := 0

	for messageIndex := range sanitized {
		for partIndex, part := range sanitized[messageIndex].Content {
			toolCall, ok := fantasy.AsMessagePart[fantasy.ToolCallPart](part)
			if !ok || json.Valid([]byte(toolCall.Input)) {
				continue
			}

			toolCall.Input = "{}"
			sanitized[messageIndex].Content[partIndex] = toolCall
			sanitizedCount++
		}
	}

	return sanitized, sanitizedCount
}
