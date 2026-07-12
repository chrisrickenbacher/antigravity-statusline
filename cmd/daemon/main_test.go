package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chrisrickenbacher/antigravity-statusline/pkg/cache"
	"github.com/chrisrickenbacher/antigravity-statusline/pkg/pricing"
)

func TestPruneOldLogs(t *testing.T) {
	tempCacheDir, err := os.MkdirTemp("", "cache-mock-prune-*")
	if err != nil {
		t.Fatalf("Failed to create temp cache: %v", err)
	}
	defer os.RemoveAll(tempCacheDir)

	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)

	// Create today's log, 1-days-old log, and 5-days-old log
	todayPath := filepath.Join(tempCacheDir, "usage_session1_2026-07-09.jsonl")
	recentPath := filepath.Join(tempCacheDir, "usage_session2_2026-07-08.jsonl")
	oldPath := filepath.Join(tempCacheDir, "usage_session3_2026-07-04.jsonl")
	otherFilePath := filepath.Join(tempCacheDir, "other_file.txt")

	_ = os.WriteFile(todayPath, []byte("{}"), 0644)
	_ = os.WriteFile(recentPath, []byte("{}"), 0644)
	_ = os.WriteFile(oldPath, []byte("{}"), 0644)
	_ = os.WriteFile(otherFilePath, []byte("{}"), 0644)

	pruneOldLogs(tempCacheDir, now)

	// verify today's log exists
	if _, err := os.Stat(todayPath); os.IsNotExist(err) {
		t.Error("Expected today's log to be preserved, but it was deleted")
	}

	// verify 1-days-old log exists
	if _, err := os.Stat(recentPath); os.IsNotExist(err) {
		t.Error("Expected recent log (1 day old) to be preserved, but it was deleted")
	}

	// verify other file is preserved
	if _, err := os.Stat(otherFilePath); os.IsNotExist(err) {
		t.Error("Expected other non-log file to be preserved, but it was deleted")
	}

	// verify 5-days-old log is deleted
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Expected 5-days-old log to be deleted, but it still exists")
	}
}

func TestAggregateSessionLogs(t *testing.T) {
	tempCacheDir, err := os.MkdirTemp("", "cache-mock-aggregate-*")
	if err != nil {
		t.Fatalf("Failed to create temp cache: %v", err)
	}
	defer os.RemoveAll(tempCacheDir)

	localDate := "2026-07-09"

	// Create a couple of mocked session files for today
	sess1Path := filepath.Join(tempCacheDir, "usage_session1_"+localDate+".jsonl")
	sess2Path := filepath.Join(tempCacheDir, "usage_session2_"+localDate+".jsonl")

	// Write entries to sess1
	entry1 := cache.LocalUsageEntry{
		Timestamp:         "2026-07-09T10:00:00Z",
		ModelID:           "gemini-3.5-flash",
		InputTokens:       100,
		CachedInputTokens: 50,
		OutputTokens:      20,
	}
	entry2 := cache.LocalUsageEntry{
		Timestamp:         "2026-07-09T10:01:00Z",
		ModelID:           "gemini-1.5-pro",
		InputTokens:       200,
		CachedInputTokens: 0,
		OutputTokens:      50,
	}

	f1, err := os.Create(sess1Path)
	if err != nil {
		t.Fatalf("failed to create session1: %v", err)
	}
	bytes1, _ := json.Marshal(entry1)
	_, _ = f1.Write(append(bytes1, '\n'))
	bytes2, _ := json.Marshal(entry2)
	_, _ = f1.Write(append(bytes2, '\n'))
	f1.Close()

	// Write entries to sess2
	entry3 := cache.LocalUsageEntry{
		Timestamp:         "2026-07-09T10:02:00Z",
		ModelID:           "Gemini 3.5 Flash", // Case/space variations to normalize
		InputTokens:       50,
		CachedInputTokens: 25,
		OutputTokens:      10,
	}

	f2, err := os.Create(sess2Path)
	if err != nil {
		t.Fatalf("failed to create session2: %v", err)
	}
	bytes3, _ := json.Marshal(entry3)
	_, _ = f2.Write(append(bytes3, '\n'))
	f2.Close()

	// Run aggregation helper
	modelTotals, totalInput, totalCached, totalOutput, err := aggregateSessionLogs(tempCacheDir, localDate)
	if err != nil {
		t.Fatalf("aggregateSessionLogs failed: %v", err)
	}

	if totalInput != 350 {
		t.Errorf("expected total input 350, got %d", totalInput)
	}
	if totalCached != 75 {
		t.Errorf("expected total cached 75, got %d", totalCached)
	}
	if totalOutput != 80 {
		t.Errorf("expected total output 80, got %d", totalOutput)
	}

	flashKey := "gemini3.5flash"
	proKey := "gemini1.5pro"

	if flashTotals, ok := modelTotals["default"][flashKey]; ok {
		if flashTotals.Input != 150 {
			t.Errorf("expected flash input 150, got %d", flashTotals.Input)
		}
		if flashTotals.Cached != 75 {
			t.Errorf("expected flash cached 75, got %d", flashTotals.Cached)
		}
		if flashTotals.Output != 30 {
			t.Errorf("expected flash output 30, got %d", flashTotals.Output)
		}
	} else {
		t.Errorf("missing aggregated model totals for key %q", flashKey)
	}

	if proTotals, ok := modelTotals["default"][proKey]; ok {
		if proTotals.Input != 200 {
			t.Errorf("expected pro input 200, got %d", proTotals.Input)
		}
		if proTotals.Cached != 0 {
			t.Errorf("expected pro cached 0, got %d", proTotals.Cached)
		}
		if proTotals.Output != 50 {
			t.Errorf("expected pro output 50, got %d", proTotals.Output)
		}
	} else {
		t.Errorf("missing aggregated model totals for key %q", proKey)
	}
}

func TestSyncPricing(t *testing.T) {
	tempCacheDir, err := os.MkdirTemp("", "cache-mock-pricing-*")
	if err != nil {
		t.Fatalf("Failed to create temp cache: %v", err)
	}
	defer os.RemoveAll(tempCacheDir)

	t.Setenv("ANTIGRAVITY_CACHE_DIR", tempCacheDir)

	mockRates := pricing.PricingCache{
		LastFetched: "",
		Models: map[string]pricing.ModelRate{
			"gemini-3.5-flash": {
				InputPricePer1M:  0.075,
				OutputPricePer1M: 0.300,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockRates)
	}))
	defer server.Close()

	res, err := syncPricing(server.URL)
	if err != nil {
		t.Fatalf("syncPricing failed on first fetch: %v", err)
	}

	if res.Models["gemini-3.5-flash"].InputPricePer1M != 0.075 {
		t.Errorf("Expected fetched input rate 0.075, got %v", res.Models["gemini-3.5-flash"].InputPricePer1M)
	}

	var cached pricing.PricingCache
	if err := cache.ReadJSON("pricing_cache.json", &cached); err != nil {
		t.Fatalf("failed to read cached pricing: %v", err)
	}

	if cached.LastFetched == "" {
		t.Error("Expected pricing_cache.json to have LastFetched timestamp set")
	}

	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	res2, err := syncPricing(server.URL)
	if err != nil {
		t.Fatalf("syncPricing failed on cached fetch: %v", err)
	}

	if res2.Models["gemini-3.5-flash"].InputPricePer1M != 0.075 {
		t.Errorf("Expected cached input rate 0.075, got %v", res2.Models["gemini-3.5-flash"].InputPricePer1M)
	}
}
