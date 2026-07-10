package cache

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type LocalUsageEntry struct {
	Timestamp         string `json:"timestamp"`
	ModelID           string `json:"model_id"`
	InputTokens       int64  `json:"input_tokens"`
	CachedInputTokens int64  `json:"cached_input_tokens"`
	OutputTokens      int64  `json:"output_tokens"`
	TotalInputTokens  int64  `json:"total_input_tokens"`
	TotalOutputTokens int64  `json:"total_output_tokens"`
}

func GetCacheDir() (string, error) {
	if cacheDir := os.Getenv("ANTIGRAVITY_CACHE_DIR"); cacheDir != "" {
		return cacheDir, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".gemini", "antigravity-cli", "cache"), nil
}

func WriteJSON(filename string, data interface{}) error {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	marshalled, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	targetPath := filepath.Join(cacheDir, filename)
	tempPath := targetPath + ".tmp"

	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temporary cache file: %w", err)
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(marshalled); err != nil {
		return fmt.Errorf("failed to write data to temporary cache file: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary cache file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary cache file: %w", err)
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		return fmt.Errorf("failed to atomically rename temporary cache file to target: %w", err)
	}

	return nil
}

func ReadJSON(filename string, target interface{}) error {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}

	targetPath := filepath.Join(cacheDir, filename)

	fileBytes, err := os.ReadFile(targetPath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(fileBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON from file %s: %w", targetPath, err)
	}

	return nil
}

// ReadLastLine reads the last non-empty line of a file.
func ReadLastLine(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lastLine string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lastLine = line
		}
	}
	return lastLine, scanner.Err()
}

// AppendLocalUsage logs the turn usage to usage_<conversation_id>_<YYYY-MM-DD>.jsonl.
func AppendLocalUsage(convID, modelID string, input, cached, output, totalInput, totalOutput int64) error {
	if convID == "" {
		return nil
	}

	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	dateStr := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("usage_%s_%s.jsonl", convID, dateStr)
	filePath := filepath.Join(cacheDir, filename)

	// 1. Deduplicate by comparing against the last line of this session's file
	if lastLine, err := ReadLastLine(filePath); err == nil && lastLine != "" {
		var lastEntry LocalUsageEntry
		if err := json.Unmarshal([]byte(lastLine), &lastEntry); err == nil {
			if lastEntry.TotalInputTokens == totalInput && lastEntry.TotalOutputTokens == totalOutput {
				return nil // Turn already logged in this session, skip
			}
		}
	}

	// 2. Append new turn usage atomically using standard file appends
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open session log: %w", err)
	}
	defer file.Close()

	entry := LocalUsageEntry{
		Timestamp:         time.Now().Format(time.RFC3339),
		ModelID:           modelID,
		InputTokens:       input,
		CachedInputTokens: cached,
		OutputTokens:      output,
		TotalInputTokens:  totalInput,
		TotalOutputTokens: totalOutput,
	}

	bytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal usage entry: %w", err)
	}

	if _, err := file.Write(append(bytes, '\n')); err != nil {
		return fmt.Errorf("failed to append usage entry: %w", err)
	}
	_ = file.Sync()

	return nil
}

// GetSessionCachedTokens reads today's session log file and sums all cached input tokens.
func GetSessionCachedTokens(convID string) (int64, error) {
	_, cached, _, err := GetSessionTotals(convID)
	return cached, err
}

// GetSessionTotals reads today's session log file and sums all input, cached, and output tokens.
func GetSessionTotals(convID string) (int64, int64, int64, error) {
	if convID == "" {
		return 0, 0, 0, nil
	}

	cacheDir, err := GetCacheDir()
	if err != nil {
		return 0, 0, 0, err
	}

	dateStr := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("usage_%s_%s.jsonl", convID, dateStr)
	filePath := filepath.Join(cacheDir, filename)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, 0, nil
		}
		return 0, 0, 0, err
	}
	defer file.Close()

	var totalInput, totalCached, totalOutput int64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry LocalUsageEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			totalInput += entry.InputTokens
			totalCached += entry.CachedInputTokens
			totalOutput += entry.OutputTokens
		}
	}
	return totalInput, totalCached, totalOutput, scanner.Err()
}
