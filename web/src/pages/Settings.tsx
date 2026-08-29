import { useEffect, useState, type FormEvent } from 'react'
import { api, ApiError, type Webhook } from '../api'

export default function Settings() {
  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [kind, setKind] = useState<Webhook['kind']>('slack')
  const [targetUrl, setTargetUrl] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  function reload() {
    api.webhooks().then(setWebhooks)
  }

  useEffect(reload, [])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      await api.createWebhook(kind, targetUrl)
      setTargetUrl('')
      reload()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not add webhook')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div>
      <header className="page-header">
        <h1>Settings</h1>
      </header>

      <section className="panel">
        <h2>Alert webhooks</h2>
        <p className="muted small">
          Fired once per genuinely new finding — never re-fired for a finding a previous scan already reported.
        </p>
        <form onSubmit={handleSubmit} className="inline-form">
          <select value={kind} onChange={(e) => setKind(e.target.value as Webhook['kind'])}>
            <option value="slack">Slack</option>
            <option value="discord">Discord</option>
            <option value="generic">Generic JSON</option>
          </select>
          <input
            placeholder="https://hooks.slack.com/services/..."
            required
            value={targetUrl}
            onChange={(e) => setTargetUrl(e.target.value)}
          />
          <button type="submit" className="primary-button" disabled={submitting}>
            Add webhook
          </button>
        </form>
        {error && <div className="form-error">{error}</div>}

        {webhooks.length === 0 ? (
          <p className="muted">No webhooks configured.</p>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>Kind</th>
                <th>Target</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {webhooks.map((w) => (
                <tr key={w.id}>
                  <td>{w.kind}</td>
                  <td className="mono small">{w.targetUrl}</td>
                  <td>
                    <button
                      className="link-button danger"
                      onClick={async () => {
                        await api.deleteWebhook(w.id)
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
