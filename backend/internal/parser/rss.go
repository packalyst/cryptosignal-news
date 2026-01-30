package parser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// FeedParser parses RSS, Atom, and JSON feeds
type FeedParser struct {
	parser     *gofeed.Parser
	httpClient *http.Client
	userAgent  string
}

// Feed represents a parsed feed
type Feed struct {
	Title       string
	Link        string
	Description string
	Language    string
	Items       []FeedItem
	FeedType    string
}

// FeedItem represents a single item from a feed
type FeedItem struct {
	GUID        string
	Title       string
	Link        string
	Description string
	Content     string
	PubDate     time.Time
	Categories  []string
	Author      string
	ImageURL    string
}

// NewFeedParser creates a new feed parser
func NewFeedParser() *FeedParser {
	return &FeedParser{
		parser: gofeed.NewParser(),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		userAgent: "CryptoSignalNews/1.0 (+https://cryptosignal.news)",
	}
}

// NewFeedParserWithClient creates a parser with a custom HTTP client
func NewFeedParserWithClient(client *http.Client) *FeedParser {
	return &FeedParser{
		parser:     gofeed.NewParser(),
		httpClient: client,
		userAgent:  "CryptoSignalNews/1.0 (+https://cryptosignal.news)",
	}
}

// Parse parses feed data from bytes
func (p *FeedParser) Parse(data []byte) (*Feed, error) {
	feed, err := p.parser.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	return p.convertFeed(feed), nil
}

// ParseURL fetches and parses a feed from a URL
func (p *FeedParser) ParseURL(ctx context.Context, url string) (*Feed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", p.userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	// Limit response size to 10MB
	limitedReader := io.LimitReader(resp.Body, 10*1024*1024)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read feed body: %w", err)
	}

	return p.Parse(data)
}

// convertFeed converts gofeed.Feed to our Feed struct
func (p *FeedParser) convertFeed(gf *gofeed.Feed) *Feed {
	feed := &Feed{
		Title:       gf.Title,
		Link:        gf.Link,
		Description: gf.Description,
		Language:    gf.Language,
		FeedType:    gf.FeedType,
		Items:       make([]FeedItem, 0, len(gf.Items)),
	}

	for _, item := range gf.Items {
		feedItem := p.convertItem(item)
		feed.Items = append(feed.Items, feedItem)
	}

	return feed
}

// convertItem converts gofeed.Item to our FeedItem struct
func (p *FeedParser) convertItem(item *gofeed.Item) FeedItem {
	fi := FeedItem{
		GUID:        p.extractGUID(item),
		Title:       strings.TrimSpace(item.Title),
		Link:        strings.TrimSpace(item.Link),
		Description: item.Description,
		Content:     item.Content,
		Categories:  item.Categories,
	}

	// Extract publication date
	if item.PublishedParsed != nil {
		fi.PubDate = *item.PublishedParsed
	} else if item.UpdatedParsed != nil {
		fi.PubDate = *item.UpdatedParsed
	} else {
		// Try to parse from string
		fi.PubDate = p.parseDateString(item.Published, item.Updated)
	}

	// Ensure we have a valid time (not zero)
	if fi.PubDate.IsZero() {
		fi.PubDate = time.Now().UTC()
	}

	// Extract author
	if item.Author != nil {
		fi.Author = item.Author.Name
	} else if len(item.Authors) > 0 && item.Authors[0] != nil {
		fi.Author = item.Authors[0].Name
	}

	// Extract image
	if item.Image != nil {
		fi.ImageURL = item.Image.URL
	}

	// Ensure categories is not nil
	if fi.Categories == nil {
		fi.Categories = []string{}
	}

	return fi
}

// extractGUID extracts or generates a GUID for the item
func (p *FeedParser) extractGUID(item *gofeed.Item) string {
	if item.GUID != "" {
		return item.GUID
	}
	if item.Link != "" {
		return item.Link
	}
	// Fallback: use title hash
	return fmt.Sprintf("generated-%x", hashString(item.Title+item.Published))
}

// parseDateString attempts to parse various date formats
func (p *FeedParser) parseDateString(dates ...string) time.Time {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		time.RFC3339Nano,
		time.RFC822Z,
		time.RFC822,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"02 Jan 2006 15:04:05 -0700",
		"2006-01-02",
	}

	for _, dateStr := range dates {
		if dateStr == "" {
			continue
		}
		dateStr = strings.TrimSpace(dateStr)
		for _, format := range formats {
			if t, err := time.Parse(format, dateStr); err == nil {
				return t.UTC()
			}
		}
	}

	return time.Time{}
}

// hashString creates a simple hash of a string
func hashString(s string) uint32 {
	var h uint32
	for _, c := range s {
		h = 31*h + uint32(c)
	}
	return h
}

// GetDescription returns the best description for a feed item
// Prefers content over description, cleans HTML
func (fi *FeedItem) GetDescription() string {
	if fi.Content != "" {
		return fi.Content
	}
	return fi.Description
}

// GetCleanDescription returns a cleaned description without HTML
func (fi *FeedItem) GetCleanDescription(cleaner *Cleaner, maxLen int) string {
	desc := fi.GetDescription()
	if cleaner != nil {
		desc = cleaner.Clean(desc)
	}
	if maxLen > 0 && len(desc) > maxLen {
		desc = desc[:maxLen-3] + "..."
	}
	return desc
}
