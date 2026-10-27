import { Navigate, Outlet, useLocation } from 'react-router'
import { useAuth } from './useAuth'

/** Renders the nested routes for a signed-in user and sends everyone else to the login page. */
export function RequireAuth() {
  const { isAuthenticated } = useAuth()
  const location = useLocation()

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location }} />
  }
  return <Outlet />
}
