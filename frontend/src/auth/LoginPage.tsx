import { useState, type SyntheticEvent } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router'
import { ApiError } from '../api/errors'
import { useAuth } from './useAuth'

interface LocationState {
  from?: { pathname: string; search: string }
}

function messageFor(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 401) return 'Invalid username or password.'
    if (error.status === 400) return 'Enter your username and password.'
  }
  return 'Something went wrong. Please try again.'
}

export function LoginPage() {
  const { isAuthenticated, login } = useAuth()
  const navigate = useNavigate()
  const from = (useLocation().state as LocationState | null)?.from
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  if (isAuthenticated) return <Navigate to="/" replace />

  async function submit(): Promise<void> {
    setPending(true)
    setError(null)
    try {
      await login(username, password)
      void navigate(from ? `${from.pathname}${from.search}` : '/', { replace: true })
    } catch (err) {
      setError(messageFor(err))
      setPending(false)
    }
  }

  function onSubmit(event: SyntheticEvent): void {
    event.preventDefault()
    void submit()
  }

  return (
    <main className="login">
      <h1>Inventory Dashboard</h1>
      <form onSubmit={onSubmit} className="card login__form">
        <h2>Sign in</h2>
        <label>
          Username
          <input
            value={username}
            onChange={(e) => {
              setUsername(e.target.value)
            }}
            autoComplete="username"
            required
          />
        </label>
        <label>
          Password
          <input
            type="password"
            value={password}
            onChange={(e) => {
              setPassword(e.target.value)
            }}
            autoComplete="current-password"
            required
          />
        </label>
        {error && (
          <p role="alert" className="error">
            {error}
          </p>
        )}
        <button type="submit" disabled={pending}>
          {pending ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </main>
  )
}
