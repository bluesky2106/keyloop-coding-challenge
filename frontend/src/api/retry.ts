import { ApiError } from './errors'

/**
 * Retry a failed query once, unless the server has said no: a 4xx response (not found,
 * invalid, unauthorised) will give the same answer again.
 */
export function shouldRetry(failureCount: number, error: Error): boolean {
  if (error instanceof ApiError && error.status >= 400 && error.status < 500) return false
  return failureCount < 1
}
