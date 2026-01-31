package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// TranslationResult represents the result of a translation
type TranslationResult struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	FromLang    string `json:"from_lang"`
}

// TranslatorService handles article translation using Groq
type TranslatorService struct {
	groq  *GroqClient
	cache *AICache
	model string
}

// NewTranslatorService creates a new translator service
func NewTranslatorService(groq *GroqClient, cache *AICache, model string) *TranslatorService {
	if model == "" {
		model = "llama-3.1-8b-instant" // Fast model with 500k tokens/day
	}
	return &TranslatorService{
		groq:  groq,
		cache: cache,
		model: model,
	}
}

// TranslateArticle translates an article's title and description to English
func (t *TranslatorService) TranslateArticle(ctx context.Context, title, description, fromLang string) (*TranslationResult, error) {
	// Don't translate if already English
	if strings.ToLower(fromLang) == "en" {
		return &TranslationResult{
			Title:       title,
			Description: description,
			FromLang:    fromLang,
		}, nil
	}

	// Map language codes to full names for the prompt
	langNames := map[string]string{
		"ko": "Korean",
		"zh": "Chinese",
		"ja": "Japanese",
		"es": "Spanish",
		"pt": "Portuguese",
		"de": "German",
		"fr": "French",
		"ru": "Russian",
		"tr": "Turkish",
		"it": "Italian",
		"nl": "Dutch",
		"pl": "Polish",
		"vi": "Vietnamese",
		"id": "Indonesian",
		"th": "Thai",
		"ar": "Arabic",
		"fa": "Persian",
		"uk": "Ukrainian",
	}

	langName := langNames[strings.ToLower(fromLang)]
	if langName == "" {
		langName = fromLang // Use code if name not found
	}

	// Truncate description if too long to save tokens
	desc := description
	if len(desc) > 2000 {
		desc = desc[:2000]
	}

	prompt := fmt.Sprintf(`Translate this %s cryptocurrency news article to English. Return ONLY valid JSON with "title" and "description" fields.

Title: %s

Description: %s

Response format:
{"title": "translated title", "description": "translated description"}`, langName, title, desc)

	req := &ChatRequest{
		Model:       t.model,
		Temperature: 0.3, // Lower temperature for accurate translations
		MaxTokens:   1024,
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "You are a professional translator specializing in cryptocurrency and financial news. Translate accurately while preserving technical terms and coin names. Respond ONLY with valid JSON.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	resp, err := t.groq.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}

	content := resp.GetMessageContent()
	content = cleanJSONResponse(content)

	var result TranslationResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// Try to extract JSON
		jsonStr := extractJSON(content)
		if jsonStr != "" {
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				// Return original if translation parsing fails
				log.Printf("warning: failed to parse translation: %v", err)
				return &TranslationResult{
					Title:       title,
					Description: description,
					FromLang:    fromLang,
				}, nil
			}
		} else {
			return &TranslationResult{
				Title:       title,
				Description: description,
				FromLang:    fromLang,
			}, nil
		}
	}

	result.FromLang = fromLang
	return &result, nil
}

// TranslateArticles translates multiple articles concurrently
// Returns a map of original title -> TranslationResult
func (t *TranslatorService) TranslateArticles(ctx context.Context, articles []ArticleToTranslate) map[string]*TranslationResult {
	results := make(map[string]*TranslationResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrent translations to avoid rate limiting
	semaphore := make(chan struct{}, 3)

	for _, article := range articles {
		// Skip English articles
		if strings.ToLower(article.Language) == "en" {
			continue
		}

		wg.Add(1)
		go func(a ArticleToTranslate) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Add small delay between requests
			time.Sleep(200 * time.Millisecond)

			result, err := t.TranslateArticle(ctx, a.Title, a.Description, a.Language)
			if err != nil {
				log.Printf("warning: failed to translate article '%s': %v", a.Title[:min(50, len(a.Title))], err)
				return
			}

			mu.Lock()
			results[a.Title] = result
			mu.Unlock()
		}(article)
	}

	wg.Wait()
	return results
}

// ArticleToTranslate represents an article that needs translation
type ArticleToTranslate struct {
	Title       string
	Description string
	Language    string
}

// NeedsTranslation checks if a language code needs translation
func NeedsTranslation(lang string) bool {
	return strings.ToLower(lang) != "en" && lang != ""
}

// min returns the smaller of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
