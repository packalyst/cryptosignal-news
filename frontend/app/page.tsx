'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { Article, MarketSummary } from '@/lib/types';
import { getBreakingNews, getNews, getMarketSummary } from '@/lib/api';
import ArticleCard from '@/components/ArticleCard';
import BreakingNewsBanner from '@/components/BreakingNewsBanner';
import SentimentBadge from '@/components/SentimentBadge';
import LoadingSpinner from '@/components/LoadingSpinner';

export default function HomePage() {
  const [breakingNews, setBreakingNews] = useState<Article[]>([]);
  const [latestNews, setLatestNews] = useState<Article[]>([]);
  const [marketSummary, setMarketSummary] = useState<MarketSummary | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadData = async () => {
      try {
        const [breaking, latest, summary] = await Promise.all([
          getBreakingNews().catch(() => []),
          getNews({ limit: 6 }).catch(() => ({ data: [] })),
          getMarketSummary().catch(() => null),
        ]);

        setBreakingNews(breaking);
        setLatestNews(latest.data);
        setMarketSummary(summary);
      } catch (err) {
        console.error('Failed to load homepage data:', err);
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, []);

  return (
    <div>
      {/* Breaking News Banner */}
      {breakingNews.length > 0 && <BreakingNewsBanner articles={breakingNews} />}

      {/* Hero Section */}
      <section className="py-16 md:py-24">
        <div className="max-w-4xl mx-auto px-4 text-center">
          <h1 className="text-4xl md:text-5xl lg:text-6xl font-bold leading-tight">
            Real-time Crypto News
            <span className="block mt-2 bg-gradient-to-r from-primary-400 to-primary-600 bg-clip-text text-transparent">
              Powered by AI
            </span>
          </h1>
          <p className="mt-6 text-lg md:text-xl text-dark-300 max-w-2xl mx-auto">
            Stay ahead with AI-powered sentiment analysis and breaking news aggregated from 150+ trusted sources.
          </p>
          <div className="mt-10 flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/news" className="btn-primary px-8 py-3 text-lg">
              Browse News
            </Link>
            <Link href="/ai" className="px-8 py-3 text-lg bg-dark-700 hover:bg-dark-600 rounded-lg transition-colors">
              AI Insights
            </Link>
          </div>
        </div>
      </section>

      {/* Market Sentiment Overview */}
      {marketSummary && (
        <section className="py-12 border-y border-dark-800 bg-dark-900/30">
          <div className="max-w-7xl mx-auto px-4">
            <div className="flex flex-col md:flex-row items-center justify-between gap-6">
              <div className="flex items-center gap-4">
                <div>
                  <span className="text-sm text-dark-400 uppercase tracking-wider">Market Sentiment</span>
                  <div className="mt-1">
                    <SentimentBadge sentiment={marketSummary.overall_sentiment} size="lg" />
                  </div>
                </div>
              </div>
              <p className="text-dark-300 text-center md:text-left max-w-xl">
                {marketSummary.summary}
              </p>
              <Link
                href="/ai"
                className="flex-shrink-0 text-primary-400 hover:text-primary-300 transition-colors flex items-center gap-2"
              >
                View Details
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </Link>
            </div>

            {/* Top Coins Quick View */}
            {marketSummary.top_coins && marketSummary.top_coins.length > 0 && (
              <div className="mt-8 flex flex-wrap justify-center gap-3">
                {marketSummary.top_coins.slice(0, 6).map((coin) => (
                  <div
                    key={coin.symbol}
                    className={`px-4 py-2 rounded-lg border ${
                      coin.sentiment === 'bullish'
                        ? 'bg-bullish/10 border-bullish/30 text-bullish'
                        : coin.sentiment === 'bearish'
                        ? 'bg-bearish/10 border-bearish/30 text-bearish'
                        : 'bg-neutral/10 border-neutral/30 text-neutral'
                    }`}
                  >
                    <span className="font-bold">{coin.symbol}</span>
                    <span className="ml-2 text-sm opacity-75 capitalize">{coin.sentiment}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </section>
      )}

      {/* Latest News Section */}
      <section className="py-16">
        <div className="max-w-7xl mx-auto px-4">
          <div className="flex items-center justify-between mb-8">
            <div>
              <h2 className="text-2xl md:text-3xl font-bold text-white">Latest News</h2>
              <p className="text-dark-400 mt-1">Real-time updates from trusted sources</p>
            </div>
            <Link
              href="/news"
              className="text-primary-400 hover:text-primary-300 transition-colors flex items-center gap-2"
            >
              View All
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </Link>
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-12">
              <LoadingSpinner size="lg" />
            </div>
          ) : latestNews.length > 0 ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {latestNews.map((article) => (
                <ArticleCard key={article.id} article={article} />
              ))}
            </div>
          ) : (
            <div className="text-center py-12">
              <p className="text-dark-400">No news available</p>
            </div>
          )}
        </div>
      </section>

      {/* Categories Section */}
      <section className="py-12 bg-dark-900/50">
        <div className="max-w-7xl mx-auto px-4">
          <h2 className="text-2xl font-bold text-white text-center mb-8">Browse by Category</h2>
          <div className="flex flex-wrap gap-3 justify-center">
            {[
              { name: 'Bitcoin', icon: 'BTC' },
              { name: 'Ethereum', icon: 'ETH' },
              { name: 'DeFi', icon: null },
              { name: 'NFT', icon: null },
              { name: 'Trading', icon: null },
              { name: 'Regulation', icon: null },
              { name: 'Research', icon: null },
              { name: 'Markets', icon: null },
            ].map((cat) => (
              <Link
                key={cat.name}
                href={`/news?category=${cat.name.toLowerCase()}`}
                className="group px-6 py-3 bg-dark-800 hover:bg-dark-700 border border-dark-700 hover:border-dark-600 rounded-xl transition-all duration-200"
              >
                <span className="text-dark-300 group-hover:text-white transition-colors">
                  {cat.name}
                </span>
              </Link>
            ))}
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-16">
        <div className="max-w-7xl mx-auto px-4">
          <h2 className="text-2xl md:text-3xl font-bold text-white text-center mb-12">
            Why CryptoSignal News?
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {/* Feature 1 */}
            <div className="card text-center">
              <div className="w-12 h-12 bg-primary-500/20 rounded-xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-6 h-6 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-white mb-2">Real-time Updates</h3>
              <p className="text-dark-400">
                News aggregated from 150+ sources, updated every minute
              </p>
            </div>

            {/* Feature 2 */}
            <div className="card text-center">
              <div className="w-12 h-12 bg-bullish/20 rounded-xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-6 h-6 text-bullish" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-white mb-2">AI Sentiment Analysis</h3>
              <p className="text-dark-400">
                Understand market mood with AI-powered sentiment scores
              </p>
            </div>

            {/* Feature 3 */}
            <div className="card text-center">
              <div className="w-12 h-12 bg-purple-500/20 rounded-xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-6 h-6 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-white mb-2">Trading Signals</h3>
              <p className="text-dark-400">
                Get AI-generated buy/sell signals based on news analysis
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 bg-gradient-to-b from-dark-900/50 to-transparent">
        <div className="max-w-4xl mx-auto px-4 text-center">
          <h2 className="text-3xl font-bold text-white mb-4">
            Ready to Trade Smarter?
          </h2>
          <p className="text-dark-300 mb-8 max-w-xl mx-auto">
            Get instant access to AI-powered market insights and stay ahead of the curve.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/ai" className="btn-primary px-8 py-3 text-lg">
              Explore AI Features
            </Link>
            <Link
              href="/search"
              className="px-8 py-3 text-lg border border-dark-600 hover:border-dark-500 rounded-lg transition-colors"
            >
              Search News
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}
