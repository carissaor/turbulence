# ✈️ flight-tracker

> Flight price monitoring and prediction — because the world situation shouldn't catch your wallet off guard.

---

## Overview

We all love travelling. But between global conflicts, pandemics, economic shifts, and geopolitical tensions, flight prices have never been more unpredictable. **flight-tracker** pulls real-time pricing data across multiple routes and world-event signals to build up a dataset for forecasting where prices are headed.

---

## What It Does Right Now

- Connects to a local PostgreSQL database
- Fetches the cheapest available fares for 6 routes out of YVR via the Travelpayouts API
- Pulls world-event signals from Polymarket — real money prediction markets for geopolitical events (conflicts, pandemics, oil prices, travel bans)
- Saves each price snapshot and event probability with a timestamp so history accumulates over time

Each run adds new rows to the database. Run it daily and you build up the price history and world-event correlation data needed for prediction.

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

Each market returns a 0–1 probability representing what traders think is the likelihood of that event occurring. These signals get stored alongside price snapshots for future correlation analysis.

---

## Roadmap

- [x] Route price fetching (Travelpayouts)
- [x] PostgreSQL storage with timestamped snapshots
- [x] World-event signals (Polymarket)
- [ ] Scheduled data collection (cron / cloud deploy)
- [ ] Price prediction model
- [ ] Route comparison and trend visualization
- [ ] Price alert notifications

---

## Getting Started

### Prerequisites

- Go 1.21+
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

### Run

```bash
go run main.go
```

### Example Output

```
🐘 Connected to PostgreSQL!

🔍 YVR → LHR
  💰 $787 | departs 2026-04-27 | direct

🔍 YVR → NRT
  💰 $478 | departs 2026-09-28 | direct

🔍 YVR → SYD
  💰 $1152 | departs 2026-04-20 | direct

🔍 YVR → CDG
  💰 $624 | departs 2026-04-12 | direct

🔍 YVR → JFK
  💰 $327 | departs 2026-04-30 | direct

🔍 YVR → HKG
  💰 $541 | departs 2026-05-03 | direct

🌐 Fetching world event signals from Polymarket...
  Found 4 relevant markets
  🌍 1% | US x Iran ceasefire by March 15?
  🌍 18% | US x Iran ceasefire by March 31?
  🌍 62% | Will crude oil hit $80 by end of Q2?
  🌍 9% | Will WHO declare a health emergency in 2026?

✅ Done!
```

---

## Project Structure

```
flight-tracker/
├── main.go         # Fetch prices and world events, save to DB
├── schema.sql      # Database table definitions
├── .env.example    # Environment variable template
├── go.mod
└── go.sum
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