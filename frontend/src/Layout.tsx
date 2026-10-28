import { Link, Outlet } from 'react-router'
import { useMe } from './api/hooks'
import { useAuth } from './auth/useAuth'
import { BrandMark } from './shared/icons'

export function Layout() {
  const { logout } = useAuth()
  const me = useMe()

  return (
    <>
      <header className="topbar">
        <div className="topbar__inner">
          <Link to="/" className="topbar__title">
            <BrandMark className="topbar__mark" />
            <span className="topbar__name">Inventory Dashboard</span>
          </Link>
          <div className="topbar__user">
            {me.data && (
              <span className="topbar__who">
                <strong>{me.data.dealership.name}</strong>
                <span className="topbar__username">{me.data.user.username}</span>
              </span>
            )}
            <button type="button" className="secondary" onClick={logout}>
              Sign out
            </button>
          </div>
        </div>
      </header>
      <main className="content">
        <Outlet />
      </main>
    </>
  )
}
