interface SentimentBadgeProps {
  sentiment?: 'bullish' | 'bearish' | 'neutral' | string;
  score?: number;
  size?: 'sm' | 'md' | 'lg';
}

const sentimentConfig = {
  bullish: {
    bg: 'bg-bullish/20',
    text: 'text-bullish',
    border: 'border-bullish/30',
    icon: (
      <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M5.293 9.707a1 1 0 010-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 01-1.414 1.414L11 7.414V15a1 1 0 11-2 0V7.414L6.707 9.707a1 1 0 01-1.414 0z" clipRule="evenodd" />
      </svg>
    ),
  },
  bearish: {
    bg: 'bg-bearish/20',
    text: 'text-bearish',
    border: 'border-bearish/30',
    icon: (
      <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M14.707 10.293a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 111.414-1.414L9 12.586V5a1 1 0 012 0v7.586l2.293-2.293a1 1 0 011.414 0z" clipRule="evenodd" />
      </svg>
    ),
  },
  neutral: {
    bg: 'bg-neutral/20',
    text: 'text-neutral',
    border: 'border-neutral/30',
    icon: (
      <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
      </svg>
    ),
  },
};

const sizeClasses = {
  sm: 'px-2 py-0.5 text-xs',
  md: 'px-3 py-1 text-sm',
  lg: 'px-4 py-2 text-base',
};

export default function SentimentBadge({ sentiment, score, size = 'md' }: SentimentBadgeProps) {
  // Fallback to neutral if sentiment is undefined or invalid
  const validSentiment: 'bullish' | 'bearish' | 'neutral' =
    sentiment === 'bullish' || sentiment === 'bearish' || sentiment === 'neutral'
      ? sentiment
      : 'neutral';
  const config = sentimentConfig[validSentiment];

  return (
    <span
      className={`inline-flex items-center gap-1.5 font-medium rounded-full border ${config.bg} ${config.text} ${config.border} ${sizeClasses[size]} capitalize`}
    >
      {config.icon}
      {sentiment}
      {score !== undefined && (
        <span className="opacity-75">({Math.round(score * 100)}%)</span>
      )}
    </span>
  );
}
