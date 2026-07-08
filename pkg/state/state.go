package state

type StdinPayload struct {
	Model          ModelInfo     `json:"model"`
	ConversationID string        `json:"conversation_id"`
	ContextWindow  ContextWindow `json:"context_window"`
	TerminalWidth  int           `json:"terminal_width"`
	AgentState     string        `json:"agent_state"`
}

type ModelInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type TokenUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type ContextWindow struct {
	CurrentUsage        TokenUsage `json:"current_usage"`
	TotalInputTokens    int64      `json:"total_input_tokens"`
	TotalOutputTokens   int64      `json:"total_output_tokens"`
	ContextWindowSize   int64      `json:"context_window_size"`
	RemainingPercentage float64    `json:"remaining_percentage"`
}

type ApiUsage struct {
	GCPProjectID      string  `json:"gcp_project_id"`
	LastPollTime      string  `json:"last_poll_time"`
	Status            string  `json:"status"`
	ErrorMessage      string  `json:"error_message"`
	TodayCostUSD      float64 `json:"today_cost_usd"`
	TodayInputTokens  int64   `json:"today_input_tokens"`
	TodayOutputTokens int64   `json:"today_output_tokens"`
}
