import { DESTINATION_EMOJI, DESTINATION_LABELS } from "../constants";

export default function RouteCard({ route, selected, onClick }) {
  const diff = route.latest_price - route.lowest_price;
  const isUp = diff > 0;

  return (
    <button
      className={`route-card ${selected ? "selected" : ""}`}
      onClick={onClick}
    >
      <div className="route-card-top">
        <span className="flag">
          {DESTINATION_EMOJI[route.destination] || "🌍"}
        </span>
        <span className="destination">
          {DESTINATION_LABELS[route.destination] || route.destination}
        </span>
      </div>
      <div className="route-card-price">
        ${route.latest_price.toLocaleString()}
      </div>
      <div className="route-card-meta">
        <span className="depart-date">Departs {route.depart_date}</span>
        {diff !== 0 && (
          <span className={`price-diff ${isUp ? "up" : "down"}`}>
            {isUp ? "▲" : "▼"} ${Math.abs(diff)} from lowest
          </span>
        )}
      </div>
    </button>
  );
}