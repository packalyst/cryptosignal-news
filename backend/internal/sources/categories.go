package sources

import "strings"

// Category represents a news category with metadata for filtering and display
type Category struct {
	Slug        string   // URL-friendly identifier
	Name        string   // Display name
	Description string   // Brief description of the category
	Keywords    []string // Keywords for auto-categorization of articles
	Color       string   // Hex color code for UI display
}

// categories holds all available news categories
var categories = []Category{
	{
		Slug:        "general",
		Name:        "General",
		Description: "General cryptocurrency news and updates",
		Keywords:    []string{"crypto", "cryptocurrency", "blockchain", "digital asset", "web3"},
		Color:       "#6B7280",
	},
	{
		Slug:        "bitcoin",
		Name:        "Bitcoin",
		Description: "Bitcoin-specific news, analysis, and developments",
		Keywords:    []string{"bitcoin", "btc", "satoshi", "lightning network", "halving", "mining btc"},
		Color:       "#F7931A",
	},
	{
		Slug:        "ethereum",
		Name:        "Ethereum",
		Description: "Ethereum ecosystem news and updates",
		Keywords:    []string{"ethereum", "eth", "vitalik", "eip", "erc", "solidity", "dapp", "smart contract"},
		Color:       "#627EEA",
	},
	{
		Slug:        "defi",
		Name:        "DeFi",
		Description: "Decentralized finance protocols and news",
		Keywords:    []string{"defi", "yield", "liquidity", "amm", "dex", "lending", "borrowing", "staking", "farming", "tvl", "aave", "uniswap", "compound"},
		Color:       "#8B5CF6",
	},
	{
		Slug:        "nft",
		Name:        "NFT",
		Description: "Non-fungible tokens, digital art, and collectibles",
		Keywords:    []string{"nft", "opensea", "collectible", "digital art", "pfp", "metaverse", "gaming nft", "blur"},
		Color:       "#EC4899",
	},
	{
		Slug:        "trading",
		Name:        "Trading",
		Description: "Trading analysis, market movements, and price action",
		Keywords:    []string{"trading", "price", "analysis", "technical", "chart", "bullish", "bearish", "pump", "dump", "rally", "correction", "support", "resistance"},
		Color:       "#10B981",
	},
	{
		Slug:        "research",
		Name:        "Research",
		Description: "In-depth research, reports, and analysis",
		Keywords:    []string{"research", "report", "analysis", "study", "data", "metrics", "on-chain", "fundamental"},
		Color:       "#3B82F6",
	},
	{
		Slug:        "institutional",
		Name:        "Institutional",
		Description: "Institutional adoption, ETFs, and corporate news",
		Keywords:    []string{"institutional", "etf", "grayscale", "blackrock", "fidelity", "corporate", "treasury", "adoption", "hedge fund", "investment"},
		Color:       "#1E40AF",
	},
	{
		Slug:        "mining",
		Name:        "Mining",
		Description: "Cryptocurrency mining news and hash rate updates",
		Keywords:    []string{"mining", "miner", "hash rate", "asic", "gpu", "proof of work", "pow", "difficulty"},
		Color:       "#78716C",
	},
	{
		Slug:        "layer2",
		Name:        "Layer 2",
		Description: "Layer 2 scaling solutions and rollups",
		Keywords:    []string{"layer 2", "l2", "rollup", "optimistic", "zk", "arbitrum", "optimism", "polygon", "base", "scaling", "zksync"},
		Color:       "#06B6D4",
	},
	{
		Slug:        "altcoins",
		Name:        "Altcoins",
		Description: "Alternative cryptocurrency news and updates",
		Keywords:    []string{"altcoin", "solana", "cardano", "polkadot", "avalanche", "cosmos", "near", "token", "memecoin", "shitcoin"},
		Color:       "#F59E0B",
	},
	{
		Slug:        "regulation",
		Name:        "Regulation",
		Description: "Regulatory news, policy, and legal developments",
		Keywords:    []string{"regulation", "sec", "cftc", "legal", "lawsuit", "compliance", "ban", "policy", "government", "law", "tax"},
		Color:       "#DC2626",
	},
	{
		Slug:        "security",
		Name:        "Security",
		Description: "Security incidents, hacks, and vulnerabilities",
		Keywords:    []string{"hack", "exploit", "vulnerability", "security", "breach", "scam", "rug pull", "phishing", "audit"},
		Color:       "#EF4444",
	},
	{
		Slug:        "gaming",
		Name:        "Gaming",
		Description: "Blockchain gaming and play-to-earn news",
		Keywords:    []string{"gaming", "play to earn", "p2e", "game", "gamefi", "metaverse", "virtual world", "axie"},
		Color:       "#A855F7",
	},
}

// GetAllCategories returns all available categories
func GetAllCategories() []Category {
	result := make([]Category, len(categories))
	copy(result, categories)
	return result
}

// GetCategoryBySlug returns a category by its slug, or nil if not found
func GetCategoryBySlug(slug string) *Category {
	for i := range categories {
		if categories[i].Slug == slug {
			cat := categories[i]
			return &cat
		}
	}
	return nil
}

// GetCategorySlugs returns all category slugs
func GetCategorySlugs() []string {
	result := make([]string, len(categories))
	for i, cat := range categories {
		result[i] = cat.Slug
	}
	return result
}

// MatchCategory attempts to match text to a category based on keywords
// Returns the best matching category slug or "general" if no match
func MatchCategory(text string) string {
	// Convert to lowercase for matching
	lowerText := strings.ToLower(text)

	bestMatch := "general"
	bestScore := 0

	for _, cat := range categories {
		score := 0
		for _, keyword := range cat.Keywords {
			if strings.Contains(lowerText, keyword) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestMatch = cat.Slug
		}
	}

	return bestMatch
}

// CategoryExists checks if a category with the given slug exists
func CategoryExists(slug string) bool {
	for _, cat := range categories {
		if cat.Slug == slug {
			return true
		}
	}
	return false
}
