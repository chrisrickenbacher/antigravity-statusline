package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/chrisrickenbacher/antigravity-statusline/pkg/cache"
	"github.com/chrisrickenbacher/antigravity-statusline/pkg/layout"
	"github.com/chrisrickenbacher/antigravity-statusline/pkg/pricing"
	"github.com/chrisrickenbacher/antigravity-statusline/pkg/state"
)

func main() {
	stdinBytes, err := io.ReadAll(os.Stdin)
	if err != nil || len(stdinBytes) == 0 {
		return
	}

	var payload state.StdinPayload
	if err := json.Unmarshal(stdinBytes, &payload); err != nil {
		return
	}

	// Record this turn's usage in its own isolated session log
	_ = cache.AppendLocalUsage(
		payload.ConversationID,
		payload.Model.ID,
		payload.ContextWindow.CurrentUsage.InputTokens,
		payload.ContextWindow.CurrentUsage.CachedInputTokens,
		payload.ContextWindow.CurrentUsage.OutputTokens,
		payload.ContextWindow.TotalInputTokens,
		payload.ContextWindow.TotalOutputTokens,
	)

	// Since the CLI doesn't send total_cached_tokens, calculate it from session logs
	if totalCached, err := cache.GetSessionCachedTokens(payload.ConversationID); err == nil {
		payload.ContextWindow.TotalCachedTokens = totalCached
	}

	var priceCache pricing.PricingCache
	var priceCachePtr *pricing.PricingCache
	if err := cache.ReadJSON("pricing_cache.json", &priceCache); err == nil {
		priceCachePtr = &priceCache
	} else {
		if embedded, dErr := pricing.GetDefaultPricing(); dErr == nil {
			priceCache = *embedded
			priceCachePtr = &priceCache
		}
	}

	var apiUsage state.ApiUsage
	var apiUsagePtr *state.ApiUsage
	apiUsageErr := cache.ReadJSON("api_usage.json", &apiUsage)
	if apiUsageErr == nil {
		apiUsagePtr = &apiUsage
	}

	output := layout.RenderStatusLine(&payload, priceCachePtr, apiUsagePtr, apiUsageErr, time.Now())
	fmt.Print(output)
}
