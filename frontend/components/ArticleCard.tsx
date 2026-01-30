'use client';

import { Article } from '@/lib/types';

interface ArticleCardProps {
  article: Article;
}

const sentimentColors = {
  bullish: 'bg-bullish/20 text-bullish border-bullish/30',
  bearish: 'bg-bearish/20 text-bearish border-bearish/30',
  neutral: 'bg-neutral/20 text-neutral border-neutral/30',
};

const coinColors: Record<string, string> = {
  BTC: 'bg-orange-500/20 text-orange-400 border-orange-500/30',
  ETH: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
  SOL: 'bg-gradient-to-r from-purple-500/20 to-green-500/20 text-green-400 border-green-500/30',
  XRP: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
  ADA: 'bg-blue-600/20 text-blue-300 border-blue-600/30',
  DOGE: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
  DOT: 'bg-pink-500/20 text-pink-400 border-pink-500/30',
  LINK: 'bg-blue-400/20 text-blue-300 border-blue-400/30',
  AVAX: 'bg-red-500/20 text-red-400 border-red-500/30',
  MATIC: 'bg-purple-600/20 text-purple-300 border-purple-600/30',
};

const defaultCoinColor = 'bg-dark-600/50 text-dark-300 border-dark-500/30';

export default function ArticleCard({ article }: ArticleCardProps) {
  return (
    <a
      href={article.link}
      target="_blank"
      rel="noopener noreferrer"
      className="block card hover:border-dark-600 hover:bg-dark-800/70 transition-all duration-200 group"
    >
      <div className="flex flex-col h-full">
        {/* Header with source and time */}
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-primary-400">{article.source}</span>
            {article.category && (
              <>
                <span className="text-dark-600">|</span>
                <span className="text-sm text-dark-400">{article.category}</span>
              </>
            )}
          </div>
          <span className="text-sm text-dark-500">{article.time_ago}</span>
        </div>

        {/* Breaking badge */}
        {article.is_breaking && (
          <div className="mb-2">
            <span className="inline-flex items-center gap-1.5 px-2 py-0.5 text-xs font-semibold bg-red-500/20 text-red-400 border border-red-500/30 rounded-full animate-pulse">
              <span className="w-1.5 h-1.5 bg-red-500 rounded-full"></span>
              BREAKING
            </span>
          </div>
        )}

        {/* Title */}
        <h3 className="text-lg font-semibold text-white group-hover:text-primary-400 transition-colors mb-2 line-clamp-2">
          {article.title}
        </h3>

        {/* Description */}
        {article.description && (
          <p className="text-dark-400 text-sm mb-4 line-clamp-2 flex-grow">
            {article.description}
          </p>
        )}

        {/* Footer with coins and sentiment */}
        <div className="flex items-center justify-between mt-auto pt-3 border-t border-dark-700/50">
          {/* Coin badges */}
          <div className="flex flex-wrap gap-1.5">
            {article.mentioned_coins?.slice(0, 5).map((coin) => (
              <span
                key={coin}
                className={`px-2 py-0.5 text-xs font-medium rounded border ${coinColors[coin] || defaultCoinColor}`}
              >
                {coin}
              </span>
            ))}
            {article.mentioned_coins && article.mentioned_coins.length > 5 && (
              <span className="px-2 py-0.5 text-xs text-dark-500">
                +{article.mentioned_coins.length - 5}
              </span>
            )}
          </div>

          {/* Sentiment badge */}
          {article.sentiment && (
            <span
              className={`px-2 py-0.5 text-xs font-medium rounded border capitalize ${sentimentColors[article.sentiment]}`}
            >
              {article.sentiment}
            </span>
          )}
        </div>
      </div>
    </a>
  );
}
