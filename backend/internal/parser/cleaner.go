package parser

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

// Cleaner provides text cleaning utilities
type Cleaner struct {
	htmlTagRegex     *regexp.Regexp
	whitespaceRegex  *regexp.Regexp
	multiSpaceRegex  *regexp.Regexp
	urlRegex         *regexp.Regexp
	cdataStartRegex  *regexp.Regexp
	cdataEndRegex    *regexp.Regexp
}

// NewCleaner creates a new text cleaner
func NewCleaner() *Cleaner {
	return &Cleaner{
		htmlTagRegex:    regexp.MustCompile(`<[^>]*>`),
		whitespaceRegex: regexp.MustCompile(`[\r\n\t]+`),
		multiSpaceRegex: regexp.MustCompile(`\s{2,}`),
		urlRegex:        regexp.MustCompile(`https?://\S+`),
		cdataStartRegex: regexp.MustCompile(`<!\[CDATA\[`),
		cdataEndRegex:   regexp.MustCompile(`\]\]>`),
	}
}

// Clean removes HTML tags and normalizes whitespace
func (c *Cleaner) Clean(text string) string {
	if text == "" {
		return ""
	}

	// Remove CDATA markers
	text = c.cdataStartRegex.ReplaceAllString(text, "")
	text = c.cdataEndRegex.ReplaceAllString(text, "")

	// Strip HTML tags
	text = c.htmlTagRegex.ReplaceAllString(text, " ")

	// Decode HTML entities (multiple passes for nested entities)
	for i := 0; i < 3; i++ {
		decoded := html.UnescapeString(text)
		if decoded == text {
			break
		}
		text = decoded
	}

	// Normalize whitespace
	text = c.whitespaceRegex.ReplaceAllString(text, " ")
	text = c.multiSpaceRegex.ReplaceAllString(text, " ")

	// Trim
	text = strings.TrimSpace(text)

	return text
}

// StripHTMLTags removes all HTML tags from text
func (c *Cleaner) StripHTMLTags(text string) string {
	return c.htmlTagRegex.ReplaceAllString(text, "")
}

// DecodeHTMLEntities decodes HTML entities
func (c *Cleaner) DecodeHTMLEntities(text string) string {
	return html.UnescapeString(text)
}

// NormalizeWhitespace replaces multiple whitespace with single space
func (c *Cleaner) NormalizeWhitespace(text string) string {
	text = c.whitespaceRegex.ReplaceAllString(text, " ")
	text = c.multiSpaceRegex.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// Truncate truncates text to maxLen characters, adding ellipsis if needed
func (c *Cleaner) Truncate(text string, maxLen int) string {
	if maxLen <= 0 || len(text) <= maxLen {
		return text
	}

	// Find a good break point
	truncated := text[:maxLen-3]

	// Try to break at word boundary
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLen/2 {
		truncated = truncated[:lastSpace]
	}

	return strings.TrimSpace(truncated) + "..."
}

// TruncateRunes truncates text to maxLen runes (unicode-aware)
func (c *Cleaner) TruncateRunes(text string, maxLen int) string {
	if maxLen <= 0 {
		return text
	}

	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}

	// Find word boundary
	truncated := runes[:maxLen-3]
	lastSpace := -1
	for i := len(truncated) - 1; i >= maxLen/2; i-- {
		if unicode.IsSpace(truncated[i]) {
			lastSpace = i
			break
		}
	}

	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return strings.TrimSpace(string(truncated)) + "..."
}

// RemoveURLs removes URLs from text
func (c *Cleaner) RemoveURLs(text string) string {
	return c.urlRegex.ReplaceAllString(text, "")
}

// ExtractURLs extracts all URLs from text
func (c *Cleaner) ExtractURLs(text string) []string {
	return c.urlRegex.FindAllString(text, -1)
}

// CleanTitle cleans a title string
func (c *Cleaner) CleanTitle(title string) string {
	// Remove HTML and entities
	title = c.Clean(title)

	// Remove common prefixes
	prefixes := []string{"BREAKING:", "ALERT:", "UPDATE:", "WATCH:", "JUST IN:"}
	upper := strings.ToUpper(title)
	for _, prefix := range prefixes {
		if strings.HasPrefix(upper, prefix) {
			title = strings.TrimSpace(title[len(prefix):])
			break
		}
	}

	return title
}

// SanitizeForDB prepares text for database storage
func (c *Cleaner) SanitizeForDB(text string, maxLen int) string {
	// Clean the text
	text = c.Clean(text)

	// Remove null bytes
	text = strings.ReplaceAll(text, "\x00", "")

	// Truncate if needed
	if maxLen > 0 {
		text = c.Truncate(text, maxLen)
	}

	return text
}

// ExtractFirstParagraph extracts the first paragraph from text
func (c *Cleaner) ExtractFirstParagraph(text string) string {
	// First clean the text
	text = c.Clean(text)

	// Split by common paragraph markers
	markers := []string{"\n\n", ". ", ".\n"}
	for _, marker := range markers {
		idx := strings.Index(text, marker)
		if idx > 50 { // At least 50 chars
			return strings.TrimSpace(text[:idx+1])
		}
	}

	// Return first 500 chars if no good break point
	return c.Truncate(text, 500)
}
