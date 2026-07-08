package cache

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
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
