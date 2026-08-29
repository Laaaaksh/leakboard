import { useEffect, useState, type FormEvent } from 'react'
import { api, ApiError, type AllowlistEntry } from '../api'

export default function Allowlist() {
  const [entries, setEntries] = useState<AllowlistEntry[]>([])
  const [ruleId, setRuleId] = useState('')
  const [pathPattern, setPathPattern] = useState('')
  const [regex, setRegex] = useState('')
  const [reason, setReason] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  function reload() {
    api.allowlist().then(setEntries)
  }

  useEffect(reload, [])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      await api.createAllowlistEntry({ ruleId, pathPattern, regex, reason })
      setRuleId('')
      setPathPattern('')
      setRegex('')
      setReason('')
      reload()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not add entry')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div>
      <header className="page-header">
        <h1>Allowlist</h1>
      </header>
      <p className="muted">
        Entries here apply org-wide: a matching finding is never persisted, for any repo, on any future scan. Use
        this for a known-safe pattern (a test fixture, a documentation placeholder) — not for dismissing one specific
        finding, which the "False positive" button on a finding does automatically.
      </p>

      <section className="panel">
        <h2>Add a rule</h2>
        <form onSubmit={handleSubmit} className="stacked-form">
          <div className="form-row">
            <label>
              Rule ID
              <input
                placeholder="e.g. generic-api-key"
                value={ruleId}
                onChange={(e) => setRuleId(e.target.value)}
              />
            </label>
            <label>
              Path glob
              <input
                placeholder="e.g. testdata/**"
                value={pathPattern}
                onChange={(e) => setPathPattern(e.target.value)}
              />
            </label>
          </div>
          <label>
            Regex (matched against the secret value)
            <input placeholder="e.g. ^sk-fake-" value={regex} onChange={(e) => setRegex(e.target.value)} />
          </label>
          <label>
            Reason
            <input placeholder="Why is this safe to ignore?" value={reason} onChange={(e) => setReason(e.target.value)} />
          </label>
          <button type="submit" className="primary-button" disabled={submitting}>
            Add rule
          </button>
        </form>
        {error && <div className="form-error">{error}</div>}
      </section>

      <section className="panel">
        <h2>Active rules ({entries.length})</h2>
        {entries.length === 0 ? (
          <p className="muted">No allowlist rules yet.</p>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>Rule ID</th>
                <th>Path</th>
                <th>Regex</th>
                <th>Fingerprint</th>
                <th>Reason</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e) => (
                <tr key={e.id}>
                  <td className="mono">{e.ruleId || '—'}</td>
                  <td className="mono">{e.pathPattern || '—'}</td>
                  <td className="mono">{e.regex || '—'}</td>
                  <td className="mono small">{e.fingerprint ? `${e.fingerprint.slice(0, 12)}…` : '—'}</td>
                  <td>{e.reason || '—'}</td>
                  <td>
                    <button
                      className="link-button danger"
                      onClick={async () => {
                        await api.deleteAllowlistEntry(e.id)
                        reload()
                      }}
                    >
                      Remove
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  )
}
