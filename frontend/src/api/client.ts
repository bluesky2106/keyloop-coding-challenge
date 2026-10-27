import createClient, { type Middleware } from 'openapi-fetch'
import type { paths } from './schema'
import { tokenStore } from './tokenStore'

const withToken: Middleware = {
  onRequest({ request }) {
    const token = tokenStore.get()
    if (token) request.headers.set('Authorization', `Bearer ${token}`)
    return request
  },

  onResponse({ request, response }) {
    // An expired or revoked token on any call signs the user out: the app reacts to the
    // token disappearing. A failed login is also a 401 but there is no session to end.
    if (response.status === 401 && !new URL(request.url).pathname.endsWith('/auth/login')) {
      tokenStore.clear()
    }
    return response
  },
}

/**
 * The typed API client. Paths and payloads come from docs/api/openapi.yaml through the
 * generated schema, so a change to the contract is a compile error here.
 */
export const api = createClient<paths>({
  baseUrl: `${window.location.origin}/api/v1`,
  // Resolved on every call so tests can replace the global fetch.
  fetch: (request) => globalThis.fetch(request),
})
api.use(withToken)
