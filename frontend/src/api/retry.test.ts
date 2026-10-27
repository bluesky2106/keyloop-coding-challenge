import { describe, expect, it } from 'vitest'
import { ApiError } from './errors'
import { shouldRetry } from './retry'

describe('shouldRetry', () => {
  it('retries a server or network failure once', () => {
    expect(shouldRetry(0, new ApiError(500, 'internal_error', 'boom'))).toBe(true)
    expect(shouldRetry(0, new TypeError('network down'))).toBe(true)
    expect(shouldRetry(1, new ApiError(500, 'internal_error', 'boom'))).toBe(false)
  })

  it.each([400, 401, 404, 409])('does not retry a %i, which would only repeat', (status) => {
    expect(shouldRetry(0, new ApiError(status, 'x', 'x'))).toBe(false)
  })
})
