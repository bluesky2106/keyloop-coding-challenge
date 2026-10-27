import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { tokenStore } from '../api/tokenStore'
import { server } from './server'

// A request no test has described is a bug in the test, not something to ignore.
beforeAll(() => {
  server.listen({ onUnhandledFrame: 'error' })
})

afterEach(() => {
  cleanup()
  server.resetHandlers()
  tokenStore.clear()
  sessionStorage.clear()
})

afterAll(() => {
  server.close()
})
