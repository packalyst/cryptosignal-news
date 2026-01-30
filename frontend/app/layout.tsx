import type { Metadata } from 'next';
import Link from 'next/link';
import './globals.css';

export const metadata: Metadata = {
  title: 'CryptoSignal News - Real-time Crypto News & AI Analysis',
  description: 'Stay ahead with real-time crypto news aggregation and AI-powered sentiment analysis from 150+ trusted sources.',
  keywords: 'crypto, news, bitcoin, ethereum, trading, sentiment, AI, analysis',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark">
      <body className="bg-dark-950 text-white min-h-screen flex flex-col">
        <header className="sticky top-0 z-50 bg-dark-900/95 backdrop-blur border-b border-dark-700">
          <div className="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
            <Link href="/" className="text-xl font-bold flex items-center gap-2">
              <svg className="w-8 h-8 text-primary-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
              </svg>
              <span>CryptoSignal<span className="text-primary-500">News</span></span>
            </Link>
            <nav className="hidden md:flex items-center gap-6">
              <Link href="/" className="text-dark-300 hover:text-white transition-colors">Home</Link>
              <Link href="/news" className="text-dark-300 hover:text-white transition-colors">News</Link>
              <Link href="/ai" className="text-dark-300 hover:text-white transition-colors">AI Insights</Link>
              <Link href="/search" className="text-dark-300 hover:text-white transition-colors flex items-center gap-1">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                Search
              </Link>
            </nav>
            {/* Mobile menu button */}
            <MobileMenuButton />
          </div>
        </header>
        <main className="flex-grow">{children}</main>
        <footer className="border-t border-dark-800 py-8 mt-auto">
          <div className="max-w-7xl mx-auto px-4">
            <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
              <div className="md:col-span-2">
                <Link href="/" className="text-xl font-bold flex items-center gap-2 mb-4">
                  <svg className="w-6 h-6 text-primary-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
                  </svg>
                  <span>CryptoSignal<span className="text-primary-500">News</span></span>
                </Link>
                <p className="text-dark-400 text-sm max-w-md">
                  Real-time crypto news aggregation with AI-powered sentiment analysis. Stay ahead of the market with breaking news from 150+ trusted sources.
                </p>
              </div>
              <div>
                <h4 className="text-white font-semibold mb-4">Quick Links</h4>
                <ul className="space-y-2 text-sm">
                  <li><Link href="/news" className="text-dark-400 hover:text-white transition-colors">Latest News</Link></li>
                  <li><Link href="/ai" className="text-dark-400 hover:text-white transition-colors">AI Insights</Link></li>
                  <li><Link href="/search" className="text-dark-400 hover:text-white transition-colors">Search</Link></li>
                </ul>
              </div>
              <div>
                <h4 className="text-white font-semibold mb-4">Categories</h4>
                <ul className="space-y-2 text-sm">
                  <li><Link href="/news?category=bitcoin" className="text-dark-400 hover:text-white transition-colors">Bitcoin</Link></li>
                  <li><Link href="/news?category=ethereum" className="text-dark-400 hover:text-white transition-colors">Ethereum</Link></li>
                  <li><Link href="/news?category=defi" className="text-dark-400 hover:text-white transition-colors">DeFi</Link></li>
                  <li><Link href="/news?category=trading" className="text-dark-400 hover:text-white transition-colors">Trading</Link></li>
                </ul>
              </div>
            </div>
            <div className="mt-8 pt-8 border-t border-dark-800 text-center text-dark-500 text-sm">
              <p>&copy; {new Date().getFullYear()} CryptoSignal News. All rights reserved.</p>
            </div>
          </div>
        </footer>
      </body>
    </html>
  );
}

function MobileMenuButton() {
  return (
    <div className="md:hidden">
      <details className="relative">
        <summary className="list-none cursor-pointer p-2 -mr-2 text-dark-300 hover:text-white">
          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </summary>
        <nav className="absolute right-0 mt-2 w-48 bg-dark-800 border border-dark-700 rounded-lg shadow-xl py-2 z-50">
          <Link href="/" className="block px-4 py-2 text-dark-300 hover:text-white hover:bg-dark-700 transition-colors">Home</Link>
          <Link href="/news" className="block px-4 py-2 text-dark-300 hover:text-white hover:bg-dark-700 transition-colors">News</Link>
          <Link href="/ai" className="block px-4 py-2 text-dark-300 hover:text-white hover:bg-dark-700 transition-colors">AI Insights</Link>
          <Link href="/search" className="block px-4 py-2 text-dark-300 hover:text-white hover:bg-dark-700 transition-colors">Search</Link>
        </nav>
      </details>
    </div>
  );
}
