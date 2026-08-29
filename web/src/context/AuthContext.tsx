import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { api, type SessionInfo } from '../api'

interface AuthState {
  loading: boolean
  session: SessionInfo | null
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<SessionInfo | null>(null)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    const s = await api.session()
    setSession(s)
  }, [])

  useEffect(() => {
    refresh().finally(() => setLoading(false))
  }, [refresh])

  return <AuthContext.Provider value={{ loading, session, refresh }}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
