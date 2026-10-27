import { QueryClient } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router'
import App from '../App'
import { tokenStore } from '../api/tokenStore'
import { Providers } from '../Providers'

/** Shows the router's current URL so tests can assert on it. */
function LocationProbe() {
  const { pathname, search } = useLocation()
  return <output data-testid="location">{pathname + search}</output>
}

interface Options {
  route?: string
  /** Sign in before rendering. Pass null to start signed out. */
  token?: string | null
}

export function renderApp({ route = '/', token = 'token-1' }: Options = {}) {
  if (token !== null) tokenStore.set(token)
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const user = userEvent.setup()
  const result = render(
    <Providers client={client}>
      <MemoryRouter initialEntries={[route]}>
        <App />
        <LocationProbe />
      </MemoryRouter>
    </Providers>,
  )
  return { user, client, ...result }
}
