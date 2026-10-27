import { Navigate, Outlet, useLocation } from 'react-router'
import { useAuth } from './useAuth'

/** Renders the nested routes for a signed-in user and sends everyone else to the login page. */
export function RequireAuth() {
  const { isAuthenticated, signedOutByUser } = useAuth()
  const location = useLocation()

  if (!isAuthenticated) {
    // After an expiry the user wants their page back. After choosing to sign out they do not,
    // and the next person to sign in on this tab certainly should not land on it.
    return <Navigate to="/login" replace state={signedOutByUser ? null : { from: location }} />
  }
  return <Outlet />
}
