export interface Article {
  id: number;
  title: string;
  link: string;
  description: string;
  source: string;
  source_key: string;
  category: string;
  pub_date: string;
  time_ago: string;
  sentiment?: 'bullish' | 'bearish' | 'neutral';
  sentiment_score?: number;
  mentioned_coins?: string[];
  is_breaking: boolean;
}

export interface NewsResponse {
  data: Article[];
  pagination: {
    total: number;
    limit: number;
    offset: number;
    has_more: boolean;
  };
}

export interface NewsParams {
  limit?: number;
  offset?: number;
  category?: string;
  source?: string;
  language?: string;
  sort?: 'newest' | 'trending';
}

export interface Source {
  id: number;
  key: string;
  name: string;
  category: string;
  language: string;
  is_enabled: boolean;
}

export interface Category {
  key: string;
  name: string;
  count: number;
}

export interface SentimentResult {
  symbol: string;
  sentiment: 'bullish' | 'bearish' | 'neutral';
  score: number;
  article_count: number;
  bullish_count?: number;
  bearish_count?: number;
  neutral_count?: number;
  updated_at?: string;
  recent_headlines?: string[];
}

export interface MarketSummary {
  overall_sentiment: 'bullish' | 'bearish' | 'neutral';
  summary: string;
  key_developments: string[];
  mentioned_coins?: string[];
  notable_events?: string[];
  top_coins?: CoinSentiment[];
  generated_at: string;
  article_count?: number;
}

export interface CoinSentiment {
  symbol: string;
  name: string;
  sentiment: 'bullish' | 'bearish' | 'neutral';
  score: number;
  change_24h?: number;
}

export interface TradingSignal {
  coin: string;
  direction: 'bullish' | 'bearish' | 'neutral';
  strength: 'strong' | 'moderate' | 'weak';
  catalyst: string;
  source_title?: string;
}

export interface SignalsResponse {
  signals: TradingSignal[];
  market_mood: string;
  generated_at: string;
  article_count: number;
}
