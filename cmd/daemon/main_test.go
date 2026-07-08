package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveProjectID(t *testing.T) {
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
