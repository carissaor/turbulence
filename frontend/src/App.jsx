import { useState, useEffect } from 'react'
import axios from 'axios'
import {
  LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid
} from 'recharts'
import './App.css'

const API = 'http://localhost:8080'

const DESTINATION_LABELS = {
  LHR: 'London',
  NRT: 'Tokyo',
  SYD: 'Sydney',
  CDG: 'Paris',
  JFK: 'New York',
  HKG: 'Hong Kong',
}

const DESTINATION_EMOJI = {
  LHR: '🇬🇧',
  NRT: '🇯🇵',
  SYD: '🇦🇺',
  CDG: '🇫🇷',
  JFK: '🇺🇸',
  HKG: '🇭🇰',
}

function RouteCard({ route, selected, onClick }) {
  const diff = route.latest_price - route.lowest_price
  const isUp = diff > 0

  return (
    <button
      className={`route-card ${selected ? 'selected' : ''}`}
      onClick={onClick}
    >
      <div className="route-card-top">
        <span className="flag">{DESTINATION_EMOJI[route.destination] || '🌍'}</span>
        <span className="destination">{DESTINATION_LABELS[route.destination] || route.destination}</span>
      </div>
      <div className="route-card-price">${route.latest_price.toLocaleString()}</div>
      <div className="route-card-meta">
        <span className="depart-date">Departs {route.depart_date}</span>
        {diff !== 0 && (
          <span className={`price-diff ${isUp ? 'up' : 'down'}`}>
            {isUp ? '▲' : '▼'} ${Math.abs(diff)} from lowest
          </span>
        )}
      </div>
    </button>
  )
}

function PriceChart({ route }) {
  const [history, setHistory] = useState([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!route) return
    axios.get(`${API}/api/prices?route=${route.origin}-${route.destination}`)
      .then(res => {
        const data = res.data.prices?.map(p => ({
          date: p.fetched_at?.slice(0, 10),
          price: p.price,
        })) || []
        setHistory(data)
        setLoading(false)
      })
      .catch(() => setLoading(false))
  }, [route])

  if (!route) return null
  if (loading) return <div className="chart-empty">Loading...</div>
  if (history.length === 0) return <div className="chart-empty">No price history yet — run the collector to build up data.</div>

  return (
    <div className="chart-wrapper">
      <h2 className="chart-title">
        {route.origin} → {route.destination}
        <span className="chart-subtitle">{DESTINATION_LABELS[route.destination]} price history</span>
      </h2>
      <ResponsiveContainer width="100%" height={240}>
        <LineChart data={history} margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(0,0,0,0.06)" />
          <XAxis dataKey="date" tick={{ fontSize: 11, fill: '#888' }} />
          <YAxis tick={{ fontSize: 11, fill: '#888' }} tickFormatter={v => `$${v}`} />
          <Tooltip
            contentStyle={{ background: '#fff', border: '1px solid #e2e8f0', borderRadius: 8, boxShadow: '0 4px 16px rgba(0,0,0,0.08)' }}
            formatter={v => [`$${v}`, 'Price']}
            labelStyle={{ color: '#64748b', fontSize: 12 }}
          />
          <Line
            type="monotone"
            dataKey="price"
            stroke="#0891b2"
            strokeWidth={2.5}
            dot={{ r: 4, fill: '#0891b2', strokeWidth: 0 }}
            activeDot={{ r: 6, fill: '#0e7490' }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}

function EventsPanel({ events }) {
  if (!events.length) return (
    <div className="events-empty">No world events detected right now.</div>
  )

  return (
    <ul className="events-list">
      {events.map((e, i) => {
        const pct = Math.round(e.probability * 100)
        const risk = pct >= 60 ? 'high' : pct >= 30 ? 'medium' : 'low'
        return (
          <li key={i} className={`event-item risk-${risk}`}>
            <div className="event-question">{e.question}</div>
            <div className="event-bar-row">
              <div className="event-bar">
                <div className="event-bar-fill" style={{ width: `${pct}%` }} />
              </div>
              <span className="event-pct">{pct}%</span>
            </div>
          </li>
        )
      })}
    </ul>
  )
}

export default function App() {
  const [routes, setRoutes] = useState([])
  const [events, setEvents] = useState([])
  const [selectedId, setSelectedId] = useState(null)
  const [loading, setLoading] = useState(true)

  const selectedRoute = routes.find(r => r.id === selectedId) || routes[0] || null

  useEffect(() => {
    Promise.all([
      axios.get(`${API}/api/routes`),
      axios.get(`${API}/api/events`),
    ]).then(([routesRes, eventsRes]) => {
      setRoutes(routesRes.data || [])
      setEvents(eventsRes.data || [])
      setLoading(false)
    }).catch(() => setLoading(false))
  }, [])

  return (
    <div className="app">
      <header className="header">
        <div className="header-inner">
          <div className="logo">✈️ flight-tracker</div>
          <div className="header-sub">YVR departures · live price monitoring</div>
        </div>
      </header>

      <main className="main">
        {loading ? (
          <div className="loading">Connecting to API...</div>
        ) : (
          <>
            <section className="section">
              <h2 className="section-title">Routes</h2>
              <div className="routes-grid">
                {routes.map(r => (
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
                Prediction market probabilities from Polymarket — higher probability events may impact flight prices.
              </p>
              <EventsPanel events={events} />
            </section>
          </>
        )}
      </main>
    </div>
  )
}