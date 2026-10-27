import { Link, Outlet } from 'react-router'
import { useMe } from './api/hooks'
import { useAuth } from './auth/useAuth'

export function Layout() {
  const { logout } = useAuth()
  const me = useMe()

  return (
    <>
      <header className="topbar">
        <Link to="/" className="topbar__title">
          Inventory Dashboard
        </Link>
        <div className="topbar__user">
          {me.data && (
            <span>
              <strong>{me.data.dealership.name}</strong> · {me.data.user.username}
            </span>
          )}
          <button type="button" className="secondary" onClick={logout}>
            Sign out
          </button>
        </div>
      </header>
      <main className="content">
        <Outlet />
      </main>
    </>
  )
}
