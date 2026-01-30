package ai

import (
	"bytes"
	"text/template"
)

// SentimentPrompt is the template for sentiment analysis
const SentimentPrompt = `Analyze the sentiment of this crypto news article.

Title: {{.Title}}
Description: {{.Description}}

Respond with ONLY valid JSON:
{
  "sentiment": "bullish" | "bearish" | "neutral",
  "score": <float -1.0 to 1.0>,
  "confidence": <float 0.0 to 1.0>,
  "reasoning": "<brief explanation>",
  "coins_mentioned": ["BTC", "ETH", ...]
}`

// SummaryPrompt is the template for daily market summaries
const SummaryPrompt = `Summarize these {{.Count}} crypto news articles from the last 24 hours.

Articles:
{{range .Articles}}
- {{.Title}} ({{.Source}})
{{end}}

Create a market summary with:
1. Overall market sentiment (bullish/bearish/neutral)
2. Top 3-5 key developments
3. Notable price movements mentioned
4. Any regulatory news

Respond with ONLY valid JSON:
{
  "overall_sentiment": "bullish" | "bearish" | "neutral",
  "summary": "<2-3 paragraph summary>",
  "key_developments": ["...", "..."],
  "mentioned_coins": ["BTC", "ETH", ...],
  "notable_events": ["...", "..."]
}`

// SignalsPrompt is the template for trading signal generation
const SignalsPrompt = `Based on these recent crypto news articles, identify potential trading signals.

Articles:
{{range .Articles}}
- {{.Title}} ({{.Source}}, {{.TimeAgo}})
{{end}}

Identify news that might impact prices. Respond with ONLY valid JSON:
{
  "signals": [
    {
      "coin": "BTC",
      "direction": "bullish" | "bearish",
      "strength": "strong" | "moderate" | "weak",
      "catalyst": "<brief description>",
      "source_title": "<article title>"
    }
  ],
  "market_mood": "risk_on" | "risk_off" | "neutral"
}`

// AnalyzeTextPrompt is the template for custom text analysis
const AnalyzeTextPrompt = `Analyze the following crypto-related text and provide insights.

Text: {{.Text}}

Provide analysis including sentiment, key topics, and any actionable insights.

Respond with ONLY valid JSON:
{
  "sentiment": "bullish" | "bearish" | "neutral",
  "score": <float -1.0 to 1.0>,
  "key_topics": ["...", "..."],
  "coins_mentioned": ["BTC", "ETH", ...],
  "insights": "<analysis and insights>",
  "actionable": <boolean>
}`

// SentimentData holds data for sentiment prompt
type SentimentData struct {
	Title       string
	Description string
}

// SummaryData holds data for summary prompt
type SummaryData struct {
	Count    int
	Articles []ArticleSummary
}

// ArticleSummary is a simplified article for prompts
type ArticleSummary struct {
	Title   string
	Source  string
	TimeAgo string
}

// SignalsData holds data for signals prompt
type SignalsData struct {
	Articles []ArticleSummary
}

// AnalyzeTextData holds data for custom text analysis
type AnalyzeTextData struct {
	Text string
}

// RenderPrompt renders a template with the provided data
func RenderPrompt(tmpl string, data interface{}) (string, error) {
	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// RenderSentimentPrompt renders the sentiment analysis prompt
func RenderSentimentPrompt(title, description string) (string, error) {
	return RenderPrompt(SentimentPrompt, SentimentData{
		Title:       title,
		Description: description,
	})
}

// RenderSummaryPrompt renders the market summary prompt
func RenderSummaryPrompt(articles []ArticleSummary) (string, error) {
	return RenderPrompt(SummaryPrompt, SummaryData{
		Count:    len(articles),
		Articles: articles,
	})
}

// RenderSignalsPrompt renders the trading signals prompt
func RenderSignalsPrompt(articles []ArticleSummary) (string, error) {
	return RenderPrompt(SignalsPrompt, SignalsData{
		Articles: articles,
	})
}

// RenderAnalyzeTextPrompt renders the custom text analysis prompt
func RenderAnalyzeTextPrompt(text string) (string, error) {
	return RenderPrompt(AnalyzeTextPrompt, AnalyzeTextData{
		Text: text,
	})
}
