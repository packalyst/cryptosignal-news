package models

import (
	"strconv"
	"time"
)

// Source represents a news source
type Source struct {
	ID               int        `json:"id" db:"id"`
	Key              string     `json:"key" db:"key"`
	Name             string     `json:"name" db:"name"`
	RSSURL           string     `json:"rss_url" db:"rss_url"`
	WebsiteURL       string     `json:"website_url,omitempty" db:"website_url"`
	Category         string     `json:"category,omitempty" db:"category"`
	Language         string     `json:"language" db:"language"`
	IsEnabled        bool       `json:"is_enabled" db:"is_enabled"`
	ReliabilityScore float64    `json:"reliability_score" db:"reliability_score"`
	LastFetchAt      *time.Time `json:"last_fetch_at,omitempty" db:"last_fetch_at"`
	ErrorCount       int        `json:"error_count" db:"error_count"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// SourceStats contains statistics about a source's fetch performance
type SourceStats struct {
	SourceID        int       `json:"source_id"`
	SourceKey       string    `json:"source_key"`
	ArticlesFetched int       `json:"articles_fetched"`
	ErrorCount      int       `json:"error_count"`
	LastFetchAt     time.Time `json:"last_fetch_at"`
	AvgFetchTime    float64   `json:"avg_fetch_time_ms"`
}

// SourceResponse is the API response format for a source
type SourceResponse struct {
	ID               int        `json:"id"`
	Key              string     `json:"key"`
	Name             string     `json:"name"`
	WebsiteURL       string     `json:"website_url,omitempty"`
	Category         string     `json:"category,omitempty"`
	Language         string     `json:"language"`
	IsEnabled        bool       `json:"is_enabled"`
	ReliabilityScore float64    `json:"reliability_score"`
	LastFetchAt      *time.Time `json:"last_fetch_at,omitempty"`
	ArticleCount     int        `json:"article_count,omitempty"`
}

// ToResponse converts a Source to SourceResponse
func (s *Source) ToResponse() SourceResponse {
	return SourceResponse{
		ID:               s.ID,
		Key:              s.Key,
		Name:             s.Name,
		WebsiteURL:       s.WebsiteURL,
		Category:         s.Category,
		Language:         s.Language,
		IsEnabled:        s.IsEnabled,
		ReliabilityScore: s.ReliabilityScore,
		LastFetchAt:      s.LastFetchAt,
	}
}

// Article represents a news article
type Article struct {
	ID             int64     `json:"id" db:"id"`
	SourceID       int       `json:"source_id" db:"source_id"`
	GUID           string    `json:"guid" db:"guid"`
	Title          string    `json:"title" db:"title"`
	Link           string    `json:"link" db:"link"`
	Description    string    `json:"description,omitempty" db:"description"`
	PubDate        time.Time `json:"pub_date" db:"pub_date"`
	Categories     []string  `json:"categories" db:"categories"`
	Sentiment      string    `json:"sentiment,omitempty" db:"sentiment"`
	SentimentScore float64   `json:"sentiment_score,omitempty" db:"sentiment_score"`
	MentionedCoins []string  `json:"mentioned_coins" db:"mentioned_coins"`
	IsBreaking     bool      `json:"is_breaking" db:"is_breaking"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`

	// Translation fields
	OriginalTitle       string `json:"original_title,omitempty" db:"original_title"`
	OriginalDescription string `json:"original_description,omitempty" db:"original_description"`
	OriginalLanguage    string `json:"original_language,omitempty" db:"original_language"`
	TranslationStatus   string `json:"translation_status,omitempty" db:"translation_status"`

	// Joined fields
	SourceName string `json:"source_name,omitempty" db:"source_name"`
	SourceKey  string `json:"source_key,omitempty" db:"source_key"`
}

// ArticleFilter contains filter options for querying articles
type ArticleFilter struct {
	SourceID   int
	SourceKey  string
	Category   string
	Coin       string
	Query      string
	IsBreaking *bool
	Since      *time.Time
	Before     *time.Time
	Limit      int
	Offset     int
}

// ArticleResponse is the API response format for an article
type ArticleResponse struct {
	ID             int64    `json:"id"`
	Title          string   `json:"title"`
	Link           string   `json:"link"`
	Description    string   `json:"description,omitempty"`
	Source         string   `json:"source"`
	SourceKey      string   `json:"source_key"`
	Categories     []string `json:"categories,omitempty"`
	PubDate        string   `json:"pub_date"`
	TimeAgo        string   `json:"time_ago"`
	Sentiment      string   `json:"sentiment,omitempty"`
	SentimentScore float64  `json:"sentiment_score,omitempty"`
	MentionedCoins []string `json:"mentioned_coins,omitempty"`
	IsBreaking     bool     `json:"is_breaking"`
}

// ToResponse converts an Article to ArticleResponse (shows all categories)
func (a *Article) ToResponse() ArticleResponse {
	return a.ToResponseWithFilter(nil)
}

// ToResponseWithFilter converts an Article to ArticleResponse
// If filterCategories is provided, only shows categories that match the filter
func (a *Article) ToResponseWithFilter(filterCategories []string) ArticleResponse {
	resp := ArticleResponse{
		ID:             a.ID,
		Title:          a.Title,
		Link:           a.Link,
		Description:    a.Description,
		Source:         a.SourceName,
		SourceKey:      a.SourceKey,
		PubDate:        a.PubDate.Format(time.RFC3339),
		TimeAgo:        timeAgo(a.PubDate),
		Sentiment:      a.Sentiment,
		SentimentScore: a.SentimentScore,
		IsBreaking:     a.IsBreaking,
	}

	if len(a.MentionedCoins) > 0 {
		resp.MentionedCoins = a.MentionedCoins
	}

	if len(a.Categories) > 0 {
		if len(filterCategories) > 0 {
			// Only include categories that match the filter
			filterSet := make(map[string]bool, len(filterCategories))
			for _, fc := range filterCategories {
				filterSet[fc] = true
			}
			matched := []string{}
			for _, cat := range a.Categories {
				if filterSet[cat] {
					matched = append(matched, cat)
				}
			}
			if len(matched) > 0 {
				resp.Categories = matched
			}
		} else {
			// No filter, show all categories
			resp.Categories = a.Categories
		}
	}

	return resp
}

// Category represents a news category with count
type Category struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// timeAgo returns a human-readable time difference
func timeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return formatDuration(mins, "m")
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return formatDuration(hours, "h")
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return formatDuration(days, "d")
	default:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1w ago"
		}
		return formatDuration(weeks, "w")
	}
}

func formatDuration(value int, unit string) string {
	return strconv.Itoa(value) + unit + " ago"
}

// IsHealthy returns true if the source is enabled and has a low error count
func (s *Source) IsHealthy() bool {
	return s.IsEnabled && s.ErrorCount < 5
}

// NeedsBackoff returns true if the source has too many errors
func (s *Source) NeedsBackoff() bool {
	return s.ErrorCount >= 3
}

// GetBackoffDuration returns how long to wait before retrying this source
func (s *Source) GetBackoffDuration() time.Duration {
	if s.ErrorCount < 3 {
		return 0
	}
	// Exponential backoff: 5min, 15min, 30min, 1h, 2h, etc.
	minutes := 5 * (1 << (s.ErrorCount - 3))
	if minutes > 120 {
		minutes = 120
	}
	return time.Duration(minutes) * time.Minute
}

// Translation status constants
const (
	TranslationNone      = "none"      // English, no translation needed
	TranslationPending   = "pending"   // Needs translation
	TranslationCompleted = "completed" // Successfully translated
	TranslationFailed    = "failed"    // Translation failed
)

// NewArticle creates a new article with sensible defaults
func NewArticle(sourceID int, guid, title, link string, pubDate time.Time) *Article {
	return &Article{
		SourceID:          sourceID,
		GUID:              guid,
		Title:             title,
		Link:              link,
		PubDate:           pubDate,
		Categories:        []string{},
		MentionedCoins:    []string{},
		TranslationStatus: TranslationNone,
		CreatedAt:         time.Now().UTC(),
	}
}

// SetForTranslation marks an article as needing translation and stores originals
func (a *Article) SetForTranslation(language string) {
	a.OriginalTitle = a.Title
	a.OriginalDescription = a.Description
	a.OriginalLanguage = language
	a.TranslationStatus = TranslationPending
}

// SetDescription sets and sanitizes the article description
func (a *Article) SetDescription(desc string) {
	a.Description = desc
}

// SetCategories sets the article categories
func (a *Article) SetCategories(cats []string) {
	if cats == nil {
		a.Categories = []string{}
		return
	}
	a.Categories = cats
}

// SetMentionedCoins sets the detected cryptocurrency mentions
func (a *Article) SetMentionedCoins(coins []string) {
	if coins == nil {
		a.MentionedCoins = []string{}
		return
	}
	a.MentionedCoins = coins
}
