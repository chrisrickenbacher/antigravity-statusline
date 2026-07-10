package layout

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/chrisrickenbacher/antigravity-statusline/pkg/pricing"
	"github.com/chrisrickenbacher/antigravity-statusline/pkg/state"
)

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{250, "250"},
		{999, "999"},
		{1000, "1K"},
		{1200, "1.2K"},
		{1250, "1.25K"},
		{45000, "45K"},
	}

	for _, tc := range tests {
		res := FormatTokens(tc.input)
		if res != tc.expected {
			t.Errorf("FormatTokens(%d) expected '%s', got '%s'", tc.input, tc.expected, res)
		}
	}
}

func TestShortenModelName(t *testing.T) {
	tests := []struct {
		displayName string
		modelID     string
		expected    string
	}{
		{"Gemini 3.5 Flash (High)", "gemini-3.5-flash", "gemini-3.5-flash"},
		{"Gemini 1.5 Pro", "gemini-1.5-pro", "gemini-1.5-pro"},
		{"Gemini 4.0 Ultra", "gemini-4.0-ultra", "gemini-4.0-ultra"},
		{"Custom Model", "custom-model", "custom-model"},
		{"Custom Model Only", "", "Custom Model Only"},
		{"", "", "unknown"},
	}

	for _, tc := range tests {
		res := ShortenModelName(tc.displayName, tc.modelID)
		if res != tc.expected {
			t.Errorf("ShortenModelName(%q, %q) expected '%s', got '%s'", tc.displayName, tc.modelID, tc.expected, res)
		}
	}
}

