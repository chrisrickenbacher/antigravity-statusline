package main

import (
	"bufio"
	"context"
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

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/chrisrickenbacher/antigravity-statusline/pkg/cache"
	"github.com/chrisrickenbacher/antigravity-statusline/pkg/pricing"
	"github.com/chrisrickenbacher/antigravity-statusline/pkg/state"
)

var resolvedProjectID string

func resolveProjectID(flagProj string) string {
	if flagProj != "" {
		return flagProj
	}
	// Try reading settings.json from ANTIGRAVITY_CACHE_DIR/../settings.json or ~/.gemini/antigravity-cli/settings.json
	var settingsPath string
	if cacheDir := os.Getenv("ANTIGRAVITY_CACHE_DIR"); cacheDir != "" {
		settingsPath = filepath.Join(filepath.Dir(cacheDir), "settings.json")
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			settingsPath = filepath.Join(home, ".gemini", "antigravity-cli", "settings.json")
		}
	}

	if settingsPath != "" {
		if settingsBytes, err := os.ReadFile(settingsPath); err == nil {
			var settings struct {
				GCP struct {
					Project string `json:"project"`
				} `json:"gcp"`
			}
			if err := json.Unmarshal(settingsBytes, &settings); err == nil && settings.GCP.Project != "" {
				return settings.GCP.Project
			}
		}
	}

	if envProj := os.Getenv("GCP_PROJECT_ID"); envProj != "" {
		return envProj
	}
	if envProj := os.Getenv("GOOGLE_CLOUD_PROJECT"); envProj != "" {
		return envProj
	}

	home, err := os.UserHomeDir()
	if err == nil {
		activeConfigPath := filepath.Join(home, ".config", "gcloud", "active_config")
		if activeConfigBytes, err := os.ReadFile(activeConfigPath); err == nil {
			configName := strings.TrimSpace(string(activeConfigBytes))
			if configName == "" {
				configName = "default"
			}
			configPath := filepath.Join(home, ".config", "gcloud", "configurations", "config_"+configName)
			if configBytes, err := os.ReadFile(configPath); err == nil {
				lines := strings.Split(string(configBytes), "\n")
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if strings.HasPrefix(trimmed, "project") {
						parts := strings.Split(trimmed, "=")
						if len(parts) == 2 {
							return strings.TrimSpace(parts[1])
						}
					}
				}
			}
		}
	}
	return ""
}

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fallback := func(originalErr error) (*pricing.PricingCache, error) {
		if readErr == nil {
			return &current, nil
		}
		defaultCache, err := pricing.GetDefaultPricing()
		if err != nil {
			return nil, fmt.Errorf("failed to load embedded default pricing: %w", err)
		}
		defaultCache.LastFetched = time.Now().Format(time.RFC3339)
		_ = cache.WriteJSON("pricing_cache.json", defaultCache)
		return defaultCache, fmt.Errorf("%w (created local fallback)", originalErr)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", pricingURL, nil)
	if err != nil {
		return fallback(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fallback(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallback(fmt.Errorf("unexpected status code: %d", resp.StatusCode))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fallback(err)
	}

	var fetched pricing.PricingCache
	if err := json.Unmarshal(bodyBytes, &fetched); err != nil {
		return fallback(err)
	}

	fetched.LastFetched = time.Now().Format(time.RFC3339)
	if err := cache.WriteJSON("pricing_cache.json", &fetched); err != nil {
		return nil, fmt.Errorf("failed to write pricing cache: %w", err)
	}

	return &fetched, nil
}

type TokenStats struct {
	TotalTokens int64
	TotalCost   float64
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
		if strings.HasPrefix(name, "local_usage_") && strings.HasSuffix(name, ".jsonl") {
			dateStr := strings.TrimSuffix(strings.TrimPrefix(name, "local_usage_"), ".jsonl")
			parsedDate, err := time.Parse("2006-01-02", dateStr)
			if err == nil && parsedDate.Before(cutoff) {
				_ = os.Remove(filepath.Join(cacheDir, name))
			}
		}
	}
}

func aggregateTodayLocalUsage(cacheDir string, localDate string) (map[string]int64, error) {
	filename := fmt.Sprintf("local_usage_%s.jsonl", localDate)
	filePath := filepath.Join(cacheDir, filename)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]int64), nil
		}
		return nil, err
	}
	defer file.Close()

	usageByModel := make(map[string]int64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry struct {
			ModelID           string `json:"model_id"`
			CachedInputTokens int64  `json:"cached_input_tokens"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		normalizedModel := pricing.NormalizeModelID(entry.ModelID)
		usageByModel[normalizedModel] += entry.CachedInputTokens
	}

	return usageByModel, scanner.Err()
}

func queryMetric(ctx context.Context, client *monitoring.MetricClient, projectID string, metricType string, startTime, endTime time.Time, priceCache *pricing.PricingCache, isOutput bool, localCached map[string]int64) (TokenStats, error) {
	typeValue := "input"
	if isOutput {
		typeValue = "output"
	}
	filter := fmt.Sprintf(`metric.type = "%s" AND metric.labels.type = "%s"`, metricType, typeValue)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + projectID,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(startTime),
			EndTime:   timestamppb.New(endTime),
		},
		Aggregation: &monitoringpb.Aggregation{
			AlignmentPeriod:  durationpb.New(endTime.Sub(startTime)),
			PerSeriesAligner: monitoringpb.Aggregation_ALIGN_SUM,
		},
	}

	var stats TokenStats
	it := client.ListTimeSeries(ctx, req)

	gcpTokensByModel := make(map[string]int64)

	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return TokenStats{}, nil
			}
			return TokenStats{}, err
		}

		var modelID string
		if resp.Resource != nil && resp.Resource.Labels != nil {
			modelID = resp.Resource.Labels["model_user_id"]
		}
		if modelID == "" && resp.Metric != nil && resp.Metric.Labels != nil {
			modelID = resp.Metric.Labels["model_id"]
		}

		normModel := pricing.NormalizeModelID(modelID)

		var seriesTokens int64
		for _, point := range resp.Points {
			if point.Value == nil {
				continue
			}
			seriesTokens += point.Value.GetInt64Value()
		}

		stats.TotalTokens += seriesTokens

		if isOutput {
			rates, err := pricing.ResolveRates(priceCache, modelID)
			if err == nil {
				stats.TotalCost += float64(seriesTokens) * rates.OutputRate
			}
		} else {
			gcpTokensByModel[normModel] += seriesTokens
		}
	}

	if !isOutput {
		for normModel, gcpInput := range gcpTokensByModel {
			rates, err := pricing.ResolveRates(priceCache, normModel)
			if err != nil {
				continue
			}

			loggedCached := localCached[normModel]
			cached := loggedCached
			if cached > gcpInput {
				cached = gcpInput
			}
			standard := gcpInput - cached

			stats.TotalCost += (float64(standard) * rates.InputRate) + (float64(cached) * rates.CachedInputRate)
		}
	}

	return stats, nil
}

func main() {
	pricingURL := flag.String("pricing-url", "https://raw.githubusercontent.com/chrisrickenbacher/antigravity-statusline/main/pkg/pricing/pricing.json", "GCP pricing endpoint URL")
	gcpProjectID := flag.String("project", "", "GCP Project ID (overrides env)")
	cacheDirOverride := flag.String("cache-dir", "", "Custom cache directory path")
	flag.Parse()

	if *cacheDirOverride != "" {
		os.Setenv("ANTIGRAVITY_CACHE_DIR", *cacheDirOverride)
	}

	cacheDir, cacheDirErr := cache.GetCacheDir()
	if cacheDirErr != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve cache directory: %v\n", cacheDirErr)
		os.Exit(1)
	}

	pruneOldLogs(cacheDir, time.Now())

	localDate := time.Now().Format("2006-01-02")
	localCached, err := aggregateTodayLocalUsage(cacheDir, localDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to aggregate today's local caching logs: %v\n", err)
		localCached = make(map[string]int64)
	}

	pricingCache, priceErr := syncPricing(*pricingURL)
	if priceErr != nil {
		fmt.Fprintf(os.Stderr, "warning: pricing synchronization failed: %v\n", priceErr)
	}

	projectID := resolveProjectID(*gcpProjectID)
	resolvedProjectID = projectID
	if projectID == "" {
		writeErrorStatus("config_error", "unresolved GCP Project ID")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	client, err := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/monitoring.read"))
	if err != nil {
		if strings.Contains(err.Error(), "credentials") {
			writeErrorStatus("auth_error", err.Error())
		} else {
			writeErrorStatus("network_error", err.Error())
		}
		os.Exit(1)
	}
	defer client.Close()

	now := time.Now()
	localMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startTime := localMidnight.UTC()
	endTime := now.UTC()

	const metricType = "aiplatform.googleapis.com/publisher/online_serving/token_count"

	inputStats, err := queryMetric(ctx, client, projectID, metricType, startTime, endTime, pricingCache, false, localCached)
	if err != nil {
		handleError(err)
		os.Exit(1)
	}

	outputStats, err := queryMetric(ctx, client, projectID, metricType, startTime, endTime, pricingCache, true, localCached)
	if err != nil {
		handleError(err)
		os.Exit(1)
	}

	var todayTotalCached int64
	for _, cachedVal := range localCached {
		todayTotalCached += cachedVal
	}

	var finalRatio float64
	if inputStats.TotalTokens > 0 {
		finalRatio = float64(todayTotalCached) / float64(inputStats.TotalTokens)
		if finalRatio > 1.0 {
			finalRatio = 1.0
		}
	}

	apiUsage := state.ApiUsage{
		GCPProjectID:      projectID,
		LastPollTime:      now.Format(time.RFC3339),
		Status:            "success",
		ErrorMessage:      "",
		TodayCostUSD:      math.Round((inputStats.TotalCost+outputStats.TotalCost)*1e6) / 1e6, // round to 6 decimal places
		TodayInputTokens:  inputStats.TotalTokens,
		TodayOutputTokens: outputStats.TotalTokens,
		CachingRatio:      finalRatio,
	}

	if err := cache.WriteJSON("api_usage.json", &apiUsage); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write api_usage cache: %v\n", err)
		os.Exit(1)
	}
}

func writeErrorStatus(status, errMsg string) {
	fmt.Fprintf(os.Stderr, "Error [%s]: %s\n", status, errMsg)
	var current state.ApiUsage
	_ = cache.ReadJSON("api_usage.json", &current)

	current.Status = status
	current.ErrorMessage = errMsg
	current.LastPollTime = time.Now().Format(time.RFC3339)
	if resolvedProjectID != "" {
		current.GCPProjectID = resolvedProjectID
	}

	_ = cache.WriteJSON("api_usage.json", &current)
}

func handleError(err error) {
	errStr := err.Error()
	if strings.Contains(errStr, "credentials") || strings.Contains(errStr, "unauthenticated") {
		writeErrorStatus("auth_error", errStr)
	} else {
		writeErrorStatus("network_error", errStr)
	}
}
