import { describe, expect, it } from 'vitest'
import { ApiError, toApiError } from './errors'

describe('toApiError', () => {
  it('reads the code, message and fields of an API error body', () => {
    const error = toApiError(400, {
      error: { code: 'validation_error', message: 'invalid', fields: { note: 'too long' } },
    })
    expect(error).toBeInstanceOf(ApiError)
    expect(error).toMatchObject({
      status: 400,
      code: 'validation_error',
      message: 'invalid',
      fields: { note: 'too long' },
    })
  })

  it('copes with a body that is not an API error', () => {
    for (const body of [undefined, null, 'oops', {}, { error: {} }]) {
      const error = toApiError(502, body)
      expect(error.status).toBe(502)
      expect(error.code).toBe('unknown_error')
      expect(error.message).toBe('Request failed with status 502')
      expect(error.fields).toEqual({})
    }
  })
})
