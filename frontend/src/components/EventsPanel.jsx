export default function EventsPanel({ events }) {
  if (!events.length) {
    return (
      <div className="events-empty">No world events detected right now.</div>
    );
  }

  const sortedEvents = [...events].sort(
    (a, b) => b.probability - a.probability,
  );

  return (
    <ul className="events-list">
      {sortedEvents.map((e, i) => {
        const pct = Math.round(e.probability * 100);
        const risk = pct >= 60 ? "high" : pct >= 30 ? "medium" : "low";

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
        );
      })}
    </ul>
  );
}