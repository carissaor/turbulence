export default function ChaosScore({ score, level, label, insight, marketCount }) {
  const color = {
    LOW:      '#0d9488',
    MODERATE: '#d97706',
    HIGH:     '#dc2626',
    EXTREME:  '#000000',
    UNKNOWN:  '#94a3b8',
  }[level] || '#94a3b8'

  const emoji = {
    LOW:      '🟢',
    MODERATE: '🟡',
    HIGH:     '🔴',
    EXTREME:  '🚨',
    UNKNOWN:  '⚪',
  }[level] || '⚪'

  return (
    <div className="chaos-card">
      <div className="chaos-left">
        <div className="chaos-label">🌍 Global Chaos Score</div>
        <div className="chaos-score" style={{ color }}>
          {score.toFixed(0)}
          <span className="chaos-score-max">/100</span>
        </div>
        <div className="chaos-level" style={{ color }}>
          {emoji} {label}
        </div>
        <div className="chaos-insight">{insight}</div>
      </div>
      <div className="chaos-right">
        <div className="chaos-bar-track">
          <div
            className="chaos-bar-fill"
            style={{
              width: `${score}%`,
              background: color,
            }}
          />
        </div>
        <div className="chaos-meta">based on {marketCount} Polymarket signals</div>
      </div>
    </div>
  )
}