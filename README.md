# ✈️ flight-tracker

> Flight price monitoring and prediction 

---

## Overview

I love travelling!! But between global conflicts, pandemics, economic shifts, and geopolitical tensions, flight prices have never been more unpredictable. **flight-tracker** pulls real-time pricing data across multiple routes and world-event signals to build up a dataset for forecasting where prices are headed.

---

## What It Does Right Now

- Fetches the cheapest available fares for 6 routes out of YVR via the Travelpayouts API
- Pulls world-event signals from Polymarket — real money prediction markets for geopolitical events (conflicts, pandemics, oil prices, travel bans)
- Saves each price snapshot and event probability with a timestamp so history accumulates over time
- REST API serving price history and world-event data to a React frontend

---

## Routes Tracked

| Origin | Destination |
|--------|-------------|
| YVR | LHR — London |
| YVR | NRT — Tokyo |
| YVR | SYD — Sydney |
| YVR | CDG — Paris |
| YVR | JFK — New York |
| YVR | HKG — Hong Kong |

---

## World Event Signals

Uses the [Polymarket](https://polymarket.com) Gamma API (no API key required) to fetch prediction market probabilities for events that historically impact flight prices:

- Wars and invasions
- Pandemic declarations
- Travel bans and airspace closures
- Ceasefires and peace deals
- Crude oil price movements
- Financial crises

Each market returns a 0–1 probability representing what traders think is the likelihood of that event occurring. These get stored alongside price snapshots for correlation analysis.

---

## Roadmap

- [x] Route price fetching (Travelpayouts)
- [x] PostgreSQL storage with timestamped snapshots
- [x] World-event signals (Polymarket)
- [x] REST API (Go)
- [ ] React frontend dashboard
- [ ] Scheduled data collection (Railway)
- [ ] Deploy API to Railway, frontend to Vercel
- [ ] Price prediction model (Python)
- [ ] Price alert notifications

---

## Getting Started

### Prerequisites

- Go 1.21+
- Node.js 18+ (for frontend)
- PostgreSQL running locally
- [Travelpayouts API token](https://travelpayouts.com) (free)
- No API key needed for Polymarket

### Installation

```bash
git clone https://github.com/your-username/flight-tracker.git
cd flight-tracker
go mod tidy
```

### Database Setup

```bash
psql postgres -c "CREATE DATABASE flight_tracker;"
psql "postgres://YOUR_USER@localhost:5432/flight_tracker" -f schema.sql
```

### Configuration

```bash
cp .env.example .env
```

```env
DATABASE_URL=postgres://YOUR_USER@localhost:5432/flight_tracker?sslmode=disable
TRAVELPAYOUTS_TOKEN=your_token_here
ORIGIN=YVR
```

### Run the Collector

Fetches latest prices and world events, saves to DB:

```bash
go run ./cmd/collector
```

### Run the API Server

```bash
go run ./cmd/api
```

API runs on `http://localhost:8080`

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/routes` | All routes with latest and lowest price |
| GET | `/api/prices?route=YVR-LHR` | Price history for a specific route |
| GET | `/api/events` | Latest Polymarket world-event signals |

### Example Responses

**GET /api/routes**
```json
[
  {
    "id": 1,
    "origin": "YVR",
    "destination": "LHR",
    "lowest_price": 787,
    "latest_price": 787,
    "depart_date": "2026-04-27"
  }
]
```

**GET /api/events**
```json
[
  {
    "question": "US x Iran ceasefire by March 31?",
    "probability": 0.18,
    "volume": 42381,
    "fetched_at": "2026-03-13T20:42:51Z"
  }
]
```

---

## Project Structure

```
flight-tracker/
├── cmd/
│   ├── api/
│   │   └── main.go          # REST API server
│   └── collector/
│       └── main.go          # Price + event collector
├── frontend/                # React dashboard (Vite)
├── schema.sql               # Database table definitions
├── .env.example             # Environment variable template
├── go.mod
├── go.sum
└── README.md
```

---

## Database Schema

```sql
routes   -- city pairs being tracked (e.g. YVR → LHR)
prices   -- price snapshots per route with timestamps
events   -- Polymarket world-event probabilities with timestamps
```

---

## Data Sources

| Source | Purpose | Auth |
|--------|---------|------|
| [Travelpayouts](https://travelpayouts.com) | Live flight prices by route | API token |
| [Polymarket](https://polymarket.com) | World-event prediction markets | None |

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

## Disclaimer

Flight price predictions are based on historical data and world-event signals. They are not financial advice. Always verify prices directly with airlines or booking platforms before making travel decisions.