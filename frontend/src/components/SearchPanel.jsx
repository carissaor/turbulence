import { useState } from "react";
import axios from "axios";
import { DESTINATION_LABELS, DESTINATION_EMOJI } from "../constants";

const API = import.meta.env.VITE_API_URL;

const getCurrentMonth = () => {
  const now = new Date();

  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");

  return `${year}-${month}`;
};

export default function SearchPanel() {
  const [destination, setDestination] = useState("LHR");
  const [month, setMonth] = useState("");
  const [results, setResults] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const handleSearch = async (e) => {
    e?.preventDefault();
    if (!month || loading) {
      return;
    }

    setLoading(true);
    setError(null);
    setResults(null);

    try {
      const res = await axios.get(`${API}/api/search`, {
        params: {
          origin: "YVR",
          destination,
          month,
        },
      });

      setResults(res.data.results || []);
    } catch (err) {
      console.error("Flight search failed:", err);
      const message = err.response?.data?.trim?.();
      setError(message || "Failed to fetch prices. Try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="search-panel">
      <form className="search-controls" onSubmit={handleSearch}>
        <div className="search-field">
          <label className="search-label">From</label>
          <div className="search-static">YVR — Vancouver</div>
        </div>
        <div className="search-field">
          <label className="search-label" htmlFor="destination">
            To
          </label>

          <select
            id="destination"
            className="search-select"
            value={destination}
            onChange={(e) => setDestination(e.target.value)}
          >
            {Object.entries(DESTINATION_LABELS).map(([code, name]) => (
              <option key={code} value={code}>
                {DESTINATION_EMOJI[code]} {name} ({code})
              </option>
            ))}
          </select>
        </div>

        <div className="search-field">
          <label className="search-label" htmlFor="departure-month">
            Departure Month
          </label>

          <input
            id="departure-month"
            type="month"
            className="search-input"
            value={month}
            onChange={(e) => setMonth(e.target.value)}
            min={getCurrentMonth()}
          />
        </div>

        <button
          type="submit"
          className="search-btn"
          disabled={!month || loading}
        >
          {loading ? "Searching..." : "Search Flights"}
        </button>
      </form>

      {error && <div className="search-error">{error}</div>}

      {results && results.length === 0 && (
        <div className="search-empty">No flights found for this month.</div>
      )}

      {results && results.length > 0 && (
        <ul className="search-results">
          {results.map((r) => {
            const resultKey = [
              r.destination,
              r.depart_date,
              r.airline,
              r.price,
              r.transfers,
            ].join("-");

            return (
              <li key={resultKey} className="search-result-item">
                <div className="result-route">
                  <span className="result-flag">
                    {DESTINATION_EMOJI[r.destination]}
                  </span>

                  <span className="result-dest">YVR → {r.destination}</span>

                  <span className="result-airline">{r.airline}</span>
                </div>

                <div className="result-right">
                  <div className="result-price">
                    ${Number(r.price).toLocaleString()}
                  </div>

                  <div className="result-meta">
                    {r.depart_date?.slice(0, 10)} ·{" "}
                    {r.transfers === 0
                      ? "Direct"
                      : `${r.transfers} stop${r.transfers > 1 ? "s" : ""}`}
                  </div>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
