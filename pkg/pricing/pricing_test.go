package pricing

import (
	"errors"
	"testing"
)

func TestResolveRates(t *testing.T) {
	mockCache := &PricingCache{
		LastFetched: "2026-07-08T00:00:00Z",
		Models: map[string]ModelRate{
			"gemini-1.5-flash": {
				InputPricePer1M:  0.075,
				OutputPricePer1M: 0.300,
			},
			"gemini-1.5-pro": {
				InputPricePer1M:       1.250,
				CachedInputPricePer1M: 0.125,
				OutputPricePer1M:      5.000,
			},
			"gemini-3.5-flash": {
				InputPricePer1M:  0.075,
				OutputPricePer1M: 0.300,
			},
			"claude-3-5-sonnet": {
				InputPricePer1M:  3.000,
				OutputPricePer1M: 15.000,
			},
		},
	}

	t.Run("exact match case-sensitive and case-insensitive", func(t *testing.T) {
		rates, err := ResolveRates(mockCache, "gemini-1.5-pro")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		expectedInput := 1.250 / 1e6
		expectedCached := 0.125 / 1e6
		expectedOutput := 5.000 / 1e6
		if rates.InputRate != expectedInput || rates.CachedInputRate != expectedCached || rates.OutputRate != expectedOutput {
			t.Errorf("Expected rates %f/%f/%f, got %f/%f/%f", expectedInput, expectedCached, expectedOutput, rates.InputRate, rates.CachedInputRate, rates.OutputRate)
		}
	})

	t.Run("defaults to 10%% for gemini without explicit cached rate", func(t *testing.T) {
		rates, err := ResolveRates(mockCache, "gemini-3.5-flash")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		expectedInput := 0.075 / 1e6
		expectedCached := expectedInput * 0.1
		if rates.CachedInputRate != expectedCached {
			t.Errorf("Expected cached rate %f, got %f", expectedCached, rates.CachedInputRate)
		}
	})

	t.Run("defaults to 100%% for non-gemini without explicit cached rate", func(t *testing.T) {
		rates, err := ResolveRates(mockCache, "claude-3-5-sonnet")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		expectedInput := 3.000 / 1e6
		expectedCached := expectedInput
		if rates.CachedInputRate != expectedCached {
			t.Errorf("Expected cached rate %f, got %f", expectedCached, rates.CachedInputRate)
		}
	})

	t.Run("prefix or contains match", func(t *testing.T) {
		rates, err := ResolveRates(mockCache, "gemini-3.5-flash-high")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		expectedInput := 0.075 / 1e6
		expectedOutput := 0.300 / 1e6
		if rates.InputRate != expectedInput || rates.OutputRate != expectedOutput {
			t.Errorf("Expected rates %f/%f, got %f/%f", expectedInput, expectedOutput, rates.InputRate, rates.OutputRate)
		}
	})

	t.Run("spaces and parentheses normalization", func(t *testing.T) {
		rates, err := ResolveRates(mockCache, "Gemini 3.5 Flash (High)")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		expectedInput := 0.075 / 1e6
		expectedOutput := 0.300 / 1e6
		if rates.InputRate != expectedInput || rates.OutputRate != expectedOutput {
			t.Errorf("Expected rates %f/%f, got %f/%f", expectedInput, expectedOutput, rates.InputRate, rates.OutputRate)
		}
	})

	t.Run("missing model in cache", func(t *testing.T) {
		_, err := ResolveRates(mockCache, "gemini-4.0-ultra")
		if !errors.Is(err, ErrNoPricingCache) {
			t.Errorf("Expected ErrNoPricingCache, got: %v", err)
		}
	})

	t.Run("nil cache", func(t *testing.T) {
		_, err := ResolveRates(nil, "gemini-1.5-pro")
		if !errors.Is(err, ErrNoPricingCache) {
			t.Errorf("Expected ErrNoPricingCache, got: %v", err)
		}
	})
}

func TestCalculateCost(t *testing.T) {
	rates := Rates{
		InputRate:       0.075 / 1e6,
		CachedInputRate: 0.0075 / 1e6,
		OutputRate:      0.300 / 1e6,
	}

	// 200k standard input tokens, 800k cached input tokens, and 2M output tokens
	cost := CalculateCost(200000, 800000, 2000000, rates)
	expectedCost := (200000.0 * rates.InputRate) + (800000.0 * rates.CachedInputRate) + (2000000.0 * rates.OutputRate)
	diff := cost - expectedCost
	if diff < 0 {
		diff = -diff
	}
	if diff > 1e-9 {
		t.Errorf("Expected cost %f, got %f (diff: %e)", expectedCost, cost, diff)
	}
}

func TestGetDefaultPricing(t *testing.T) {
	cache, err := GetDefaultPricing()
	if err != nil {
		t.Fatalf("Expected no error from GetDefaultPricing, got: %v", err)
	}

	if cache == nil {
		t.Fatal("Expected pricing cache to be non-nil")
	}

	if len(cache.Models) == 0 {
		t.Error("Expected models to be populated in default pricing cache")
	}

	// Verify our key models exist
	expectedModels := []string{"gemini-3.5-flash", "gemini-3.1-pro"}
	for _, model := range expectedModels {
		if _, ok := cache.Models[model]; !ok {
			t.Errorf("Expected model %q in default pricing cache", model)
		}
	}
}
