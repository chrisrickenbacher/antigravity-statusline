package state

type StdinPayload struct {
	Model          ModelInfo     `json:"model"`
	ConversationID string        `json:"conversation_id"`
	ContextWindow  ContextWindow `json:"context_window"`
	TerminalWidth  int           `json:"terminal_width"`
	AgentState     string        `json:"agent_state"`
	IsOAuth        bool          `json:"is_oauth"`
	ProjectID      string        `json:"project_id"`
}

type ModelInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type TokenUsage struct {
	InputTokens       int64 `json:"input_tokens"`
	OutputTokens      int64 `json:"output_tokens"`
	CachedInputTokens int64 `json:"cache_read_input_tokens"`
}

type ContextWindow struct {
	CurrentUsage        TokenUsage `json:"current_usage"`
	TotalInputTokens    int64      `json:"total_input_tokens"`
	TotalOutputTokens   int64      `json:"total_output_tokens"`
	TotalCachedTokens   int64      `json:"total_cached_tokens"`
	ContextWindowSize   int64      `json:"context_window_size"`
	RemainingPercentage float64    `json:"remaining_percentage"`
}

type ModelUsage struct {
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CachedTokens int64   `json:"cached_tokens"`
	CostUSD      float64 `json:"cost_usd"`
}

type ProjectUsage struct {
	TodayCostUSD float64               `json:"today_cost_usd"`
	Models       map[string]ModelUsage `json:"models,omitempty"`
}

type ApiUsage struct {
	LastPollTime string                  `json:"last_poll_time"`
	Status       string                  `json:"status"`
	ErrorMessage string                  `json:"error_message,omitempty"`
	Projects     map[string]ProjectUsage `json:"projects"`
}

