import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, type Finding, type Stats } from '../api'
import StatusBadge from '../components/StatusBadge'

const STATUS_FILTERS: Array<{ value: string; label: string }> = [
  { value: '', label: 'All' },
  { value: 'new', label: 'New' },
  { value: 'acknowledged', label: 'Acknowledged' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'false_positive', label: 'False positive' },
]

export default function Dashboard() {
  const [findings, setFindings] = useState<Finding[]>([])
  const [stats, setStats] = useState<Stats | null>(null)
  const [status, setStatus] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.stats().then(setStats)
  }, [])

  useEffect(() => {
    setLoading(true)
    api
      .findings(status ? { status } : {})
      .then(setFindings)
      .finally(() => setLoading(false))
  }, [status])

  return (
    <div>
      <header className="page-header">
        <h1>Findings</h1>
      </header>

      {stats && (
        <div className="stat-row">
          <StatTile label="New" value={stats.findingCounts.new ?? 0} tone="new" />
          <StatTile label="Acknowledged" value={stats.findingCounts.acknowledged ?? 0} tone="acknowledged" />
          <StatTile label="Resolved" value={stats.findingCounts.resolved ?? 0} tone="resolved" />
          <StatTile label="Repos tracked" value={stats.repoCount} tone="neutral" />
        </div>
      )}

      <div className="filter-row">
        {STATUS_FILTERS.map((f) => (
          <button
            key={f.value}
            className={`chip ${status === f.value ? 'chip-active' : ''}`}
            onClick={() => setStatus(f.value)}
          >
            {f.label}
          </button>
        ))}
      </div>

      {loading ? (
        <p className="muted">Loading…</p>
      ) : findings.length === 0 ? (
        <div className="empty-state">
          <p>No findings here yet.</p>
          <p className="muted">
            Connect a GitHub org or add a repo on the <Link to="/repos">Repos</Link> page to start scanning.
          </p>
        </div>
      ) : (
        <table className="data-table">
          <thead>
            <tr>
              <th>Rule</th>
              <th>Repo</th>
              <th>Location</th>
              <th>Status</th>
              <th>First seen</th>
            </tr>
          </thead>
          <tbody>
            {findings.map((f) => (
              <tr key={f.id}>
                <td>
                  <Link to={`/findings/${f.id}`} className="rule-link">
                    {f.ruleId}
                  </Link>
                </td>
                <td>{f.repoName}</td>
                <td className="mono">
                  {f.filePath}:{f.startLine}
                </td>
                <td>
                  <StatusBadge status={f.status} />
                </td>
                <td className="muted">{new Date(f.firstSeenAt).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

function StatTile({ label, value, tone }: { label: string; value: number; tone: string }) {
  return (
    <div className={`stat-tile tone-${tone}`}>
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
    </div>
  )
}
