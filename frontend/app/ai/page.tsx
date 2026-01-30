'use client';

import { useState, useEffect } from 'react';
import { MarketSummary, SentimentResult, SignalsResponse } from '@/lib/types';
import { getMarketSummary, getTradingSignals, getSentiment } from '@/lib/api';
import SentimentBadge from '@/components/SentimentBadge';
import LoadingSpinner from '@/components/LoadingSpinner';

const TOP_COINS = ['BTC', 'ETH', 'SOL', 'XRP', 'ADA', 'DOGE'];

export default function AIPage() {
  const [marketSummary, setMarketSummary] = useState<MarketSummary | null>(null);
  const [signalsData, setSignalsData] = useState<SignalsResponse | null>(null);
  const [coinSentiments, setCoinSentiments] = useState<SentimentResult[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'overview' | 'signals' | 'sentiment'>('overview');

  useEffect(() => {
    const loadData = async () => {
      try {
        setLoading(true);
        setError(null);

        // Load all data in parallel
        const [summaryData, signalsData, ...sentimentData] = await Promise.all([
          getMarketSummary().catch(() => null),
          getTradingSignals().catch(() => null),
          ...TOP_COINS.map(coin => getSentiment(coin).catch(() => null)),
        ]);

        setMarketSummary(summaryData);
        setSignalsData(signalsData);
        setCoinSentiments(sentimentData.filter((s): s is SentimentResult => s !== null));
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load AI insights');
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, []);

  if (loading) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-20 flex items-center justify-center">
        <div className="text-center">
          <LoadingSpinner size="lg" />
          <p className="mt-4 text-dark-400">Analyzing market data...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-20 text-center">
        <div className="inline-flex items-center gap-2 px-4 py-2 bg-red-500/20 text-red-400 rounded-lg mb-4">
          <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
          </svg>
          {error}
        </div>
        <button
          onClick={() => window.location.reload()}
          className="btn-primary"
        >
          Try Again
        </button>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-white mb-2">AI Insights</h1>
        <p className="text-dark-400">
          Market sentiment and trading signals powered by AI analysis
        </p>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 mb-8 border-b border-dark-800">
        {[
          { id: 'overview', label: 'Market Overview' },
          { id: 'signals', label: 'Trading Signals' },
          { id: 'sentiment', label: 'Coin Sentiment' },
        ].map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id as typeof activeTab)}
            className={`px-4 py-3 text-sm font-medium transition-colors border-b-2 -mb-px ${
              activeTab === tab.id
                ? 'text-primary-400 border-primary-500'
                : 'text-dark-400 border-transparent hover:text-white'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Overview Tab */}
      {activeTab === 'overview' && (
        <div className="space-y-8">
          {/* Market Summary Card */}
          {marketSummary ? (
            <div className="card">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-xl font-semibold text-white">Market Summary</h2>
                <SentimentBadge sentiment={marketSummary.overall_sentiment} size="lg" />
              </div>

              <p className="text-dark-300 text-lg mb-6">{marketSummary.summary}</p>

              {/* Key Developments */}
              {marketSummary.key_developments && marketSummary.key_developments.length > 0 && (
                <div className="mb-6">
                  <h3 className="text-sm font-medium text-dark-400 uppercase tracking-wider mb-3">
                    Key Developments
                  </h3>
                  <ul className="space-y-2">
                    {marketSummary.key_developments.map((dev, index) => (
                      <li key={index} className="flex items-start gap-2 text-dark-300">
                        <span className="text-primary-500 mt-1">-</span>
                        {dev}
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Top Coins Grid */}
              {marketSummary.top_coins && marketSummary.top_coins.length > 0 && (
                <div>
                  <h3 className="text-sm font-medium text-dark-400 uppercase tracking-wider mb-3">
                    Top Coins Sentiment
                  </h3>
                  <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
                    {marketSummary.top_coins.map((coin) => (
                      <div
                        key={coin.symbol}
                        className="p-4 bg-dark-900/50 rounded-lg border border-dark-700/50 text-center"
                      >
                        <span className="text-lg font-bold text-white block mb-1">
                          {coin.symbol}
                        </span>
                        <SentimentBadge sentiment={coin.sentiment} size="sm" />
                      </div>
                    ))}
                  </div>
                </div>
              )}

              <p className="mt-6 text-xs text-dark-500">
                Last updated: {new Date(marketSummary.generated_at).toLocaleString()}
              </p>
            </div>
          ) : (
            <div className="card text-center py-12">
              <p className="text-dark-400">Market summary not available</p>
            </div>
          )}
        </div>
      )}

      {/* Signals Tab */}
      {activeTab === 'signals' && (
        <div>
          {signalsData?.signals && signalsData.signals.length > 0 ? (
            <div className="space-y-6">
              {/* Market Mood Header */}
              {signalsData.market_mood && (
                <div className="card">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="text-lg font-semibold text-white">Market Mood</h3>
                      <p className="text-dark-300">{signalsData.market_mood}</p>
                    </div>
                    <div className="text-right text-sm text-dark-500">
                      Based on {signalsData.article_count} articles
                    </div>
                  </div>
                </div>
              )}

              {/* Signals Grid */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {signalsData.signals.map((signal, index) => (
                  <div key={index} className="card">
                    <div className="flex items-center justify-between mb-3">
                      <h3 className="text-xl font-bold text-white">{signal.coin}</h3>
                      <span className={`px-3 py-1 rounded-full text-sm font-medium ${
                        signal.direction === 'bullish'
                          ? 'bg-bullish/20 text-bullish'
                          : signal.direction === 'bearish'
                          ? 'bg-bearish/20 text-bearish'
                          : 'bg-neutral/20 text-neutral'
                      }`}>
                        {signal.direction}
                      </span>
                    </div>

                    <div className="mb-3">
                      <span className="text-xs text-dark-500 uppercase tracking-wider">Strength</span>
                      <div className="flex items-center gap-2 mt-1">
                        <div className="flex gap-1">
                          {[1, 2, 3].map((level) => (
                            <div
                              key={level}
                              className={`w-3 h-3 rounded-full ${
                                (signal.strength === 'strong' && level <= 3) ||
                                (signal.strength === 'moderate' && level <= 2) ||
                                (signal.strength === 'weak' && level <= 1)
                                  ? signal.direction === 'bullish'
                                    ? 'bg-bullish'
                                    : signal.direction === 'bearish'
                                    ? 'bg-bearish'
                                    : 'bg-neutral'
                                  : 'bg-dark-700'
                              }`}
                            />
                          ))}
                        </div>
                        <span className="text-sm text-dark-400 capitalize">{signal.strength}</span>
                      </div>
                    </div>

                    <div>
                      <span className="text-xs text-dark-500 uppercase tracking-wider">Catalyst</span>
                      <p className="text-dark-300 text-sm mt-1">{signal.catalyst}</p>
                    </div>

                    {signal.source_title && (
                      <p className="text-xs text-dark-500 mt-3 truncate">
                        Source: {signal.source_title}
                      </p>
                    )}
                  </div>
                ))}
              </div>

              <p className="text-xs text-dark-500 text-center">
                Last updated: {new Date(signalsData.generated_at).toLocaleString()}
              </p>
            </div>
          ) : (
            <div className="card text-center py-12">
              <svg className="w-12 h-12 text-dark-600 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              <p className="text-dark-400 text-lg">No trading signals available</p>
              <p className="text-dark-500 text-sm mt-2">
                Signals are generated based on market conditions and news analysis
              </p>
            </div>
          )}
        </div>
      )}

      {/* Sentiment Tab */}
      {activeTab === 'sentiment' && (
        <div>
          {coinSentiments.length > 0 ? (
            <div className="space-y-6">
              {/* Coin Sentiment Cards */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {coinSentiments.map((sentiment) => (
                  <div key={sentiment.symbol} className="card">
                    <div className="flex items-center justify-between mb-4">
                      <h3 className="text-xl font-bold text-white">{sentiment.symbol}</h3>
                      <SentimentBadge sentiment={sentiment.sentiment} score={sentiment.score} />
                    </div>

                    {/* Article count */}
                    <p className="text-sm text-dark-400 mb-4">
                      Based on {sentiment.article_count} articles
                    </p>

                    {/* Recent Headlines */}
                    {sentiment.recent_headlines && sentiment.recent_headlines.length > 0 && (
                      <div>
                        <h4 className="text-xs font-medium text-dark-500 uppercase tracking-wider mb-2">
                          Recent Headlines
                        </h4>
                        <ul className="space-y-1">
                          {sentiment.recent_headlines.slice(0, 3).map((headline, index) => (
                            <li key={index} className="text-sm text-dark-300 truncate">
                              - {headline}
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="card text-center py-12">
              <svg className="w-12 h-12 text-dark-600 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
              <p className="text-dark-400 text-lg">No sentiment data available</p>
              <p className="text-dark-500 text-sm mt-2">
                Sentiment analysis requires recent news articles
              </p>
            </div>
          )}
        </div>
      )}

      {/* Disclaimer */}
      <div className="mt-12 p-4 bg-dark-900/30 rounded-lg border border-dark-800">
        <p className="text-xs text-dark-500 text-center">
          <strong className="text-dark-400">Disclaimer:</strong> AI-generated insights are for informational purposes only and should not be considered financial advice. Always do your own research before making investment decisions.
        </p>
      </div>
    </div>
  );
}
