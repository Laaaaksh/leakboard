import { useEffect, useState, type FormEvent } from 'react'
import { api, ApiError, type Connection, type Repo } from '../api'

export default function Repos() {
  const [repos, setRepos] = useState<Repo[]>([])
  const [connections, setConnections] = useState<Connection[]>([])
  const [scanning, setScanning] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)

  function reload() {
    api.repos().then(setRepos)
    api.connections().then(setConnections)
  }

  useEffect(reload, [])

  async function scanNow(id: number) {
    setScanning(id)
    setError(null)
    try {
      await api.scanRepoNow(id)
      reload()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Scan failed')
    } finally {
      setScanning(null)
    }
  }

  async function removeRepo(id: number) {
    await api.deleteRepo(id)
    reload()
  }

  return (
    <div>
      <header className="page-header">
        <h1>Repos</h1>
      </header>

      <ConnectOrgForm onConnected={reload} />
      <AddRepoForm onAdded={reload} />

      {connections.length > 0 && (
        <section className="panel">
          <h2>Connected organizations</h2>
          <table className="data-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>GitHub org</th>
                <th>Connected</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {connections.map((c) => (
                <tr key={c.id}>
                  <td>{c.name}</td>
                  <td className="mono">{c.githubOrg}</td>
                  <td className="muted">{new Date(c.createdAt).toLocaleDateString()}</td>
                  <td>
                    <button
                      className="link-button"
                      onClick={async () => {
                        await api.syncConnection(c.id)
                        reload()
                      }}
                    >
                      Sync repos
                    </button>
                    <button
                      className="link-button danger"
                      onClick={async () => {
                        await api.deleteConnection(c.id)
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
        </section>
      )}

      {error && <div className="form-error">{error}</div>}

      <section className="panel">
        <h2>Tracked repos ({repos.length})</h2>
        {repos.length === 0 ? (
          <p className="muted">No repos tracked yet.</p>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>Repo</th>
                <th>Last scanned</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {repos.map((r) => (
                <tr key={r.id}>
                  <td>
                    <div>{r.name}</div>
                    {r.lastScanError && <div className="error-text">{r.lastScanError}</div>}
                  </td>
                  <td className="muted">{r.lastScannedAt ? new Date(r.lastScannedAt).toLocaleString() : 'never'}</td>
                  <td>
                    <button className="link-button" disabled={scanning === r.id} onClick={() => scanNow(r.id)}>
                      {scanning === r.id ? 'Scanning…' : 'Scan now'}
                    </button>
                    <button className="link-button danger" onClick={() => removeRepo(r.id)}>
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

function ConnectOrgForm({ onConnected }: { onConnected: () => void }) {
  const [name, setName] = useState('')
  const [org, setOrg] = useState('')
  const [token, setToken] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [result, setResult] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    setResult(null)
    try {
      const res = await api.createConnection(name, org, token)
      setResult(`Connected. ${res.reposAdded} repo(s) added.`)
      setName('')
      setOrg('')
      setToken('')
      onConnected()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not connect')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section className="panel">
      <h2>Connect a GitHub organization</h2>
      <p className="muted small">
        Every non-archived repo the token can see in this org is added and scanned on a schedule. Use a personal
        access token (classic or fine-grained) with read access to the org's repos.
      </p>
      <form onSubmit={handleSubmit} className="inline-form">
        <input placeholder="Display name" required value={name} onChange={(e) => setName(e.target.value)} />
        <input placeholder="GitHub org" required value={org} onChange={(e) => setOrg(e.target.value)} />
        <input
          placeholder="Access token"
          type="password"
          required
          value={token}
          onChange={(e) => setToken(e.target.value)}
        />
        <button type="submit" className="primary-button" disabled={submitting}>
          Connect
        </button>
      </form>
      {result && <div className="form-success">{result}</div>}
      {error && <div className="form-error">{error}</div>}
    </section>
  )
}

function AddRepoForm({ onAdded }: { onAdded: () => void }) {
  const [name, setName] = useState('')
  const [cloneUrl, setCloneUrl] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      await api.createRepo(name, cloneUrl)
      setName('')
      setCloneUrl('')
      onAdded()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not add repo')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section className="panel">
      <h2>Add a single repo</h2>
      <p className="muted small">For a repo outside a connected org, or a non-GitHub git remote.</p>
      <form onSubmit={handleSubmit} className="inline-form">
        <input placeholder="Display name" required value={name} onChange={(e) => setName(e.target.value)} />
        <input
          placeholder="Clone URL (https://...)"
          required
          value={cloneUrl}
          onChange={(e) => setCloneUrl(e.target.value)}
        />
        <button type="submit" className="primary-button" disabled={submitting}>
          Add repo
        </button>
      </form>
      {error && <div className="form-error">{error}</div>}
    </section>
  )
}
