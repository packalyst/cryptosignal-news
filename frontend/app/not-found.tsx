import Link from 'next/link';

export default function NotFound() {
  return (
    <div className="min-h-[60vh] flex items-center justify-center px-4">
      <div className="text-center">
        <h1 className="text-9xl font-bold text-dark-700">404</h1>
        <h2 className="text-2xl font-semibold text-white mt-4">Page Not Found</h2>
        <p className="text-dark-400 mt-2 max-w-md mx-auto">
          The page you are looking for does not exist or has been moved.
        </p>
        <div className="mt-8 flex gap-4 justify-center">
          <Link href="/" className="btn-primary px-6 py-2">
            Go Home
          </Link>
          <Link
            href="/news"
            className="px-6 py-2 border border-dark-600 hover:border-dark-500 rounded-lg transition-colors"
          >
            Browse News
          </Link>
        </div>
      </div>
    </div>
  );
}
