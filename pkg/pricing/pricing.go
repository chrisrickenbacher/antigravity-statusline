package pricing

import (
	_ "embed"
	"encoding/json"
	"errors"
	"strings"
)

var ErrNoPricingCache = errors.New("pricing cache file not found or unreadable")

//go:embed pricing.json
var defaultPricingBytes []byte

// GetDefaultPricing returns a copy of the embedded default pricing cache.
func GetDefaultPricing() (*PricingCache, error) {
	var cache PricingCache
	if err := json.Unmarshal(defaultPricingBytes, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

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

func normalizeModelID(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "(high)", "")
	s = strings.ReplaceAll(s, "high", "")
	return s
}

func ResolveRates(cache *PricingCache, modelID string) (Rates, error) {
	if cache == nil || cache.Models == nil {
		return Rates{}, ErrNoPricingCache
	}

	id := normalizeModelID(modelID)

	for modelKey, rate := range cache.Models {
		keyNorm := normalizeModelID(modelKey)
		if strings.Contains(id, keyNorm) || strings.Contains(keyNorm, id) {
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
