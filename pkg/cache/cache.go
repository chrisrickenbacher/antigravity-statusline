package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

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
