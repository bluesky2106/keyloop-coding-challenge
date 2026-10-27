import { afterEach, describe, expect, it, vi } from 'vitest'
import { tokenStore } from './tokenStore'

afterEach(() => {
  vi.restoreAllMocks()
  vi.resetModules()
})

describe('tokenStore', () => {
  it('holds a token and forgets it on clear', () => {
    tokenStore.set('abc')
    expect(tokenStore.get()).toBe('abc')
    tokenStore.clear()
    expect(tokenStore.get()).toBeNull()
  })

  it('remembers why the session ended, and forgets it on the next sign-in', () => {
    tokenStore.set('abc')
    expect(tokenStore.endReason()).toBeNull()

    tokenStore.clear('signed-out')
    expect(tokenStore.endReason()).toBe('signed-out')

    tokenStore.set('def')
    expect(tokenStore.endReason()).toBeNull()

    tokenStore.clear()
    expect(tokenStore.endReason()).toBe('expired')
  })

  it('persists the token for the tab, so a refresh keeps the session', async () => {
    tokenStore.set('abc')
    expect(sessionStorage.getItem('keyloop.access_token')).toBe('abc')

    vi.resetModules()
    const reloaded = await import('./tokenStore')
    expect(reloaded.tokenStore.get()).toBe('abc')

    tokenStore.clear()
    expect(sessionStorage.getItem('keyloop.access_token')).toBeNull()
  })

  it('notifies subscribers of every change until they unsubscribe', () => {
    const listener = vi.fn()
    const unsubscribe = tokenStore.subscribe(listener)

    tokenStore.set('a')
    tokenStore.clear()
    expect(listener).toHaveBeenCalledTimes(2)

    unsubscribe()
    tokenStore.set('b')
    expect(listener).toHaveBeenCalledTimes(2)
  })

  it('falls back to memory when storage is unavailable', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    vi.spyOn(Storage.prototype, 'removeItem').mockImplementation(() => {
      throw new Error('blocked')
    })

    tokenStore.set('abc')
    expect(tokenStore.get()).toBe('abc')
    tokenStore.clear()
    expect(tokenStore.get()).toBeNull()
  })

  it('starts signed out when storage cannot be read', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    vi.resetModules()

    const fresh = await import('./tokenStore')

    expect(fresh.tokenStore.get()).toBeNull()
  })
})
