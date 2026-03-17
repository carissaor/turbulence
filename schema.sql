-- Routes: the city pairs you're tracking prices for
CREATE TABLE IF NOT EXISTS routes (
    id          SERIAL PRIMARY KEY,
    origin      TEXT NOT NULL,
    destination TEXT NOT NULL,
    UNIQUE (origin, destination)
);

-- Prices: historical price snapshots from Travelpayouts
CREATE TABLE IF NOT EXISTS prices (
    id           SERIAL PRIMARY KEY,
    route_id     INT NOT NULL REFERENCES routes(id) ON DELETE CASCADE, 
    price        NUMERIC NOT NULL,
    currency     TEXT NOT NULL DEFAULT 'USD',
    depart_date  DATE,
    fetched_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Events: world-situation signals from Polymarket
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    question TEXT,
    probability FLOAT,
    volume FLOAT,
    end_date TIMESTAMP,
    fetched_at TIMESTAMP DEFAULT NOW()
);