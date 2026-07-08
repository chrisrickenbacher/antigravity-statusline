package state

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalStdinPayload(t *testing.T) {
	inputJSON := `{
		"model": {
			"id": "gemini-3.5-flash",
			"display_name": "Gemini 3.5 Flash (High)"
		},
		"conversation_id": "3e6e3e21-2a39-410e-a8a4-89e4658d9114",
		"context_window": {
			"current_usage": {
				"input_tokens": 1200,
				"output_tokens": 250
			},
			"total_input_tokens": 45000,
			"total_output_tokens": 3100,
			"context_window_size": 1000000,
			"remaining_percentage": 95.19
		},
		"terminal_width": 120,
		"agent_state": "idle"
	}`

	var payload StdinPayload
	err := json.Unmarshal([]byte(inputJSON), &payload)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if payload.Model.ID != "gemini-3.5-flash" {
		t.Errorf("Expected model ID 'gemini-3.5-flash', got '%s'", payload.Model.ID)
	}

	if payload.Model.DisplayName != "Gemini 3.5 Flash (High)" {
		t.Errorf("Expected display name 'Gemini 3.5 Flash (High)', got '%s'", payload.Model.DisplayName)
	}

	if payload.ConversationID != "3e6e3e21-2a39-410e-a8a4-89e4658d9114" {
		t.Errorf("Expected conversation ID '3e6e3e21-2a39-410e-a8a4-89e4658d9114', got '%s'", payload.ConversationID)
	}

	if payload.ContextWindow.CurrentUsage.InputTokens != 1200 {
		t.Errorf("Expected current input tokens 1200, got %d", payload.ContextWindow.CurrentUsage.InputTokens)
	}

	if payload.ContextWindow.CurrentUsage.OutputTokens != 250 {
		t.Errorf("Expected current output tokens 250, got %d", payload.ContextWindow.CurrentUsage.OutputTokens)
	}

	if payload.ContextWindow.TotalInputTokens != 45000 {
		t.Errorf("Expected total input tokens 45000, got %d", payload.ContextWindow.TotalInputTokens)
	}

	if payload.ContextWindow.TotalOutputTokens != 3100 {
		t.Errorf("Expected total output tokens 3100, got %d", payload.ContextWindow.TotalOutputTokens)
	}

	if payload.ContextWindow.ContextWindowSize != 1000000 {
		t.Errorf("Expected context window size 1000000, got %d", payload.ContextWindow.ContextWindowSize)
	}

	expectedPct := 95.19
	if payload.ContextWindow.RemainingPercentage != expectedPct {
		t.Errorf("Expected remaining percentage %f, got %f", expectedPct, payload.ContextWindow.RemainingPercentage)
	}

	if payload.TerminalWidth != 120 {
		t.Errorf("Expected terminal width 120, got %d", payload.TerminalWidth)
	}

	if payload.AgentState != "idle" {
		t.Errorf("Expected agent state 'idle', got '%s'", payload.AgentState)
	}
}
