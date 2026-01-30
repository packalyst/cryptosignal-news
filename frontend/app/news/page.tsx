'use client';

import { useState, useEffect, useCallback } from 'react';
import { Article, Category, Source } from '@/lib/types';
import { getNews, getCategories, getSources } from '@/lib/api';
import ArticleCard from '@/components/ArticleCard';
import FilterSelect from '@/components/FilterSelect';
import LoadingSpinner from '@/components/LoadingSpinner';

const LIMIT = 20;

export default function NewsPage() {
  const [articles, setArticles] = useState<Article[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [sources, setSources] = useState<Source[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(true);
  const [offset, setOffset] = useState(0);

  // Filters
  const [selectedCategory, setSelectedCategory] = useState('');
  const [selectedSource, setSelectedSource] = useState('');

  // Load filter options
  useEffect(() => {
    const loadFilters = async () => {
      try {
        const [categoriesData, sourcesData] = await Promise.all([
          getCategories(),
          getSources(),
        ]);
        setCategories(categoriesData);
        setSources(sourcesData.filter(s => s.is_enabled));
      } catch (err) {
        console.error('Failed to load filters:', err);
      }
    };
    loadFilters();
  }, []);

  // Load articles
  const loadArticles = useCallback(async (reset = false) => {
    try {
      if (reset) {
        setLoading(true);
        setOffset(0);
      } else {
        setLoadingMore(true);
      }
      setError(null);

      const currentOffset = reset ? 0 : offset;
      const response = await getNews({
        limit: LIMIT,
        offset: currentOffset,
        category: selectedCategory || undefined,
        source: selectedSource || undefined,
      });

      if (reset) {
        setArticles(response.data);
      } else {
        setArticles((prev) => [...prev, ...response.data]);
      }

      setHasMore(response.pagination.has_more);
      setOffset(currentOffset + LIMIT);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load articles');
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  }, [offset, selectedCategory, selectedSource]);

  // Initial load and filter changes
  useEffect(() => {
    loadArticles(true);
  }, [selectedCategory, selectedSource]);

  // Handle filter changes
  const handleCategoryChange = (value: string) => {
    setSelectedCategory(value);
  };

  const handleSourceChange = (value: string) => {
    setSelectedSource(value);
  };

  // Load more handler
  const handleLoadMore = () => {
    if (!loadingMore && hasMore) {
      loadArticles(false);
    }
  };

  // Category options
  const categoryOptions = [
    { value: '', label: 'All Categories' },
    ...categories.map((cat) => ({
      value: cat.key,
      label: `${cat.name} (${cat.count})`,
    })),
  ];

  // Source options
  const sourceOptions = [
    { value: '', label: 'All Sources' },
    ...sources.map((src) => ({
      value: src.key,
      label: src.name,
    })),
  ];

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-white mb-2">Crypto News</h1>
        <p className="text-dark-400">
          Latest news from 150+ sources, updated in real-time
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-4 mb-8 p-4 bg-dark-900/50 rounded-xl border border-dark-800">
        <FilterSelect
          label="Category"
          options={categoryOptions}
          value={selectedCategory}
          onChange={handleCategoryChange}
        />
        <FilterSelect
          label="Source"
          options={sourceOptions}
          value={selectedSource}
          onChange={handleSourceChange}
        />
        {(selectedCategory || selectedSource) && (
          <button
            onClick={() => {
              setSelectedCategory('');
              setSelectedSource('');
            }}
            className="self-end px-4 py-2 text-sm text-dark-400 hover:text-white transition-colors"
          >
            Clear Filters
          </button>
        )}
      </div>

      {/* Loading state */}
      {loading && (
        <div className="flex items-center justify-center py-20">
          <LoadingSpinner size="lg" />
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="text-center py-20">
          <div className="inline-flex items-center gap-2 px-4 py-2 bg-red-500/20 text-red-400 rounded-lg mb-4">
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
            </svg>
            {error}
          </div>
          <button
            onClick={() => loadArticles(true)}
            className="btn-primary"
          >
            Try Again
          </button>
        </div>
      )}

      {/* Articles grid */}
      {!loading && !error && (
        <>
          {articles.length === 0 ? (
            <div className="text-center py-20">
              <p className="text-dark-400 text-lg">No articles found</p>
              <p className="text-dark-500 text-sm mt-2">
                Try adjusting your filters
              </p>
            </div>
          ) : (
            <>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {articles.map((article) => (
                  <ArticleCard key={article.id} article={article} />
                ))}
              </div>

              {/* Load more button */}
              {hasMore && (
                <div className="mt-8 text-center">
                  <button
                    onClick={handleLoadMore}
                    disabled={loadingMore}
                    className="btn-primary px-8 py-3 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {loadingMore ? (
                      <span className="flex items-center gap-2">
                        <LoadingSpinner size="sm" />
                        Loading...
                      </span>
                    ) : (
                      'Load More'
                    )}
                  </button>
                </div>
              )}

              {/* End of list */}
              {!hasMore && articles.length > 0 && (
                <p className="mt-8 text-center text-dark-500">
                  You have reached the end
                </p>
              )}
            </>
          )}
        </>
      )}
    </div>
  );
}
