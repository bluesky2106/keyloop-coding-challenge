/** An error response from the API, shaped like the Error schema in the OpenAPI spec. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly fields: Record<string, string>

  constructor(status: number, code: string, message: string, fields: Record<string, string> = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
  }
}

interface ErrorBody {
  error?: { code?: string; message?: string; fields?: Record<string, string> }
}

/** Builds an ApiError from a non-2xx response and its parsed body (which may be anything). */
export function toApiError(status: number, body: unknown): ApiError {
  const detail = (body as ErrorBody | undefined)?.error
  return new ApiError(
    status,
    detail?.code ?? 'unknown_error',
    detail?.message ?? `Request failed with status ${String(status)}`,
    detail?.fields ?? {},
  )
}
