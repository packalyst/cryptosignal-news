'use client';

import { Suspense, useState, useEffect, useCallback } from 'react';
import { useSearchParams } from 'next/navigation';
import { Article } from '@/lib/types';
import { searchNews } from '@/lib/api';
import ArticleCard from '@/components/ArticleCard';
import LoadingSpinner from '@/components/LoadingSpinner';

function SearchContent() {
  const searchParams = useSearchParams();
  const query = searchParams.get('q') || '';

  const [searchQuery, setSearchQuery] = useState(query);
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasSearched, setHasSearched] = useState(false);

  const handleSearch = useCallback(async (q: string) => {
    if (!q.trim()) return;

    try {
      setLoading(true);
      setError(null);
      setHasSearched(true);

      const response = await searchNews(q);
      setArticles(response.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (query) {
      setSearchQuery(query);
      handleSearch(query);
    }
  }, [query, handleSearch]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    handleSearch(searchQuery);

    // Update URL without reload
    const url = new URL(window.location.href);
    url.searchParams.set('q', searchQuery);
    window.history.pushState({}, '', url);
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-white mb-2">Search News</h1>
        <p className="text-dark-400">
          Search through thousands of crypto news articles
        </p>
      </div>

      {/* Search Form */}
      <form onSubmit={handleSubmit} className="mb-8">
        <div className="flex gap-4">
          <div className="flex-grow relative">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search for Bitcoin, Ethereum, DeFi..."
              className="w-full bg-dark-800 border border-dark-700 text-white rounded-xl px-4 py-3 pl-12 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            />
            <svg
              className="absolute left-4 top-1/2 transform -translate-y-1/2 w-5 h-5 text-dark-500"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
          </div>
          <button
            type="submit"
            disabled={loading || !searchQuery.trim()}
            className="btn-primary px-8 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? <LoadingSpinner size="sm" /> : 'Search'}
          </button>
        </div>
      </form>

      {/* Loading */}
      {loading && (
        <div className="flex items-center justify-center py-20">
          <LoadingSpinner size="lg" />
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="text-center py-20">
          <div className="inline-flex items-center gap-2 px-4 py-2 bg-red-500/20 text-red-400 rounded-lg">
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
            </svg>
            {error}
          </div>
        </div>
      )}

      {/* Results */}
      {!loading && !error && hasSearched && (
        <>
          {articles.length === 0 ? (
            <div className="text-center py-20">
              <svg className="w-16 h-16 text-dark-600 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p className="text-dark-400 text-lg">No results found for &quot;{searchQuery}&quot;</p>
              <p className="text-dark-500 text-sm mt-2">
                Try different keywords or check your spelling
              </p>
            </div>
          ) : (
            <>
              <p className="text-dark-400 mb-6">
                Found {articles.length} result{articles.length !== 1 ? 's' : ''} for &quot;{searchQuery}&quot;
              </p>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {articles.map((article) => (
                  <ArticleCard key={article.id} article={article} />
                ))}
              </div>
            </>
          )}
        </>
      )}

      {/* Initial state */}
      {!loading && !error && !hasSearched && (
        <div className="text-center py-20">
          <svg className="w-16 h-16 text-dark-600 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <p className="text-dark-400 text-lg">Enter a search term to find articles</p>
          <div className="mt-6 flex flex-wrap gap-2 justify-center">
            {['Bitcoin', 'Ethereum', 'DeFi', 'NFT', 'Regulation'].map((term) => (
              <button
                key={term}
                onClick={() => {
                  setSearchQuery(term);
                  handleSearch(term);
                }}
                className="px-4 py-2 bg-dark-800 hover:bg-dark-700 border border-dark-700 rounded-lg text-sm transition-colors"
              >
                {term}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

export default function SearchPage() {
  return (
    <Suspense
      fallback={
        <div className="max-w-7xl mx-auto px-4 py-8">
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-white mb-2">Search News</h1>
            <p className="text-dark-400">
              Search through thousands of crypto news articles
            </p>
          </div>
          <div className="flex items-center justify-center py-20">
            <LoadingSpinner size="lg" />
          </div>
        </div>
      }
    >
      <SearchContent />
    </Suspense>
  );
}
