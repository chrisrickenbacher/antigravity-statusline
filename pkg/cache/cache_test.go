package cache

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type TestData struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestGetCacheDir(t *testing.T) {
	t.Run("uses environment override", func(t *testing.T) {
		customPath := "/tmp/custom-cache-dir"
		t.Setenv("ANTIGRAVITY_CACHE_DIR", customPath)

		dir, err := GetCacheDir()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if dir != customPath {
			t.Errorf("Expected cache dir '%s', got '%s'", customPath, dir)
		}
	})

	t.Run("defaults to home subfolder when env is empty", func(t *testing.T) {
		os.Unsetenv("ANTIGRAVITY_CACHE_DIR")
		dir, err := GetCacheDir()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if dir == "" {
			t.Errorf("Expected non-empty default cache directory path")
		}
	})
}

func TestWriteAndReadJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "statusline-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("ANTIGRAVITY_CACHE_DIR", tempDir)

	testFile := "test_api_usage.json"
	payload := TestData{Name: "Gemini", Value: 42}

	err = WriteJSON(testFile, &payload)
	if err != nil {
		t.Fatalf("Failed to write JSON: %v", err)
	}

	expectedFilePath := filepath.Join(tempDir, testFile)
	if _, err := os.Stat(expectedFilePath); os.IsNotExist(err) {
		t.Fatalf("Expected file to exist at %s, but it does not", expectedFilePath)
	}

	var parsed TestData
	err = ReadJSON(testFile, &parsed)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	if parsed.Name != payload.Name || parsed.Value != payload.Value {
		t.Errorf("Expected parsed data %+v to match payload %+v", parsed, payload)
	}

	var missing TestData
	err = ReadJSON("does_not_exist.json", &missing)
	if err == nil {
		t.Fatalf("Expected error when reading non-existent file, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected error wrapping os.ErrNotExist, got: %v", err)
	}
}

func TestAppendLocalUsage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "statusline-append-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("ANTIGRAVITY_CACHE_DIR", tempDir)

	convID := "conv-xyz-123"
	modelID := "gemini-3.5-flash"

	// 1. First write
	err = AppendLocalUsage(convID, modelID, 100, 50, 20, 1000, 300)
	if err != nil {
		t.Fatalf("Expected no error on first append, got: %v", err)
	}

	// 2. Duplicate write (same totals)
	err = AppendLocalUsage(convID, modelID, 100, 50, 20, 1000, 300)
	if err != nil {
		t.Fatalf("Expected no error on duplicate append, got: %v", err)
	}

	// 3. New write (different totals)
	err = AppendLocalUsage(convID, modelID, 200, 80, 40, 1200, 340)
	if err != nil {
		t.Fatalf("Expected no error on new append, got: %v", err)
	}

	// Let's read the lines of the session log file
	dateStr := time.Now().Format("2006-01-02")
	expectedPath := filepath.Join(tempDir, "usage_"+convID+"_"+dateStr+".jsonl")

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Expected session log file to exist at %s, but it does not", expectedPath)
	}

	contentBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read session log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(contentBytes)), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected exactly 2 lines in session log (the duplicate should be deduplicated), got %d: %q", len(lines), lines)
	}
}

func TestGetSessionCachedTokens(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "statusline-get-cached-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("ANTIGRAVITY_CACHE_DIR", tempDir)

	convID := "conv-session-cached"
	modelID := "gemini-3.5-flash"

	// Check empty or non-existent file returns 0
	cached, err := GetSessionCachedTokens(convID)
	if err != nil {
		t.Fatalf("Expected no error for empty session, got: %v", err)
	}
	if cached != 0 {
		t.Errorf("Expected 0 cached tokens, got: %d", cached)
	}

	// Append multiple logs
	err = AppendLocalUsage(convID, modelID, 100, 50, 20, 1000, 300)
	if err != nil {
		t.Fatalf("Append failure: %v", err)
	}

	err = AppendLocalUsage(convID, modelID, 200, 80, 40, 1200, 340)
	if err != nil {
		t.Fatalf("Append failure: %v", err)
	}

	// Total cached should be 50 + 80 = 130
	cached, err = GetSessionCachedTokens(convID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cached != 130 {
		t.Errorf("Expected 130 cached tokens, got: %d", cached)
	}

	// Check that empty convID returns 0 without error
	cached, err = GetSessionCachedTokens("")
	if err != nil {
		t.Fatalf("Expected no error for empty convID, got: %v", err)
	}
	if cached != 0 {
		t.Errorf("Expected 0 cached tokens for empty convID, got: %d", cached)
	}
}

func TestGetSessionTotals(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "statusline-get-totals-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("ANTIGRAVITY_CACHE_DIR", tempDir)

	convID := "conv-session-totals"
	modelID := "gemini-3.5-flash"

	// Check empty or non-existent file returns 0
	input, cached, output, err := GetSessionTotals(convID)
	if err != nil {
		t.Fatalf("Expected no error for empty session, got: %v", err)
	}
	if input != 0 || cached != 0 || output != 0 {
		t.Errorf("Expected all 0s, got: input=%d, cached=%d, output=%d", input, cached, output)
	}

	// Append multiple logs
	err = AppendLocalUsage(convID, modelID, 100, 50, 20, 1000, 300)
	if err != nil {
		t.Fatalf("Append failure: %v", err)
	}

	err = AppendLocalUsage(convID, modelID, 200, 80, 40, 1200, 340)
	if err != nil {
		t.Fatalf("Append failure: %v", err)
	}

	// Sums should be: input: 100+200=300, cached: 50+80=130, output: 20+40=60
	input, cached, output, err = GetSessionTotals(convID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if input != 300 || cached != 130 || output != 60 {
		t.Errorf("Expected (300, 130, 60), got: (%d, %d, %d)", input, cached, output)
	}

	// Check that empty convID returns 0 without error
	input, cached, output, err = GetSessionTotals("")
	if err != nil {
		t.Fatalf("Expected no error for empty convID, got: %v", err)
	}
	if input != 0 || cached != 0 || output != 0 {
		t.Errorf("Expected all 0s, got: (%d, %d, %d)", input, cached, output)
	}
}
