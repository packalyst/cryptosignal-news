import { CoinSentiment } from '@/lib/types';
import SentimentBadge from './SentimentBadge';

interface CoinSentimentCardProps {
  coin: CoinSentiment;
}

const coinIcons: Record<string, string> = {
  BTC: 'Bitcoin',
  ETH: 'Ethereum',
  SOL: 'Solana',
  XRP: 'Ripple',
  ADA: 'Cardano',
  DOGE: 'Dogecoin',
  DOT: 'Polkadot',
  LINK: 'Chainlink',
  AVAX: 'Avalanche',
  MATIC: 'Polygon',
};

export default function CoinSentimentCard({ coin }: CoinSentimentCardProps) {
  const displayName = coin.name || coinIcons[coin.symbol] || coin.symbol;

  return (
    <div className="card hover:border-dark-600 transition-colors">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-xl font-bold text-white">{coin.symbol}</h3>
          <p className="text-sm text-dark-400">{displayName}</p>
        </div>
        <SentimentBadge sentiment={coin.sentiment} />
      </div>

      {/* Sentiment score bar */}
      <div className="mb-3">
        <div className="flex justify-between text-sm mb-1">
          <span className="text-dark-400">Sentiment Score</span>
          <span className={
            coin.sentiment === 'bullish' ? 'text-bullish' :
            coin.sentiment === 'bearish' ? 'text-bearish' : 'text-neutral'
          }>
            {Math.round(coin.score * 100)}%
          </span>
        </div>
        <div className="w-full h-2 bg-dark-700 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all duration-500 ${
              coin.sentiment === 'bullish' ? 'bg-bullish' :
              coin.sentiment === 'bearish' ? 'bg-bearish' : 'bg-neutral'
            }`}
            style={{ width: `${Math.round(coin.score * 100)}%` }}
          />
        </div>
      </div>

      {/* 24h change if available */}
      {coin.change_24h !== undefined && (
        <div className="flex items-center justify-between pt-3 border-t border-dark-700/50">
          <span className="text-sm text-dark-400">24h Change</span>
          <span className={`text-sm font-medium ${
            coin.change_24h >= 0 ? 'text-bullish' : 'text-bearish'
          }`}>
            {coin.change_24h >= 0 ? '+' : ''}{coin.change_24h.toFixed(2)}%
          </span>
        </div>
      )}
    </div>
  );
}
