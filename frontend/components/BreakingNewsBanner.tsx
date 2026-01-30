'use client';

import { Article } from '@/lib/types';
import { useEffect, useState } from 'react';

interface BreakingNewsBannerProps {
  articles: Article[];
}

export default function BreakingNewsBanner({ articles }: BreakingNewsBannerProps) {
  const [currentIndex, setCurrentIndex] = useState(0);

  useEffect(() => {
    if (articles.length <= 1) return;

    const interval = setInterval(() => {
      setCurrentIndex((prev) => (prev + 1) % articles.length);
    }, 5000);

    return () => clearInterval(interval);
  }, [articles.length]);

  if (!articles.length) return null;

  const currentArticle = articles[currentIndex];

  return (
    <div className="bg-gradient-to-r from-red-900/30 via-red-800/20 to-red-900/30 border-y border-red-500/30">
      <div className="max-w-7xl mx-auto px-4 py-3">
        <div className="flex items-center gap-4">
          <span className="flex-shrink-0 inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-bold bg-red-500 text-white rounded animate-pulse">
            <span className="w-2 h-2 bg-white rounded-full"></span>
            BREAKING
          </span>
          <a
            href={currentArticle.link}
            target="_blank"
            rel="noopener noreferrer"
            className="flex-grow text-white hover:text-red-300 transition-colors truncate font-medium"
          >
            {currentArticle.title}
          </a>
          {articles.length > 1 && (
            <div className="flex-shrink-0 flex items-center gap-2">
              <button
                onClick={() => setCurrentIndex((prev) => (prev - 1 + articles.length) % articles.length)}
                className="p-1 text-dark-400 hover:text-white transition-colors"
                aria-label="Previous"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                </svg>
              </button>
              <span className="text-xs text-dark-400">
                {currentIndex + 1}/{articles.length}
              </span>
              <button
                onClick={() => setCurrentIndex((prev) => (prev + 1) % articles.length)}
                className="p-1 text-dark-400 hover:text-white transition-colors"
                aria-label="Next"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
