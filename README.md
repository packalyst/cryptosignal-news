# CryptoSignal News

Real-time cryptocurrency news aggregator with AI-powered sentiment analysis, translation, and trading signals.

## Features

- **Multi-source RSS Aggregation** - Fetches from 100+ crypto news sources worldwide
- **Auto Translation** - Translates non-English articles using Groq LLM
- **AI Sentiment Analysis** - Analyzes market sentiment per coin
- **Trading Signals** - Generates trading signals from news
- **Market Summaries** - Daily AI-generated market overviews
- **Rate Limiting** - Tier-based API rate limiting (anonymous, free, pro, enterprise)
- **JWT Authentication** - Secure user authentication with API key support

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Frontend  │────▶│   Backend   │────▶│  PostgreSQL │
│   Next.js   │     │   Go/Chi    │     │             │
└─────────────┘     └──────┬──────┘     └─────────────┘
                          │
                    ┌─────┴─────┐
                    │   Redis   │
                    │  (Cache)  │
                    └───────────┘
                          │
┌─────────────┐     ┌─────┴─────┐
│   Fetcher   │────▶│  Groq AI  │
│   Worker    │     │   (LLM)   │
└─────────────┘     └───────────┘
```

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Groq API key (free at https://console.groq.com)

### Setup

1. Clone the repository:
```bash
git clone git@github.com:packalyst/cryptosignal-news.git
cd cryptosignal-news
```

2. Copy environment file:
```bash
cp .env.example .env
```

3. Edit `.env` with your settings:
```bash
POSTGRES_USER=cryptonews
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=cryptonews
GROQ_API_KEY=your_groq_api_key
```

4. Start services:
```bash
# Backend only (API, fetcher, postgres, redis)
docker compose up -d

# With frontend
docker compose --profile frontend up -d
```

5. Access the application:
- Frontend: http://localhost:3000
- API: http://localhost:8080
- Status: http://localhost:8080/api/v1/status

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GROQ_API_KEY` | Groq API key for AI features | - |
| `TRANSLATION_TARGET_LANGUAGE` | Target language for articles | `en` |
| `TRANSLATION_INTERVAL` | How often to check for pending translations | `30s` |
| `TRANSLATION_BATCH_SIZE` | Articles to translate per batch | `5` |
| `MODEL_TRANSLATION` | LLM model for translation | `llama-3.1-8b-instant` |
| `MODEL_SENTIMENT` | LLM model for sentiment analysis | `llama-3.3-70b-versatile` |
| `MODEL_SUMMARY` | LLM model for summaries | `llama-3.3-70b-versatile` |
| `FETCH_INTERVAL` | RSS fetch interval | `3m` |
| `RATE_LIMIT_ENABLED` | Enable rate limiting | `true` |

## API Endpoints

### News
- `GET /api/v1/news` - List articles (paginated)
- `GET /api/v1/news/{id}` - Get single article
- `GET /api/v1/news/breaking` - Breaking news
- `GET /api/v1/news/search?q=` - Search articles
- `GET /api/v1/news/coin/{symbol}` - News by coin (BTC, ETH, etc.)

### AI
- `GET /api/v1/ai/sentiment?coin=BTC` - Sentiment analysis for a coin
- `GET /api/v1/ai/summary` - Daily market summary
- `GET /api/v1/ai/signals` - Trading signals from news

### System
- `GET /api/v1/status` - System status and translation progress
- `GET /api/v1/sources` - List news sources
- `GET /api/v1/categories` - List categories

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/refresh` - Refresh token
- `GET /api/v1/user/me` - Current user (authenticated)
- `POST /api/v1/user/api-keys` - Create API key (authenticated)

## Development

### Backend (Go)
```bash
cd backend
go build ./...
go run ./cmd/api        # Run API server
go run ./cmd/fetcher    # Run fetcher worker
```

### Frontend (Next.js)
```bash
cd frontend
npm install
npm run dev
```

### Database Migrations
Migrations run automatically on container start. Located in `backend/migrations/`.

## Project Structure

```
├── backend/
│   ├── cmd/
│   │   ├── api/          # API server entrypoint
│   │   └── fetcher/      # Fetcher worker entrypoint
│   ├── internal/
│   │   ├── ai/           # Groq AI services (sentiment, translation, signals)
│   │   ├── api/          # HTTP handlers and router
│   │   ├── auth/         # JWT and API key authentication
│   │   ├── cache/        # Redis cache
│   │   ├── config/       # Configuration
│   │   ├── database/     # PostgreSQL connection
│   │   ├── fetcher/      # RSS fetcher and translator worker
│   │   ├── middleware/   # HTTP middleware
│   │   ├── models/       # Data models
│   │   ├── repository/   # Database queries
│   │   ├── service/      # Business logic
│   │   └── sources/      # RSS feed definitions
│   └── migrations/       # SQL migrations
├── frontend/
│   ├── app/              # Next.js app router
│   ├── components/       # React components
│   └── lib/              # Utilities and API client
├── docker-compose.yml
└── .env.example
```

## License

MIT
