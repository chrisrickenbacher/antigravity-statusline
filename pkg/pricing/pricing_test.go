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
				InputPricePer1M:  1.250,
				OutputPricePer1M: 5.000,
			},
			"gemini-3.5-flash": {
				InputPricePer1M:  0.075,
				OutputPricePer1M: 0.300,
			},
		},
	}

	t.Run("exact match case-sensitive and case-insensitive", func(t *testing.T) {
		rates, err := ResolveRates(mockCache, "gemini-1.5-pro")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		expectedInput := 1.250 / 1e6
		expectedOutput := 5.000 / 1e6
		if rates.InputRate != expectedInput || rates.OutputRate != expectedOutput {
			t.Errorf("Expected rates %f/%f, got %f/%f", expectedInput, expectedOutput, rates.InputRate, rates.OutputRate)
		}

		ratesUpper, err := ResolveRates(mockCache, "GEMINI-1.5-PRO")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if ratesUpper.InputRate != expectedInput || ratesUpper.OutputRate != expectedOutput {
			t.Errorf("Expected rates %f/%f, got %f/%f", expectedInput, expectedOutput, ratesUpper.InputRate, ratesUpper.OutputRate)
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
		InputRate:  0.075 / 1e6,
		OutputRate: 0.300 / 1e6,
	}

	cost := CalculateCost(1000000, 2000000, rates)
	expectedCost := 0.075 + 0.600
	diff := cost - expectedCost
	if diff < 0 {
		diff = -diff
	}
	if diff > 1e-9 {
		t.Errorf("Expected cost %f, got %f (diff: %e)", expectedCost, cost, diff)
	}
}
