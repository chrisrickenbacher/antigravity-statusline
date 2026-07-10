package layout

import (
	"fmt"
	"strings"
	"time"

	"github.com/chrisrickenbacher/antigravity-statusline/pkg/pricing"
	"github.com/chrisrickenbacher/antigravity-statusline/pkg/state"
)

func FormatTokens(tokens int64) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	val := float64(tokens) / 1000.0
	if tokens%1000 == 0 {
		return fmt.Sprintf("%.0fK", val)
	}
	if tokens%100 == 0 {
		return fmt.Sprintf("%.1fK", val)
	}
	return fmt.Sprintf("%.2fK", val)
}

func ShortenModelName(displayName string, modelID string) string {
	if modelID != "" {
		return modelID
	}
	if displayName != "" {
		return displayName
	}
	return "unknown"
}

func RenderStatusLine(
	payload *state.StdinPayload,
	priceCache *pricing.PricingCache,
	apiUsage *state.ApiUsage,
	apiUsageErr error,
	now time.Time,
) string {
	var stateColor, stateText string
	switch strings.ToLower(payload.AgentState) {
	case "idle":
		stateColor = "\033[32m"
		stateText = "🟢 idle"
	case "thinking":
		stateColor = "\033[34m"
		stateText = "🔵 thinking"
	case "executing":
		stateColor = "\033[33m"
		stateText = "🟡 executing"
	case "waiting":
		stateColor = "\033[35m"
		stateText = "🟠 waiting"
	default:
		stateColor = "\033[37m"
		stateText = "⚪ ready"
	}
	stateSegment := fmt.Sprintf("%s%s\033[0m", stateColor, stateText)

	rates, err := pricing.ResolveRates(priceCache, payload.Model.ID)
	pricingAvailable := err == nil

	var turnCostStr, sessCostStr string
	if pricingAvailable {
		turnCost := pricing.CalculateCost(payload.ContextWindow.CurrentUsage.InputTokens, payload.ContextWindow.CurrentUsage.CachedInputTokens, payload.ContextWindow.CurrentUsage.OutputTokens, rates)
		sessCost := pricing.CalculateCost(payload.ContextWindow.TotalInputTokens, payload.ContextWindow.TotalCachedTokens, payload.ContextWindow.TotalOutputTokens, rates)
		turnCostStr = fmt.Sprintf("~$%.4f", turnCost)
		sessCostStr = fmt.Sprintf("~$%.4f", sessCost)
	} else {
		turnCostStr = "[No Pricing]"
		sessCostStr = "[No Pricing]"
	}

	var todaySegment string
	if apiUsageErr != nil {
		if payload.TerminalWidth < 60 {
			todaySegment = "[No Cache]"
		} else {
			todaySegment = "Today: [No Cache]"
		}
	} else if !pricingAvailable {
		if payload.TerminalWidth < 60 {
			todaySegment = "[No Pricing]"
		} else {
			todaySegment = "Today: [No Pricing]"
		}
	} else {
		var suffix string
		stale := false
		parsedTime, parseErr := time.Parse(time.RFC3339, apiUsage.LastPollTime)
		if parseErr == nil {
			if now.Sub(parsedTime) >= 5*time.Minute {
				stale = true
			}
		}

		if stale {
			suffix = " [Daemon Dead]"
		} else if apiUsage.Status == "auth_error" {
			suffix = " [Auth Err]"
		} else if apiUsage.Status == "network_error" {
			suffix = " [Offline]"
		} else if apiUsage.Status != "success" && apiUsage.Status != "" {
			suffix = " [Daemon Err]"
		}

		if payload.TerminalWidth < 60 && suffix != "" {
			todaySegment = fmt.Sprintf("~$%.2f%s", apiUsage.TodayCostUSD, suffix)
		} else if payload.TerminalWidth < 60 {
			todaySegment = fmt.Sprintf("Today: ~$%.2f", apiUsage.TodayCostUSD)
		} else {
			todaySegment = fmt.Sprintf("Today: ~$%.2f%s", apiUsage.TodayCostUSD, suffix)
		}
	}

	modelShort := ShortenModelName(payload.Model.DisplayName, payload.Model.ID)

	if payload.TerminalWidth >= 120 {
		var turnPart, sessPart string
		if pricingAvailable {
			turnPart = fmt.Sprintf("Turn: +%s/%s (%s)", FormatTokens(payload.ContextWindow.CurrentUsage.InputTokens), FormatTokens(payload.ContextWindow.CurrentUsage.OutputTokens), turnCostStr)
			sessPart = fmt.Sprintf("Sess: %s/%s (%s)", FormatTokens(payload.ContextWindow.TotalInputTokens), FormatTokens(payload.ContextWindow.TotalOutputTokens), sessCostStr)
		} else {
			turnPart = fmt.Sprintf("Turn: +%s/%s [No Pricing]", FormatTokens(payload.ContextWindow.CurrentUsage.InputTokens), FormatTokens(payload.ContextWindow.CurrentUsage.OutputTokens))
			sessPart = fmt.Sprintf("Sess: %s/%s [No Pricing]", FormatTokens(payload.ContextWindow.TotalInputTokens), FormatTokens(payload.ContextWindow.TotalOutputTokens))
		}
		ctxPart := fmt.Sprintf("Ctx: %.1f%%", 100.0-payload.ContextWindow.RemainingPercentage)
		return fmt.Sprintf("%s │ %s │ %s │ %s │ %s │ %s", stateSegment, modelShort, turnPart, sessPart, todaySegment, ctxPart)
	}

	if payload.TerminalWidth >= 110 {
		var turnPart, sessPart string
		if pricingAvailable {
			turnPart = fmt.Sprintf("Turn: +%s/%s (%s)", FormatTokens(payload.ContextWindow.CurrentUsage.InputTokens), FormatTokens(payload.ContextWindow.CurrentUsage.OutputTokens), turnCostStr)
			sessPart = fmt.Sprintf("Sess: %s/%s (%s)", FormatTokens(payload.ContextWindow.TotalInputTokens), FormatTokens(payload.ContextWindow.TotalOutputTokens), sessCostStr)
		} else {
			turnPart = fmt.Sprintf("Turn: +%s/%s [No Pricing]", FormatTokens(payload.ContextWindow.CurrentUsage.InputTokens), FormatTokens(payload.ContextWindow.CurrentUsage.OutputTokens))
			sessPart = fmt.Sprintf("Sess: %s/%s [No Pricing]", FormatTokens(payload.ContextWindow.TotalInputTokens), FormatTokens(payload.ContextWindow.TotalOutputTokens))
		}
		return fmt.Sprintf("%s │ %s │ %s │ %s │ %s", stateSegment, modelShort, turnPart, sessPart, todaySegment)
	}

	if payload.TerminalWidth >= 85 {
		var sessPart string
		if pricingAvailable {
			sessCostVal := pricing.CalculateCost(payload.ContextWindow.TotalInputTokens, payload.ContextWindow.TotalCachedTokens, payload.ContextWindow.TotalOutputTokens, rates)
			sessPart = fmt.Sprintf("Sess: %s/%s (~$%.3f)", FormatTokens(payload.ContextWindow.TotalInputTokens), FormatTokens(payload.ContextWindow.TotalOutputTokens), sessCostVal)
		} else {
			sessPart = fmt.Sprintf("Sess: %s/%s [No Pricing]", FormatTokens(payload.ContextWindow.TotalInputTokens), FormatTokens(payload.ContextWindow.TotalOutputTokens))
		}
		return fmt.Sprintf("%s │ %s │ %s │ %s", stateSegment, modelShort, sessPart, todaySegment)
	}

	if payload.TerminalWidth >= 60 {
		sessPart := fmt.Sprintf("Sess: %s", FormatTokens(payload.ContextWindow.TotalInputTokens))
		return fmt.Sprintf("%s │ %s │ %s", stateSegment, sessPart, todaySegment)
	}

	return fmt.Sprintf("%s │ %s", stateSegment, todaySegment)
}
