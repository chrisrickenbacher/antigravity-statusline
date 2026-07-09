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
	InputRate       float64
	CachedInputRate float64
	OutputRate      float64
}

type ModelRate struct {
	InputPricePer1M       float64 `json:"input_price_per_1m"`
	CachedInputPricePer1M float64 `json:"cached_input_price_per_1m"`
	OutputPricePer1M      float64 `json:"output_price_per_1m"`
}

type PricingCache struct {
	LastFetched string               `json:"last_fetched"`
	Models      map[string]ModelRate `json:"models"`
}

func NormalizeModelID(s string) string {
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

	id := NormalizeModelID(modelID)

	for modelKey, rate := range cache.Models {
		keyNorm := NormalizeModelID(modelKey)
		if strings.Contains(id, keyNorm) || strings.Contains(keyNorm, id) {
			inputRate := rate.InputPricePer1M / 1e6
			var cachedRate float64
			if rate.CachedInputPricePer1M > 0 {
				cachedRate = rate.CachedInputPricePer1M / 1e6
			} else if strings.Contains(id, "gemini") {
				cachedRate = inputRate * 0.1 // Default 90% discount for Gemini
			} else {
				cachedRate = inputRate // No caching discount by default
			}
			return Rates{
				InputRate:       inputRate,
				CachedInputRate: cachedRate,
				OutputRate:      rate.OutputPricePer1M / 1e6,
			}, nil
		}
	}

	return Rates{}, ErrNoPricingCache
}

func CalculateCost(input, cached, output int64, rates Rates) float64 {
	standard := input - cached
	if standard < 0 {
		standard = 0
	}
	return (float64(standard) * rates.InputRate) + (float64(cached) * rates.CachedInputRate) + (float64(output) * rates.OutputRate)
}
