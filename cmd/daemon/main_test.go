package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveProjectID(t *testing.T) {
	// Isolate HOME directory so tests don't read the real settings.json of the user
	tempHome, err := os.MkdirTemp("", "home-mock-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)
	t.Setenv("HOME", tempHome)

	t.Run("uses flag override", func(t *testing.T) {
		res := resolveProjectID("flag-project")
		if res != "flag-project" {
			t.Errorf("Expected flag-project, got %q", res)
		}
	})

	t.Run("uses GCP_PROJECT_ID env override", func(t *testing.T) {
		t.Setenv("GCP_PROJECT_ID", "env-gcp-project")
		t.Setenv("GOOGLE_CLOUD_PROJECT", "env-google-project")

		res := resolveProjectID("")
		if res != "env-gcp-project" {
			t.Errorf("Expected env-gcp-project, got %q", res)
		}
	})

	t.Run("uses GOOGLE_CLOUD_PROJECT env override when GCP_PROJECT_ID is empty", func(t *testing.T) {
		os.Unsetenv("GCP_PROJECT_ID")
		t.Setenv("GOOGLE_CLOUD_PROJECT", "env-google-project")

		res := resolveProjectID("")
		if res != "env-google-project" {
			t.Errorf("Expected env-google-project, got %q", res)
		}
	})

	t.Run("uses mock gcloud configuration", func(t *testing.T) {
		os.Unsetenv("GCP_PROJECT_ID")
		os.Unsetenv("GOOGLE_CLOUD_PROJECT")

		tempHome, err := os.MkdirTemp("", "gcloud-mock-*")
		if err != nil {
			t.Fatalf("Failed to create temp home: %v", err)
		}
		defer os.RemoveAll(tempHome)

		t.Setenv("HOME", tempHome)

		gcloudConfigDir := filepath.Join(tempHome, ".config", "gcloud")
		err = os.MkdirAll(filepath.Join(gcloudConfigDir, "configurations"), 0755)
		if err != nil {
			t.Fatalf("Failed to create mock config structure: %v", err)
		}

		err = os.WriteFile(filepath.Join(gcloudConfigDir, "active_config"), []byte("prod-env"), 0644)
		if err != nil {
			t.Fatalf("Failed to write mock active config: %v", err)
		}

		mockIni := `[core]
account = cric@yourstore.dev
project = mock-gcloud-project-id
`
		err = os.WriteFile(filepath.Join(gcloudConfigDir, "configurations", "config_prod-env"), []byte(mockIni), 0644)
		if err != nil {
			t.Fatalf("Failed to write mock config file: %v", err)
		}

		res := resolveProjectID("")
		if res != "mock-gcloud-project-id" {
			t.Errorf("Expected mock-gcloud-project-id, got %q", res)
		}
	})
}

func TestAggregateTodayLocalUsage(t *testing.T) {
	tempCacheDir, err := os.MkdirTemp("", "cache-mock-*")
	if err != nil {
		t.Fatalf("Failed to create temp cache: %v", err)
	}
	defer os.RemoveAll(tempCacheDir)

	localDate := "2026-07-09"
	logFilename := "local_usage_" + localDate + ".jsonl"
	logPath := filepath.Join(tempCacheDir, logFilename)

	mockLines := []string{
		`{"timestamp":"2026-07-09T08:00:00Z","model_id":"gemini-3.5-flash","input_tokens":1000,"output_tokens":300,"cached_input_tokens":800}`,
		`{"timestamp":"2026-07-09T08:05:00Z","model_id":"Gemini 3.5 Flash","input_tokens":2000,"output_tokens":500,"cached_input_tokens":1500}`,
		`{"timestamp":"2026-07-09T08:10:00Z","model_id":"gemini-1.5-pro","input_tokens":5000,"output_tokens":1000,"cached_input_tokens":4000}`,
		`invalid_json_line_that_should_be_skipped`,
		`{"timestamp":"2026-07-09T08:15:00Z","model_id":"gemini-1.5-pro","input_tokens":1000,"output_tokens":200,"cached_input_tokens":500}`,
	}

	err = os.WriteFile(logPath, []byte(strings.Join(mockLines, "\n")), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock log: %v", err)
	}

	res, err := aggregateTodayLocalUsage(tempCacheDir, localDate)
	if err != nil {
		t.Fatalf("aggregateTodayLocalUsage failed: %v", err)
	}

	// Normalized model IDs are used as keys
	flashKey := "gemini3.5flash"
	proKey := "gemini1.5pro"

	if val := res[flashKey]; val != 2300 {
		t.Errorf("Expected 2300 cached tokens for flash, got %d", val)
	}

	if val := res[proKey]; val != 4500 {
		t.Errorf("Expected 4500 cached tokens for pro, got %d", val)
	}
}

func TestPruneOldLogs(t *testing.T) {
	tempCacheDir, err := os.MkdirTemp("", "cache-mock-prune-*")
	if err != nil {
		t.Fatalf("Failed to create temp cache: %v", err)
	}
	defer os.RemoveAll(tempCacheDir)

	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)

	// Create today's log, 5-days-old log, and 10-days-old log
	todayPath := filepath.Join(tempCacheDir, "local_usage_2026-07-09.jsonl")
	recentPath := filepath.Join(tempCacheDir, "local_usage_2026-07-04.jsonl")
	oldPath := filepath.Join(tempCacheDir, "local_usage_2026-06-25.jsonl")
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

	// verify 5-days-old log exists
	if _, err := os.Stat(recentPath); os.IsNotExist(err) {
		t.Error("Expected recent log (5 days old) to be preserved, but it was deleted")
	}

	// verify other file is preserved
	if _, err := os.Stat(otherFilePath); os.IsNotExist(err) {
		t.Error("Expected other non-log file to be preserved, but it was deleted")
	}

	// verify 14-days-old log is deleted
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Expected 14-days-old log to be deleted, but it still exists")
	}
}
