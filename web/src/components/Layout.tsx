import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useAuth } from '../context/AuthContext'

export default function Layout() {
  const { session, refresh } = useAuth()
  const navigate = useNavigate()

  async function handleLogout() {
    await api.logout()
    await refresh()
    navigate('/login')
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark">◆</span> Leakboard
        </div>
        <nav>
          <NavLink to="/" end>
            Findings
          </NavLink>
          <NavLink to="/repos">Repos</NavLink>
          <NavLink to="/allowlist">Allowlist</NavLink>
          <NavLink to="/settings">Settings</NavLink>
        </nav>
        <div className="sidebar-footer">
          <span className="sidebar-email">{session?.email}</span>
          <button className="link-button" onClick={handleLogout}>
            Sign out
          </button>
        </div>
      </aside>
      <main className="content">
        <Outlet />
      </main>
    </div>
  )
}
