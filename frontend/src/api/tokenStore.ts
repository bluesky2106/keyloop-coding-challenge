// The access token lives in sessionStorage: it survives a page refresh but is cleared when
// the tab closes. The trade-off against memory-only storage is documented in
// docs/system-design.md (section 5). Storage can be unavailable (private windows, blocked
// site data), so every access is guarded and the store falls back to memory.

const KEY = 'keyloop.access_token'

function read(): string | null {
  try {
    return sessionStorage.getItem(KEY)
  } catch {
    return null
  }
}

let token: string | null = read()
const listeners = new Set<() => void>()

function emit(): void {
  listeners.forEach((listener) => {
    listener()
  })
}

// Arrow functions, not methods, so they can be passed around (useSyncExternalStore) unbound.
export const tokenStore = {
  get: (): string | null => token,

  set: (value: string): void => {
    token = value
    try {
      sessionStorage.setItem(KEY, value)
    } catch {
      // Memory-only is fine: the user simply signs in again after a refresh.
    }
    emit()
  },

  clear: (): void => {
    token = null
    try {
      sessionStorage.removeItem(KEY)
    } catch {
      // Nothing stored, nothing to remove.
    }
    emit()
  },

  /** For useSyncExternalStore: the callback runs whenever the token changes. */
  subscribe: (listener: () => void): (() => void) => {
    listeners.add(listener)
    return () => {
      listeners.delete(listener)
    }
  },
}
