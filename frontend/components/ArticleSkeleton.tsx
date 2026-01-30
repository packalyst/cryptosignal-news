export default function ArticleSkeleton() {
  return (
    <div className="card">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className="h-4 w-20 skeleton rounded"></div>
          <div className="h-4 w-16 skeleton rounded"></div>
        </div>
        <div className="h-4 w-12 skeleton rounded"></div>
      </div>

      <div className="h-6 w-full skeleton rounded mb-2"></div>
      <div className="h-6 w-3/4 skeleton rounded mb-4"></div>

      <div className="h-4 w-full skeleton rounded mb-1"></div>
      <div className="h-4 w-2/3 skeleton rounded mb-4"></div>

      <div className="flex items-center justify-between pt-3 border-t border-dark-700/50">
        <div className="flex gap-1.5">
          <div className="h-5 w-12 skeleton rounded"></div>
          <div className="h-5 w-12 skeleton rounded"></div>
        </div>
        <div className="h-5 w-16 skeleton rounded"></div>
      </div>
    </div>
  );
}
