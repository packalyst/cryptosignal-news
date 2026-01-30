// Package sources provides the complete list of crypto news RSS sources
// for the CryptoSignal News aggregation service.
package sources

import (
	"strings"
	"sync"
)

// FeedSource represents a single RSS feed source for crypto news
type FeedSource struct {
	Key        string   // Unique identifier (lowercase, no spaces)
	Name       string   // Display name
	RSSURL     string   // RSS feed URL
	WebsiteURL string   // Main website URL
	Category   string   // Primary category (general, bitcoin, defi, etc.)
	Language   string   // ISO 639-1 code: "en", "ko", "zh", "ja", "es", "pt", etc.
	Region     string   // Geographic region: "global", "asia", "europe", "latam", "na"
	IsPremium  bool     // Whether this is a premium-only source
	Tags       []string // Additional tags for filtering
	IsEnabled  bool     // Whether the source is currently enabled
}

var (
	// allFeedSources holds the complete list of sources
	allFeedSources []FeedSource
	// sourcesByKey provides O(1) lookup by key
	sourcesByKey map[string]*FeedSource
	// initOnce ensures sources are initialized only once
	initOnce sync.Once
)

// initFeedSources combines all sources from different files
func initFeedSources() {
	initOnce.Do(func() {
		// Combine English and International sources
		allFeedSources = make([]FeedSource, 0, 200)
		allFeedSources = append(allFeedSources, englishSources...)
		allFeedSources = append(allFeedSources, internationalSources...)

		// Build the lookup map
		sourcesByKey = make(map[string]*FeedSource, len(allFeedSources))
		for i := range allFeedSources {
			sourcesByKey[allFeedSources[i].Key] = &allFeedSources[i]
		}
	})
}

// GetAllFeedSources returns all registered sources
func GetAllFeedSources() []FeedSource {
	initFeedSources()
	result := make([]FeedSource, len(allFeedSources))
	copy(result, allFeedSources)
	return result
}

// GetFeedSourcesByLanguage returns all sources matching the given language code
func GetFeedSourcesByLanguage(lang string) []FeedSource {
	initFeedSources()
	lang = strings.ToLower(lang)
	result := make([]FeedSource, 0)
	for _, s := range allFeedSources {
		if strings.ToLower(s.Language) == lang {
			result = append(result, s)
		}
	}
	return result
}

// GetFeedSourcesByCategory returns all sources matching the given category
func GetFeedSourcesByCategory(cat string) []FeedSource {
	initFeedSources()
	cat = strings.ToLower(cat)
	result := make([]FeedSource, 0)
	for _, s := range allFeedSources {
		if strings.ToLower(s.Category) == cat {
			result = append(result, s)
		}
	}
	return result
}

// GetFeedSourcesByRegion returns all sources from the given region
func GetFeedSourcesByRegion(region string) []FeedSource {
	initFeedSources()
	region = strings.ToLower(region)
	result := make([]FeedSource, 0)
	for _, s := range allFeedSources {
		if strings.ToLower(s.Region) == region {
			result = append(result, s)
		}
	}
	return result
}

// GetEnabledFeedSources returns only enabled sources
func GetEnabledFeedSources() []FeedSource {
	initFeedSources()
	result := make([]FeedSource, 0)
	for _, s := range allFeedSources {
		if s.IsEnabled {
			result = append(result, s)
		}
	}
	return result
}

// GetPremiumFeedSources returns only premium sources
func GetPremiumFeedSources() []FeedSource {
	initFeedSources()
	result := make([]FeedSource, 0)
	for _, s := range allFeedSources {
		if s.IsPremium {
			result = append(result, s)
		}
	}
	return result
}

// GetFreeFeedSources returns only non-premium sources
func GetFreeFeedSources() []FeedSource {
	initFeedSources()
	result := make([]FeedSource, 0)
	for _, s := range allFeedSources {
		if !s.IsPremium {
			result = append(result, s)
		}
	}
	return result
}

// GetFeedSourceByKey returns a source by its unique key, or nil if not found
func GetFeedSourceByKey(key string) *FeedSource {
	initFeedSources()
	key = strings.ToLower(key)
	if source, exists := sourcesByKey[key]; exists {
		// Return a copy to prevent modification
		sourceCopy := *source
		return &sourceCopy
	}
	return nil
}

// GetFeedSourcesByTag returns all sources that have the given tag
func GetFeedSourcesByTag(tag string) []FeedSource {
	initFeedSources()
	tag = strings.ToLower(tag)
	result := make([]FeedSource, 0)
	for _, s := range allFeedSources {
		for _, t := range s.Tags {
			if strings.ToLower(t) == tag {
				result = append(result, s)
				break
			}
		}
	}
	return result
}

// GetFeedSourceCount returns the total number of registered sources
func GetFeedSourceCount() int {
	initFeedSources()
	return len(allFeedSources)
}

// GetLanguages returns a list of all unique language codes
func GetLanguages() []string {
	initFeedSources()
	langMap := make(map[string]bool)
	for _, s := range allFeedSources {
		langMap[s.Language] = true
	}
	result := make([]string, 0, len(langMap))
	for lang := range langMap {
		result = append(result, lang)
	}
	return result
}

// GetRegions returns a list of all unique regions
func GetRegions() []string {
	initFeedSources()
	regionMap := make(map[string]bool)
	for _, s := range allFeedSources {
		regionMap[s.Region] = true
	}
	result := make([]string, 0, len(regionMap))
	for region := range regionMap {
		result = append(result, region)
	}
	return result
}
