import { useCallback, useSyncExternalStore } from 'react'
import { api } from '../api/client'
import { toApiError } from '../api/errors'
import { tokenStore } from '../api/tokenStore'

export function useAuth() {
  const token = useSyncExternalStore(tokenStore.subscribe, tokenStore.get)

  const login = useCallback(async (username: string, password: string): Promise<void> => {
    const { data, error, response } = await api.POST('/auth/login', {
      body: { username, password },
    })
    if (data === undefined) throw toApiError(response.status, error)
    tokenStore.set(data.access_token)
  }, [])

  const logout = useCallback((): void => {
    tokenStore.clear('signed-out')
  }, [])

  return {
    isAuthenticated: token !== null,
    /** True when the user ended the session themselves, as opposed to it expiring. */
    signedOutByUser: token === null && tokenStore.endReason() === 'signed-out',
    login,
    logout,
  }
}
