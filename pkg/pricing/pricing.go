package pricing

import (
	"errors"
	"strings"
)

var ErrNoPricingCache = errors.New("pricing cache file not found or unreadable")

type Rates struct {
	InputRate  float64
	OutputRate float64
}

type ModelRate struct {
	InputPricePer1M  float64 `json:"input_price_per_1m"`
	OutputPricePer1M float64 `json:"output_price_per_1m"`
}

type PricingCache struct {
	LastFetched string               `json:"last_fetched"`
	Models      map[string]ModelRate `json:"models"`
}

func ResolveRates(cache *PricingCache, modelID string) (Rates, error) {
	if cache == nil || cache.Models == nil {
		return Rates{}, ErrNoPricingCache
	}

	id := strings.ToLower(modelID)

	if rate, exists := cache.Models[id]; exists {
		return Rates{
			InputRate:  rate.InputPricePer1M / 1e6,
			OutputRate: rate.OutputPricePer1M / 1e6,
		}, nil
	}

	for modelKey, rate := range cache.Models {
		keyLower := strings.ToLower(modelKey)
		if strings.Contains(id, keyLower) || strings.Contains(keyLower, id) {
			return Rates{
				InputRate:  rate.InputPricePer1M / 1e6,
				OutputRate: rate.OutputPricePer1M / 1e6,
			}, nil
		}
	}

	return Rates{}, ErrNoPricingCache
}

func CalculateCost(input, output int64, rates Rates) float64 {
	return (float64(input) * rates.InputRate) + (float64(output) * rates.OutputRate)
}
