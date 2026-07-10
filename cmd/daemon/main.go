package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chrisrickenbacher/antigravity-statusline/pkg/cache"
	"github.com/chrisrickenbacher/antigravity-statusline/pkg/pricing"
	"github.com/chrisrickenbacher/antigravity-statusline/pkg/state"
)

func syncPricing(pricingURL string) (*pricing.PricingCache, error) {
	var current pricing.PricingCache
	readErr := cache.ReadJSON("pricing_cache.json", &current)

	needsFetch := true
	if readErr == nil && current.LastFetched != "" {
		parsed, err := time.Parse(time.RFC3339, current.LastFetched)
		if err == nil && time.Since(parsed) < 24*time.Hour {
			needsFetch = false
		}
	}

	if !needsFetch {
		return &current, nil
	}

	resp, err := http.Get(pricingURL)
	if err != nil {
		if readErr == nil {
			return &current, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if readErr == nil {
			return &current, nil
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var fetched pricing.PricingCache
	if err := json.Unmarshal(bodyBytes, &fetched); err != nil {
		return nil, err
	}

	// Merge fetched rates with local embedded defaults for missing fields (like CachedInputPricePer1M)
	if embedded, err := pricing.GetDefaultPricing(); err == nil {
		for name, fetchedModel := range fetched.Models {
			if fetchedModel.CachedInputPricePer1M == 0 {
				if embeddedModel, ok := embedded.Models[name]; ok && embeddedModel.CachedInputPricePer1M > 0 {
					fetchedModel.CachedInputPricePer1M = embeddedModel.CachedInputPricePer1M
					fetched.Models[name] = fetchedModel
				}
			}
		}
	}

	fetched.LastFetched = time.Now().Format(time.RFC3339)
	_ = cache.WriteJSON("pricing_cache.json", &fetched)
	return &fetched, nil
}

func pruneOldLogs(cacheDir string, now time.Time) {
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		return
	}

	cutoff := now.AddDate(0, 0, -7)

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if strings.HasPrefix(name, "usage_") && strings.HasSuffix(name, ".jsonl") {
			base := strings.TrimSuffix(name, ".jsonl")
			if len(base) >= 10 {
				dateStr := base[len(base)-10:]
				parsedDate, err := time.Parse("2006-01-02", dateStr)
				if err == nil && parsedDate.Before(cutoff) {
					_ = os.Remove(filepath.Join(cacheDir, name))
				}
			}
		}
	}
}

type ModelTotals struct {
	Input  int64
	Cached int64
	Output int64
}

func aggregateSessionLogs(cacheDir, localDate string) (map[string]*ModelTotals, int64, int64, int64, error) {
	modelTotals := make(map[string]*ModelTotals)
	var totalInput, totalCached, totalOutput int64

	pattern := filepath.Join(cacheDir, fmt.Sprintf("usage_*_%s.jsonl", localDate))
	files, globErr := filepath.Glob(pattern)
	if globErr != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to glob session logs: %w", globErr)
	}

	for _, logPath := range files {
		file, err := os.Open(logPath)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var entry cache.LocalUsageEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}

			normModel := pricing.NormalizeModelID(entry.ModelID)
			totals, exists := modelTotals[normModel]
			if !exists {
				totals = &ModelTotals{}
				modelTotals[normModel] = totals
			}

			totals.Input += entry.InputTokens
			totals.Cached += entry.CachedInputTokens
			totals.Output += entry.OutputTokens

			totalInput += entry.InputTokens
			totalCached += entry.CachedInputTokens
			totalOutput += entry.OutputTokens
		}
		file.Close()
	}

	return modelTotals, totalInput, totalCached, totalOutput, nil
}

func main() {
	pricingURL := flag.String("pricing-url", "https://raw.githubusercontent.com/chrisrickenbacher/antigravity-statusline/main/pkg/pricing/pricing.json", "GCP pricing endpoint URL")
	cacheDirOverride := flag.String("cache-dir", "", "Custom cache directory path")
	flag.Parse()

	if *cacheDirOverride != "" {
		os.Setenv("ANTIGRAVITY_CACHE_DIR", *cacheDirOverride)
	}

	cacheDir, err := cache.GetCacheDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve cache directory: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()
	localDate := now.Format("2006-01-02")

	// 1. Clean up logs older than 7 days
	pruneOldLogs(cacheDir, now)

	// 2. Sync pricing (checks if local cached is older than 24h)
	pricingCache, _ := syncPricing(*pricingURL)
	if pricingCache == nil {
		// Fallback to embedded default pricing if sync failed and no cache exists
		pricingCache, _ = pricing.GetDefaultPricing()
	}

	status := "success"
	var errMsg string

	// 3. Scan all session files for today using helper
	modelTotals, _, _, _, err := aggregateSessionLogs(cacheDir, localDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to aggregate session logs: %v\n", err)
		status = "error"
		errMsg = err.Error()
	}

	var totalCost float64
	modelsBreakdown := make(map[string]state.ModelUsage)

	// 4. Calculate cost per model using pricing cache
	for modelID, totals := range modelTotals {
		var modelCost float64
		rates, err := pricing.ResolveRates(pricingCache, modelID)
		if err == nil {
			modelCost = pricing.CalculateCost(totals.Input, totals.Cached, totals.Output, rates)
			totalCost += modelCost
		}

		modelsBreakdown[modelID] = state.ModelUsage{
			InputTokens:  totals.Input,
			OutputTokens: totals.Output,
			CachedTokens: totals.Cached,
			CostUSD:      math.Round(modelCost*1e6) / 1e6,
		}
	}

	apiUsage := state.ApiUsage{
		LastPollTime: now.Format(time.RFC3339),
		Status:       status,
		ErrorMessage: errMsg,
		TodayCostUSD: math.Round(totalCost*1e6) / 1e6,
		Models:       modelsBreakdown,
	}

	_ = cache.WriteJSON("api_usage.json", &apiUsage)
}
