import type { ReactNode } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import Layout from './components/Layout'
import { AuthProvider, useAuth } from './context/AuthContext'
import Allowlist from './pages/Allowlist'
import AuthPage from './pages/AuthPage'
import Dashboard from './pages/Dashboard'
import FindingDetail from './pages/FindingDetail'
import Repos from './pages/Repos'
import Settings from './pages/Settings'

function RequireAuth({ children }: { children: ReactNode }) {
  const { loading, session } = useAuth()
  if (loading) return null
  if (!session?.authenticated) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function LoginRoute() {
  const { loading, session } = useAuth()
  if (loading) return null
  if (session?.authenticated) {
    return <Navigate to="/" replace />
  }
  return <AuthPage />
}

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginRoute />} />
        <Route
          element={
            <RequireAuth>
              <Layout />
            </RequireAuth>
          }
        >
          <Route path="/" element={<Dashboard />} />
          <Route path="/findings/:id" element={<FindingDetail />} />
          <Route path="/repos" element={<Repos />} />
          <Route path="/allowlist" element={<Allowlist />} />
          <Route path="/settings" element={<Settings />} />
        </Route>
      </Routes>
    </AuthProvider>
  )
}
