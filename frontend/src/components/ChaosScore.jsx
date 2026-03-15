import { CHAOS_COLORS, CHAOS_EMOJI } from '../constants';

export default function ChaosScore({ score, level, label, insight, marketCount }) {
  const color = CHAOS_COLORS[level] ?? CHAOS_COLORS.UNKNOWN;
  const emoji = CHAOS_EMOJI[level]  ?? CHAOS_EMOJI.UNKNOWN;
  const isExtreme = level === 'EXTREME';

  return (
    <div className={`chaos-card ${isExtreme ? 'chaos-card--extreme' : ''}`}>
      <div className="chaos-left">
        <div className="chaos-label">Global Chaos Score</div>
        <div className={`chaos-score ${isExtreme ? 'chaos-score--extreme' : ''}`} style={{ color }}>
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
  );
}