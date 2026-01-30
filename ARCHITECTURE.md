# CryptoSignal News - Architecture & Development Plan

## Overview
Production-ready crypto news aggregation + AI analysis platform built in Go with Next.js frontend.

## Tech Stack
- **Backend:** Go 1.22+ with Chi router
- **Frontend:** Next.js 14 + React + Tailwind
- **Database:** PostgreSQL 16
- **Cache:** Redis 7
- **AI:** Groq API (Llama 3.3 70B)
- **Deployment:** Docker Compose

## Project Structure
```
cryptosignal-news/
├── backend/                   # Go services
│   ├── cmd/
│   │   ├── api/              # Main API server
│   │   └── fetcher/          # RSS fetcher worker
│   ├── internal/
│   │   ├── config/           # Configuration
│   │   ├── database/         # PostgreSQL
│   │   ├── cache/            # Redis
│   │   ├── models/           # Data structures
│   │   ├── fetcher/          # RSS fetching
│   │   ├── parser/           # Feed parsing
│   │   ├── api/              # HTTP handlers
│   │   ├── service/          # Business logic
│   │   ├── ai/               # Groq integration
│   │   ├── auth/             # Authentication
│   │   └── realtime/         # WebSocket
│   ├── migrations/           # SQL migrations
│   ├── go.mod
│   └── Makefile
├── frontend/                  # Next.js app
│   ├── app/
│   ├── components/
│   ├── lib/
│   └── package.json
├── docker-compose.yml
└── README.md
```

## Database Schema

### sources
- id SERIAL PRIMARY KEY
- key VARCHAR(50) UNIQUE NOT NULL
- name VARCHAR(100) NOT NULL
- rss_url VARCHAR(500) NOT NULL
- website_url VARCHAR(500)
- category VARCHAR(50)
- language VARCHAR(10) DEFAULT 'en'
- is_enabled BOOLEAN DEFAULT true
- reliability_score DECIMAL(3,2) DEFAULT 1.0
- last_fetch_at TIMESTAMPTZ
- error_count INT DEFAULT 0
- created_at TIMESTAMPTZ DEFAULT NOW()

### articles
- id BIGSERIAL PRIMARY KEY
- source_id INT REFERENCES sources(id)
- guid VARCHAR(500) UNIQUE NOT NULL
- title TEXT NOT NULL
- link VARCHAR(1000) NOT NULL
- description TEXT
- pub_date TIMESTAMPTZ NOT NULL
- categories TEXT[]
- sentiment VARCHAR(20)
- sentiment_score DECIMAL(4,3)
- mentioned_coins TEXT[]
- is_breaking BOOLEAN DEFAULT false
- created_at TIMESTAMPTZ DEFAULT NOW()
- search_vector TSVECTOR (generated)

### users
- id UUID PRIMARY KEY
- email VARCHAR(255) UNIQUE NOT NULL
- password_hash VARCHAR(255)
- tier VARCHAR(20) DEFAULT 'free'
- api_key VARCHAR(64) UNIQUE
- api_calls_today INT DEFAULT 0
- created_at TIMESTAMPTZ DEFAULT NOW()

### alerts
- id SERIAL PRIMARY KEY
- user_id UUID REFERENCES users(id)
- name VARCHAR(100)
- type VARCHAR(50)
- config JSONB
- webhook_url VARCHAR(500)
- is_enabled BOOLEAN DEFAULT true
- created_at TIMESTAMPTZ DEFAULT NOW()

## API Endpoints

### Public
- GET /api/v1/news - Latest news
- GET /api/v1/news/breaking - Breaking news (last 2h)
- GET /api/v1/news/search?q= - Search
- GET /api/v1/sources - List sources
- GET /api/v1/categories - List categories
- GET /api/v1/health - Health check

### Authenticated
- GET /api/v1/news/:id - Article detail
- GET /api/v1/news/coin/:symbol - News by coin
- POST /api/v1/alerts - Create alert
- GET /api/v1/user/usage - Usage stats

### Premium
- GET /api/v1/ai/sentiment - AI sentiment
- GET /api/v1/ai/summary - Daily summary
- GET /api/v1/ai/signals - Trading signals
- WS /api/v1/ws/news - Real-time feed

## Rate Limits
- Free: 10/min, 500/day
- Pro: 60/min, 10,000/day
- Enterprise: 300/min, unlimited

## Environment Variables
```
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/cryptonews

# Redis
REDIS_URL=redis://localhost:6379

# AI
GROQ_API_KEY=your-key

# Auth
JWT_SECRET=your-secret

# Server
PORT=8080
ENV=development
```
