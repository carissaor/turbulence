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
} from "recharts";
import { DESTINATION_LABELS } from "../constants";

const API = import.meta.env.VITE_API_URL;

export default function PriceChart({ route }) {
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!route) return;
    axios
      .get(`${API}/api/prices?route=${route.origin}-${route.destination}`)
      .then((res) => {
        const data =
          res.data.prices?.map((p) => ({
            date: p.fetched_at?.slice(0, 10),
            price: p.price,
          })) || [];
        setHistory(data);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [route]);

  if (!route) return null;
  if (loading) return <div className="chart-empty">Loading...</div>;
  if (history.length === 0)
    return (
      <div className="chart-empty">
        No price history yet — run the collector to build up data.
      </div>
    );

  return (
    <div className="chart-wrapper">
      <h2 className="chart-title">
        {route.origin} → {route.destination}
        <span className="chart-subtitle">
          {DESTINATION_LABELS[route.destination]} price history
        </span>
      </h2>
      <ResponsiveContainer width="100%" height={240}>
        <LineChart
          data={history}
          margin={{ top: 8, right: 16, left: 0, bottom: 0 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(0,0,0,0.06)" />
          <XAxis dataKey="date" tick={{ fontSize: 11, fill: "#888" }} />
          <YAxis
            tick={{ fontSize: 11, fill: "#888" }}
            tickFormatter={(v) => `$${v}`}
          />
          <Tooltip
            contentStyle={{
              background: "#fff",
              border: "1px solid #e2e8f0",
              borderRadius: 8,
              boxShadow: "0 4px 16px rgba(0,0,0,0.08)",
            }}
            formatter={(v) => [`$${v}`, "Price"]}
            labelStyle={{ color: "#64748b", fontSize: 12 }}
          />
          <Line
            type="monotone"
            dataKey="price"
            stroke="#0891b2"
            strokeWidth={2.5}
            dot={{ r: 4, fill: "#0891b2", strokeWidth: 0 }}
            activeDot={{ r: 6, fill: "#0e7490" }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}