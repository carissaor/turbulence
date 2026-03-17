import { useState, useEffect } from "react";
import axios from "axios";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  ReferenceLine,
} from "recharts";
import { DESTINATION_LABELS } from "../constants";

const API = import.meta.env.VITE_API_URL;

export default function PriceChart({ route }) {
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(false);
  const [mode, setMode] = useState("depart"); // "depart" | "dailyLowest"

  useEffect(() => {
    if (!route) return;

    // setLoading(true);

    axios
      .get(
        `${API}/api/prices?route=${route.origin}-${route.destination}&mode=${mode}`
      )
      .then((res) => {
        const data =
          res.data.prices
            ?.filter((p) => p.date && Number(p.price) > 0)
            .map((p) => ({
              date: p.date,
              price: Number(p.price),
            })) || [];

        setHistory(data);
        setLoading(false);
      })
      .catch(() => {
        setHistory([]);
        setLoading(false);
      });
  }, [route, mode]);

  if (!route) return null;
  if (loading) return <div className="chart-empty">Loading...</div>;
  if (history.length === 0) {
    return (
      <div className="chart-empty">
        No price data yet — search a route above to start building history.
      </div>
    );
  }

  const prices = history.map((h) => h.price);
  const minPrice = Math.min(...prices);
  const avgPrice = Math.round(
    prices.reduce((a, b) => a + b, 0) / prices.length
  );

  return (
    <div className="chart-wrapper">
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          gap: 12,
          flexWrap: "wrap",
          marginBottom: 12,
        }}
      >
        <h2 className="chart-title" style={{ margin: 0 }}>
          {route.origin} → {route.destination}
          <span className="chart-subtitle">
            {DESTINATION_LABELS[route.destination]} ·{" "}
            {mode === "depart"
              ? "price by departure date"
              : "lowest fetched price per day"}
          </span>
        </h2>

        <div
          style={{
            display: "inline-flex",
            background: "#f1f5f9",
            borderRadius: 10,
            padding: 4,
            gap: 4,
          }}
        >
          <button
            type="button"
            onClick={() => setMode("depart")}
            style={{
              border: "none",
              borderRadius: 8,
              padding: "8px 12px",
              cursor: "pointer",
              background: mode === "depart" ? "#0891b2" : "transparent",
              color: mode === "depart" ? "#fff" : "#334155",
              fontWeight: 600,
            }}
          >
            Departure price
          </button>

          <button
            type="button"
            onClick={() => setMode("dailyLowest")}
            style={{
              border: "none",
              borderRadius: 8,
              padding: "8px 12px",
              cursor: "pointer",
              background: mode === "dailyLowest" ? "#0891b2" : "transparent",
              color: mode === "dailyLowest" ? "#fff" : "#334155",
              fontWeight: 600,
            }}
          >
            Daily lowest
          </button>
        </div>
      </div>

      <div className="chart-stats">
        <div className="chart-stat">
          <span className="chart-stat-label">Lowest</span>
          <span className="chart-stat-value lowest">
            ${minPrice.toLocaleString()}
          </span>
        </div>
        <div className="chart-stat">
          <span className="chart-stat-label">Average</span>
          <span className="chart-stat-value">${avgPrice.toLocaleString()}</span>
        </div>
      </div>

      <ResponsiveContainer width="100%" height={240}>
        <LineChart
          data={history}
          margin={{ top: 8, right: 16, left: 0, bottom: 0 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(0,0,0,0.06)" />
          <XAxis
            dataKey="date"
            tick={{ fontSize: 10, fill: "#888" }}
            tickFormatter={(d) => d?.slice(5)}
          />
          <YAxis
            tick={{ fontSize: 11, fill: "#888" }}
            tickFormatter={(v) => `$${v}`}
            domain={["auto", "auto"]}
          />
          <Tooltip
            contentStyle={{
              background: "#fff",
              border: "1px solid #e2e8f0",
              borderRadius: 8,
              boxShadow: "0 4px 16px rgba(0,0,0,0.08)",
            }}
            formatter={(v) => [`$${Number(v).toLocaleString()}`, "Price"]}
            labelStyle={{ color: "#64748b", fontSize: 12 }}
          />
          <ReferenceLine
            y={avgPrice}
            stroke="#94a3b8"
            strokeDasharray="4 4"
            label={{
              value: "avg",
              position: "right",
              fontSize: 10,
              fill: "#94a3b8",
            }}
          />
          <Line
            type="monotone"
            dataKey="price"
            stroke="#0891b2"
            strokeWidth={2.5}
            dot={{ r: 3, fill: "#0891b2", strokeWidth: 0 }}
            activeDot={{ r: 6, fill: "#0e7490" }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}