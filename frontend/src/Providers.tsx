import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useEffect, useState, type ReactNode } from 'react'
import { shouldRetry } from './api/retry'
import { tokenStore } from './api/tokenStore'

interface Props {
  children: ReactNode
  /** Tests pass their own client (for example with retries off). */
  client?: QueryClient
}

export function Providers({ children, client }: Props) {
  const [queryClient] = useState(
    () =>
      client ??
      new QueryClient({
        defaultOptions: { queries: { retry: shouldRetry, refetchOnWindowFocus: false } },
      }),
  )

  // Never show one user's cached data to the next: drop everything when the session ends,
  // whether by signing out or by the API rejecting an expired token.
  useEffect(
    () =>
      tokenStore.subscribe(() => {
        if (tokenStore.get() === null) queryClient.clear()
      }),
    [queryClient],
  )

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}
