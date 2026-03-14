-- Routes: the city pairs you're tracking prices for
CREATE TABLE IF NOT EXISTS routes (
    id          SERIAL PRIMARY KEY,
    origin      TEXT NOT NULL,
    destination TEXT NOT NULL,
    UNIQUE (origin, destination)
);

-- Flights: individual flight schedules from AviationStack
CREATE TABLE IF NOT EXISTS flights (
    id                   SERIAL PRIMARY KEY,
    flight_iata          TEXT NOT NULL,
    departure_airport    TEXT NOT NULL,
    arrival_airport      TEXT NOT NULL,
    scheduled_departure  TIMESTAMPTZ,
    UNIQUE (flight_iata, scheduled_departure)
);

-- Prices: historical price snapshots from Travelpayouts
CREATE TABLE IF NOT EXISTS prices (
    id           SERIAL PRIMARY KEY,
    route_id     INT NOT NULL REFERENCES routes(id),
    price        NUMERIC NOT NULL,
    currency     TEXT NOT NULL DEFAULT 'USD',
    depart_date  DATE,
    fetched_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Events: world-situation signals from Polymarket
-- probability is a 0-1 value (0 = won't happen, 1 = certain to happen)
CREATE TABLE IF NOT EXISTS events (
    id            SERIAL PRIMARY KEY,
    market_id     TEXT NOT NULL,       -- Polymarket's internal market ID
    question      TEXT NOT NULL,       -- e.g. "Will Russia invade another country by June?"
    probability   NUMERIC NOT NULL,    -- 0.0 to 1.0
    volume        NUMERIC,             -- total USD trading volume (higher = more reliable signal)
    fetched_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);