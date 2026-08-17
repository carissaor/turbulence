import { useState, useEffect } from "react";
import axios from "axios";
import ChaosScore from "./components/ChaosScore";
import RouteCard from "./components/RouteCard";
import PriceChart from "./components/PriceChart";
import EventsPanel from "./components/EventsPanel";
import SearchPanel from "./components/SearchPanel";
import "./App.css";

const API = import.meta.env.VITE_API_URL;

export default function App() {
  const [routes, setRoutes] = useState([]);
  const [events, setEvents] = useState([]);
  const [selectedId, setSelectedId] = useState(null);
  const [loading, setLoading] = useState(true);
  const [chaos, setChaos] = useState(null);

  const selectedRoute =
    routes.find((r) => r.id === selectedId) || routes[0] || null;

  useEffect(() => {
    Promise.all([
      axios.get(`${API}/api/routes`),
      axios.get(`${API}/api/events`),
      axios.get(`${API}/api/chaos`),
    ])
      .then(([routesRes, eventsRes, chaosRes]) => {
        setRoutes(routesRes.data || []);
        setEvents(eventsRes.data || []);
        setChaos(chaosRes.data);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  return (
    <div className="app">
      <header className="header">
        <div className="header-inner">
          <div className="logo">✈️ Turbulence</div>
          <div className="header-sub">
            YVR departures · live price monitoring
          </div>
        </div>
      </header>

      <main className="main">
        {loading ? (
          <div className="loading">Connecting to API...</div>
        ) : (
          <>
            {chaos && (
              <ChaosScore {...chaos} marketCount={chaos.market_count} />
            )}

            <section className="section">
              <h2 className="section-title">Search Flights</h2>
              <SearchPanel />
            </section>

            <section className="section">
              <h2 className="section-title">Routes</h2>
              <div className="routes-grid">
                {routes.map((r) => (
                  <RouteCard
                    key={r.id}
                    route={r}
                    selected={selectedRoute?.id === r.id}
                    onClick={() => setSelectedId(r.id)}
                  />
                ))}
              </div>
            </section>

            <section className="section">
              <h2 className="section-title">Price History</h2>
              <PriceChart route={selectedRoute} />
            </section>

            <section className="section">
              <h2 className="section-title">🌍 World Event Signals</h2>
              <p className="section-desc">
                Prediction market probabilities from Polymarket — higher
                probability events may impact flight prices.
              </p>
              <EventsPanel events={events} />
            </section>
          </>
        )}
      </main>
    </div>
  );
}