func TestRenderStatusLine(t *testing.T) {
	mockPayload := &state.StdinPayload{
		Model: state.ModelInfo{
			ID:          "gemini-3.5-flash",
			DisplayName: "Gemini 3.5 Flash (High)",
		},
		ConversationID: "3e6e3e21-2a39-410e-a8a4-89e4658d9114",
		ContextWindow: state.ContextWindow{
			CurrentUsage: state.TokenUsage{
				InputTokens:  1200,
				OutputTokens: 250,
			},
			TotalInputTokens:    45000,
			TotalOutputTokens:   3100,
			ContextWindowSize:   1000000,
			RemainingPercentage: 95.19,
		},
		TerminalWidth: 120,
		AgentState:     "idle",
	}

	mockPriceCache := &pricing.PricingCache{
		LastFetched: "2026-07-08T00:00:00Z",
		Models: map[string]pricing.ModelRate{
			"gemini-3.5-flash": {
				InputPricePer1M:  0.075,
				OutputPricePer1M: 0.300,
			},
		},
	}

	now := time.Date(2026, 7, 8, 9, 32, 0, 0, time.UTC)

	t.Run("Case 1: Healthy & Synchronized (Default State)", func(t *testing.T) {
		apiUsage := &state.ApiUsage{
			GCPProjectID:      "local-usage",
			LastPollTime:      "2026-07-08T09:30:00Z", // 2 minutes ago
			Status:            "success",
			ErrorMessage:      "",
			TodayCostUSD:      0.12,
			TodayInputTokens:  1200000,
			TodayOutputTokens: 250000,
		}

		// 1. WIDE
		mockPayload.TerminalWidth = 120
		resWide := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		expectedWide := "\033[32m🟢 idle\033[0m │ gemini-3.5-flash │ Turn: +1.2K/250 (~$0.0002) │ Sess: 45K/3.1K (~$0.0043) │ Today: ~$0.12 │ Ctx: 4.8%"
		if resWide != expectedWide {
			t.Errorf("WIDE expected:\n%q\ngot:\n%q", expectedWide, resWide)
		}

		// 1.5. WIDTH 110 (Wide without Ctx)
		mockPayload.TerminalWidth = 110
		resWide110 := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		expectedWide110 := "\033[32m🟢 idle\033[0m │ gemini-3.5-flash │ Turn: +1.2K/250 (~$0.0002) │ Sess: 45K/3.1K (~$0.0043) │ Today: ~$0.12"
		if resWide110 != expectedWide110 {
			t.Errorf("WIDTH 110 expected:\n%q\ngot:\n%q", expectedWide110, resWide110)
		}

		// 2. STANDARD
		mockPayload.TerminalWidth = 100
		resStd := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		expectedStd := "\033[32m🟢 idle\033[0m │ gemini-3.5-flash │ Sess: 45K/3.1K (~$0.004) │ Today: ~$0.12"
		if resStd != expectedStd {
			t.Errorf("STANDARD expected:\n%q\ngot:\n%q", expectedStd, resStd)
		}

		// 3. COMPACT
		mockPayload.TerminalWidth = 70
		resCmp := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		expectedCmp := "\033[32m🟢 idle\033[0m │ Sess: 45K │ Today: ~$0.12"
		if resCmp != expectedCmp {
			t.Errorf("COMPACT expected:\n%q\ngot:\n%q", expectedCmp, resCmp)
		}

		// 4. MINIMAL
		mockPayload.TerminalWidth = 50
		resMin := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		expectedMin := "\033[32m🟢 idle\033[0m │ Today: ~$0.12"
		if resMin != expectedMin {
			t.Errorf("MINIMAL expected:\n%q\ngot:\n%q", expectedMin, resMin)
		}
	})

	t.Run("Case 2: Stale Cache Warning (Daemon Dead)", func(t *testing.T) {
		apiUsage := &state.ApiUsage{
			GCPProjectID:      "local-usage",
			LastPollTime:      "2026-07-08T09:25:00Z", // 7 minutes ago (>= 5m)
			Status:            "success",
			ErrorMessage:      "",
			TodayCostUSD:      0.12,
			TodayInputTokens:  1200000,
			TodayOutputTokens: 250000,
		}

		// 1. WIDE
		mockPayload.TerminalWidth = 120
		resWide := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		if !strings.Contains(resWide, "Today: ~$0.12 [Daemon Dead]") {
			t.Errorf("Expected Daemon Dead warning in WIDE, got: %q", resWide)
		}

		// 2. STANDARD
		mockPayload.TerminalWidth = 100
		resStd := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		if !strings.Contains(resStd, "Today: ~$0.12 [Daemon Dead]") {
			t.Errorf("Expected Daemon Dead warning in STANDARD, got: %q", resStd)
		}

		// 3. COMPACT
		mockPayload.TerminalWidth = 70
		resCmp := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		if !strings.Contains(resCmp, "Today: ~$0.12 [Daemon Dead]") {
			t.Errorf("Expected Daemon Dead warning in COMPACT, got: %q", resCmp)
		}

		// 4. MINIMAL
		mockPayload.TerminalWidth = 50
		resMin := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		if !strings.Contains(resMin, "~$0.12 [Daemon Dead]") {
			t.Errorf("Expected Daemon Dead warning in MINIMAL, got: %q", resMin)
		}
	})

	t.Run("Case 3: Missing Pricing Cache", func(t *testing.T) {
		apiUsage := &state.ApiUsage{
			GCPProjectID:      "local-usage",
			LastPollTime:      "2026-07-08T09:30:00Z",
			Status:            "success",
			ErrorMessage:      "",
			TodayCostUSD:      0.12,
			TodayInputTokens:  1200000,
			TodayOutputTokens: 250000,
		}

		// WIDE with nil priceCache
		mockPayload.TerminalWidth = 120
		resWide := RenderStatusLine(mockPayload, nil, apiUsage, nil, now)
		expectedWide := "\033[32m🟢 idle\033[0m │ gemini-3.5-flash │ Turn: +1.2K/250 [No Pricing] │ Sess: 45K/3.1K [No Pricing] │ Today: [No Pricing] │ Ctx: 4.8%"
		if resWide != expectedWide {
			t.Errorf("WIDE expected:\n%q\ngot:\n%q", expectedWide, resWide)
		}
	})

	t.Run("Case 4: Authentication Failure", func(t *testing.T) {
		apiUsage := &state.ApiUsage{
			GCPProjectID:      "local-usage",
			LastPollTime:      "2026-07-08T09:30:00Z",
			Status:            "auth_error",
			ErrorMessage:      "GCP credentials missing or expired",
			TodayCostUSD:      0.12,
			TodayInputTokens:  1200000,
			TodayOutputTokens: 250000,
		}

		mockPayload.TerminalWidth = 120
		resWide := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		if !strings.Contains(resWide, "Today: ~$0.12 [Auth Err]") {
			t.Errorf("Expected Auth Err in Today, got: %q", resWide)
		}

		mockPayload.TerminalWidth = 50
		resMin := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		if !strings.Contains(resMin, "~$0.12 [Auth Err]") {
			t.Errorf("Expected Auth Err in MINIMAL, got: %q", resMin)
		}
	})

	t.Run("Case 5: General Daemon Error", func(t *testing.T) {
		apiUsage := &state.ApiUsage{
			GCPProjectID:      "local-usage",
			LastPollTime:      "2026-07-08T09:30:00Z",
			Status:            "error",
			ErrorMessage:      "failed to parse session log",
			TodayCostUSD:      0.12,
			TodayInputTokens:  1200000,
			TodayOutputTokens: 250000,
		}

		mockPayload.TerminalWidth = 120
		resWide := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		if !strings.Contains(resWide, "Today: ~$0.12 [Daemon Err]") {
			t.Errorf("Expected General Daemon Err in Today, got: %q", resWide)
		}

		mockPayload.TerminalWidth = 50
		resMin := RenderStatusLine(mockPayload, mockPriceCache, apiUsage, nil, now)
		if !strings.Contains(resMin, "~$0.12 [Daemon Err]") {
			t.Errorf("Expected General Daemon Err in MINIMAL, got: %q", resMin)
		}
	})

	t.Run("Case 6: No Daily Metrics Cache", func(t *testing.T) {
		mockPayload.TerminalWidth = 120
		resWide := RenderStatusLine(mockPayload, mockPriceCache, nil, errors.New("file not found"), now)
		if !strings.Contains(resWide, "Today: [No Cache]") {
			t.Errorf("Expected [No Cache] in WIDE, got: %q", resWide)
		}

		mockPayload.TerminalWidth = 50
		resMin := RenderStatusLine(mockPayload, mockPriceCache, nil, errors.New("file not found"), now)
		if !strings.Contains(resMin, "[No Cache]") {
			t.Errorf("Expected [No Cache] in MINIMAL, got: %q", resMin)
		}
	})
}
