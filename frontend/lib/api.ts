import { Article, NewsResponse, NewsParams, Source, Category, SentimentResult, MarketSummary, SignalsResponse } from './types';

// Server-side uses internal Docker network, client-side uses same host as browser
function getApiUrl(): string {
  if (typeof window === 'undefined') {
    // Server-side: use internal Docker URL
    return process.env.API_URL || 'http://api:8080';
  }
  // Client-side: use same host as the page, port 8080
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }
  // Auto-detect: use same hostname as the browser
  const host = window.location.hostname;
  return `http://${host}:8080`;
}

async function fetchAPI<T>(endpoint: string): Promise<T> {
  const apiUrl = getApiUrl();
  const response = await fetch(`${apiUrl}${endpoint}`);
  if (!response.ok) {
    throw new Error(`API Error: ${response.statusText}`);
  }
  return response.json();
}

export async function getNews(params?: NewsParams): Promise<NewsResponse> {
  const searchParams = new URLSearchParams();
  if (params?.limit) searchParams.set('limit', params.limit.toString());
  if (params?.offset) searchParams.set('offset', params.offset.toString());
  if (params?.category) searchParams.set('category', params.category);
  if (params?.source) searchParams.set('source', params.source);
  const query = searchParams.toString();
  return fetchAPI<NewsResponse>(`/api/v1/news${query ? `?${query}` : ''}`);
}

export async function getBreakingNews(): Promise<Article[]> {
  const response = await fetchAPI<{ data: Article[] }>('/api/v1/news/breaking');
  return response.data;
}

export async function searchNews(query: string): Promise<NewsResponse> {
  return fetchAPI<NewsResponse>(`/api/v1/news/search?q=${encodeURIComponent(query)}`);
}

export async function getArticle(id: string): Promise<Article> {
  return fetchAPI<Article>(`/api/v1/news/${id}`);
}

export async function getSources(): Promise<Source[]> {
  const response = await fetchAPI<{ data: Source[] }>('/api/v1/sources');
  return response.data;
}

export async function getCategories(): Promise<Category[]> {
  const response = await fetchAPI<{ data: Category[] }>('/api/v1/categories');
  return response.data;
}

export async function getSentiment(coin: string): Promise<SentimentResult> {
  const response = await fetchAPI<{ data: SentimentResult }>(`/api/v1/ai/sentiment?coin=${coin}`);
  return response.data;
}

export async function getMarketSummary(): Promise<MarketSummary> {
  const response = await fetchAPI<{ data: MarketSummary }>('/api/v1/ai/summary');
  return response.data;
}

export async function getTradingSignals(): Promise<SignalsResponse> {
  const response = await fetchAPI<{ data: SignalsResponse }>('/api/v1/ai/signals');
  return response.data;
}

export const fetcher = <T>(url: string): Promise<T> => fetchAPI<T>(url);
