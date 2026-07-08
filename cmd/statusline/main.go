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

	var priceCache pricing.PricingCache
	var priceCachePtr *pricing.PricingCache
	if err := cache.ReadJSON("pricing_cache.json", &priceCache); err == nil {
		priceCachePtr = &priceCache
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
