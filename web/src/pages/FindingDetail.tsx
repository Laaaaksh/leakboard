import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, type Finding } from '../api'
import StatusBadge from '../components/StatusBadge'

export default function FindingDetail() {
  const { id } = useParams()
  const [finding, setFinding] = useState<Finding | null>(null)
  const [revealed, setRevealed] = useState(false)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (!id) return
    api.finding(Number(id), revealed).then(setFinding)
  }, [id, revealed])

  if (!finding) return <p className="muted">Loading…</p>

  async function setStatus(status: Finding['status']) {
    setBusy(true)
    try {
      await api.updateFindingStatus(finding!.id, status)
      const updated = await api.finding(finding!.id, revealed)
      setFinding(updated)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div>
      <Link to="/" className="back-link">
        ← Back to findings
      </Link>

      <header className="page-header">
        <h1>{finding.ruleId}</h1>
        <StatusBadge status={finding.status} />
      </header>

      <dl className="detail-grid">
        <dt>Repository</dt>
        <dd>{finding.repoName}</dd>
        <dt>File</dt>
        <dd className="mono">
          {finding.filePath}:{finding.startLine}
        </dd>
        <dt>Commit</dt>
        <dd className="mono">{finding.commitSha}</dd>
        <dt>Author</dt>
        <dd>
          {finding.commitAuthor} {finding.commitEmail && <span className="muted">&lt;{finding.commitEmail}&gt;</span>}
        </dd>
        <dt>First seen</dt>
        <dd>{new Date(finding.firstSeenAt).toLocaleString()}</dd>
        <dt>Last seen</dt>
        <dd>{new Date(finding.lastSeenAt).toLocaleString()}</dd>
      </dl>

      <div className="secret-panel">
        <div className="secret-panel-header">
          <span>Secret value</span>
          <button className="link-button" onClick={() => setRevealed((r) => !r)}>
            {revealed ? 'Hide' : 'Reveal'}
          </button>
        </div>
        <code className="secret-value">{finding.secret}</code>
      </div>

      <div className="action-row">
        <button disabled={busy || finding.status === 'acknowledged'} onClick={() => setStatus('acknowledged')}>
          Acknowledge
        </button>
        <button disabled={busy || finding.status === 'resolved'} onClick={() => setStatus('resolved')}>
          Mark resolved
        </button>
        <button
          disabled={busy || finding.status === 'false_positive'}
          className="danger-outline"
          onClick={() => setStatus('false_positive')}
        >
          False positive
        </button>
      </div>
      <p className="muted small">
        Marking a finding as a false positive adds its fingerprint to the shared allowlist, so it won't resurface.
        For a whole class of known-safe matches (a test fixture path, a rule you don't care about), add a rule to the{' '}
        <Link to="/allowlist">Allowlist</Link> instead.
      </p>
    </div>
  )
}
