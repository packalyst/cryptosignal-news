package sources

import (
	"github.com/cryptosignal-news/backend/internal/models"
)

// Source represents an RSS feed source that can be fetched
type Source interface {
	// GetID returns the source database ID
	GetID() int

	// GetKey returns the source unique key
	GetKey() string

	// GetName returns the source display name
	GetName() string

	// GetURL returns the RSS feed URL
	GetURL() string

	// GetCategory returns the source category
	GetCategory() string

	// IsEnabled returns whether the source is active
	IsEnabled() bool
}

// DBSource wraps a models.Source to implement the Source interface
type DBSource struct {
	*models.Source
}

// NewDBSource creates a Source from a database model
func NewDBSource(s *models.Source) Source {
	return &DBSource{Source: s}
}

// GetID returns the source database ID
func (s *DBSource) GetID() int {
	return s.ID
}

// GetKey returns the source unique key
func (s *DBSource) GetKey() string {
	return s.Key
}

// GetName returns the source display name
func (s *DBSource) GetName() string {
	return s.Name
}

// GetURL returns the RSS feed URL
func (s *DBSource) GetURL() string {
	return s.RSSURL
}

// GetCategory returns the source category
func (s *DBSource) GetCategory() string {
	return s.Category
}

// IsEnabled returns whether the source is active
func (s *DBSource) IsEnabled() bool {
	return s.Source.IsEnabled
}

// ConvertSources converts a slice of model sources to Source interfaces
func ConvertSources(sources []models.Source) []Source {
	result := make([]Source, len(sources))
	for i := range sources {
		result[i] = NewDBSource(&sources[i])
	}
	return result
}
