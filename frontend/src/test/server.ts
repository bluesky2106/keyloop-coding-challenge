import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { agingVehicle, freshVehicle, makes, me, summary } from './fixtures'

/** Absolute URL of an API path, matching the client's base URL in jsdom. */
export const url = (path: string): string => `${window.location.origin}/api/v1${path}`

export function apiError(status: number, code: string, message: string, fields?: object) {
  return HttpResponse.json({ error: { code, message, fields } }, { status })
}

const vehicles = [agingVehicle, freshVehicle]

/** Sensible defaults; a test overrides only what it is about with server.use(...). */
export const server = setupServer(
  http.post(url('/auth/login'), async ({ request }) => {
    const body = (await request.json()) as { username: string; password: string }
    return body.username === 'alice' && body.password === 'secret'
      ? HttpResponse.json({ access_token: 'token-1', token_type: 'Bearer', expires_in: 3600 })
      : apiError(401, 'invalid_credentials', 'invalid username or password')
  }),
  http.get(url('/me'), () => HttpResponse.json(me)),
  http.get(url('/makes'), () => HttpResponse.json({ items: makes })),
  http.get(url('/vehicles/summary'), () => HttpResponse.json(summary)),
  http.get(url('/vehicles'), () =>
    HttpResponse.json({ items: vehicles, page: 1, page_size: 25, total: vehicles.length }),
  ),
  http.get(url('/vehicles/:id'), ({ params }) => {
    const found = vehicles.find((v) => v.id === params.id)
    return found ? HttpResponse.json(found) : apiError(404, 'not_found', 'vehicle not found')
  }),
  http.get(url('/vehicles/:id/actions'), () => HttpResponse.json({ items: [] })),
)

import type { VehiclePage } from '../api/types'

/**
 * Replaces the vehicle list handler with one that records each request's query string
 * and answers with the given page.
 */
export function captureVehicleRequests(page: Partial<VehiclePage> = {}): URLSearchParams[] {
  const requests: URLSearchParams[] = []
  server.use(
    http.get(url('/vehicles'), ({ request }) => {
      requests.push(new URL(request.url).searchParams)
      return HttpResponse.json({
        items: vehicles,
        page: 1,
        page_size: 25,
        total: vehicles.length,
        ...page,
      })
    }),
  )
  return requests
}
