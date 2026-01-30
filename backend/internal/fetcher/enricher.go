package fetcher

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"time"

	"github.com/cryptosignal-news/backend/internal/models"
)

// CoinPattern represents a cryptocurrency to detect in text
type CoinPattern struct {
	Symbol   string
	Names    []string
	Patterns []*regexp.Regexp
}

// Enricher provides article enrichment functionality
type Enricher struct {
	coinPatterns []CoinPattern
}

// NewEnricher creates a new article enricher
func NewEnricher() *Enricher {
	e := &Enricher{}
	e.initCoinPatterns()
	return e
}

// initCoinPatterns initializes the cryptocurrency detection patterns
func (e *Enricher) initCoinPatterns() {
	coins := []struct {
		symbol string
		names  []string
	}{
		{"BTC", []string{"bitcoin", "btc"}},
		{"ETH", []string{"ethereum", "ether", "eth"}},
		{"BNB", []string{"binance coin", "binance", "bnb"}},
		{"XRP", []string{"ripple", "xrp"}},
		{"SOL", []string{"solana", "sol"}},
		{"DOGE", []string{"dogecoin", "doge"}},
		{"ADA", []string{"cardano", "ada"}},
		{"AVAX", []string{"avalanche", "avax"}},
		{"DOT", []string{"polkadot", "dot"}},
		{"MATIC", []string{"polygon", "matic"}},
		{"LINK", []string{"chainlink", "link"}},
		{"UNI", []string{"uniswap", "uni"}},
		{"ATOM", []string{"cosmos", "atom"}},
		{"LTC", []string{"litecoin", "ltc"}},
		{"ETC", []string{"ethereum classic", "etc"}},
		{"XLM", []string{"stellar", "xlm"}},
		{"ALGO", []string{"algorand", "algo"}},
		{"VET", []string{"vechain", "vet"}},
		{"FIL", []string{"filecoin", "fil"}},
		{"NEAR", []string{"near protocol", "near"}},
		{"APT", []string{"aptos", "apt"}},
		{"ARB", []string{"arbitrum", "arb"}},
		{"OP", []string{"optimism"}},
		{"SUI", []string{"sui"}},
		{"SEI", []string{"sei"}},
		{"TIA", []string{"celestia", "tia"}},
		{"INJ", []string{"injective", "inj"}},
		{"PEPE", []string{"pepe"}},
		{"SHIB", []string{"shiba inu", "shib"}},
		{"BONK", []string{"bonk"}},
		{"WIF", []string{"dogwifhat", "wif"}},
		{"USDT", []string{"tether", "usdt"}},
		{"USDC", []string{"usdc", "usd coin"}},
	}

	e.coinPatterns = make([]CoinPattern, 0, len(coins))

	for _, coin := range coins {
		cp := CoinPattern{
			Symbol:   coin.symbol,
			Names:    coin.names,
			Patterns: make([]*regexp.Regexp, 0, len(coin.names)),
		}

		for _, name := range coin.names {
			// Create case-insensitive word boundary pattern
			pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(name) + `\b`)
			cp.Patterns = append(cp.Patterns, pattern)
		}

		e.coinPatterns = append(e.coinPatterns, cp)
	}
}

// ExtractMentionedCoins finds cryptocurrency mentions in text
func (e *Enricher) ExtractMentionedCoins(text string) []string {
	if text == "" {
		return []string{}
	}

	// Convert to lowercase for matching
	lowerText := strings.ToLower(text)

	found := make(map[string]bool)
	var result []string

	for _, cp := range e.coinPatterns {
		for _, pattern := range cp.Patterns {
			if pattern.MatchString(lowerText) {
				if !found[cp.Symbol] {
					found[cp.Symbol] = true
					result = append(result, cp.Symbol)
				}
				break // Found this coin, move to next
			}
		}
	}

	return result
}

// DetectCategory determines the category based on content and source
func (e *Enricher) DetectCategory(text string, sourceCategory string) string {
	if sourceCategory != "" {
		return sourceCategory
	}

	lowerText := strings.ToLower(text)

	// Category detection patterns
	categoryPatterns := map[string][]string{
		"defi": {"defi", "decentralized finance", "yield", "liquidity", "apy", "tvl", "lending", "borrowing"},
		"nft": {"nft", "non-fungible", "opensea", "blur", "digital art", "collectible"},
		"regulation": {"sec", "regulation", "law", "legal", "compliance", "ban", "sanction", "lawsuit"},
		"exchange": {"binance", "coinbase", "kraken", "exchange", "trading", "listing", "delisting"},
		"mining": {"mining", "miner", "hash rate", "proof of work", "pow"},
		"staking": {"staking", "stake", "proof of stake", "pos", "validator"},
		"layer2": {"layer 2", "l2", "rollup", "zk", "optimistic", "scaling"},
		"market": {"price", "market", "bullish", "bearish", "rally", "crash", "pump", "dump"},
		"technology": {"upgrade", "fork", "protocol", "development", "mainnet", "testnet"},
	}

	for category, patterns := range categoryPatterns {
		for _, pattern := range patterns {
			if strings.Contains(lowerText, pattern) {
				return category
			}
		}
	}

	return "general"
}

// IsBreaking determines if an article should be marked as breaking news
func (e *Enricher) IsBreaking(article *models.Article) bool {
	// Check if article is less than 2 hours old
	twoHoursAgo := time.Now().UTC().Add(-2 * time.Hour)
	if article.PubDate.After(twoHoursAgo) {
		return true
	}

	// Check for breaking keywords in title
	breakingKeywords := []string{
		"breaking",
		"just in",
		"urgent",
		"alert",
		"flash",
		"developing",
	}

	lowerTitle := strings.ToLower(article.Title)
	for _, keyword := range breakingKeywords {
		if strings.Contains(lowerTitle, keyword) {
			return true
		}
	}

	return false
}

// GenerateGUID creates a fallback GUID if one isn't provided
func (e *Enricher) GenerateGUID(article *models.Article) string {
	// Combine source ID, link, and title for uniqueness
	data := strings.Join([]string{
		string(rune(article.SourceID)),
		article.Link,
		article.Title,
	}, "|")

	hash := sha256.Sum256([]byte(data))
	return "gen-" + hex.EncodeToString(hash[:16])
}

// EnrichArticle applies all enrichments to an article
func (e *Enricher) EnrichArticle(article *models.Article, sourceCategory string) {
	// Extract mentioned coins from title and description
	text := article.Title + " " + article.Description
	coins := e.ExtractMentionedCoins(text)
	article.SetMentionedCoins(coins)

	// Detect if breaking
	article.IsBreaking = e.IsBreaking(article)

	// Ensure GUID is set
	if article.GUID == "" {
		article.GUID = e.GenerateGUID(article)
	}
}

// EnrichArticles applies enrichment to a batch of articles
func (e *Enricher) EnrichArticles(articles []models.Article, sourceCategory string) {
	for i := range articles {
		e.EnrichArticle(&articles[i], sourceCategory)
	}
}
