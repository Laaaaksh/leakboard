import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, ApiError } from '../api'
import { useAuth } from '../context/AuthContext'

export default function AuthPage() {
  const { session, refresh } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const isSetup = session?.setupRequired === true

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      if (isSetup) {
        await api.setup(email, password)
      } else {
        await api.login(email, password)
      }
      await refresh()
      navigate('/')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="brand brand-lg">
          <span className="brand-mark">◆</span> Leakboard
        </div>
        <p className="auth-subtitle">
          {isSetup ? 'Create the admin account for this instance.' : 'Sign in to your self-hosted dashboard.'}
        </p>
        <form onSubmit={handleSubmit}>
          <label>
            Email
            <input type="email" required value={email} onChange={(e) => setEmail(e.target.value)} autoFocus />
          </label>
          <label>
            Password
            <input
              type="password"
              required
              minLength={8}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </label>
          {error && <div className="form-error">{error}</div>}
          <button type="submit" className="primary-button" disabled={submitting}>
            {isSetup ? 'Create account' : 'Sign in'}
          </button>
        </form>
      </div>
    </div>
  )
}
