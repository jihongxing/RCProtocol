import { describe, it, expect, beforeEach } from 'vitest'
import { initAuth, useAuth } from '../index'

function createMemoryStorage() {
  const store = new Map<string, string>()
  return {
    getItem(key: string) {
      return store.get(key) || null
    },
    setItem(key: string, value: string) {
      store.set(key, value)
    },
    removeItem(key: string) {
      store.delete(key)
    }
  }
}

describe('useAuth', () => {
  beforeEach(() => {
    initAuth(createMemoryStorage())
    const auth = useAuth()
    auth.logout()
  })

  it('persists token and user on login', () => {
    const auth = useAuth()
    auth.login('token-1', {
      user_id: 'u-1',
      email: 'demo@example.com',
      display_name: 'Demo',
      org_id: 'org-1',
      org_name: 'Org',
      role: 'Consumer'
    })

    expect(auth.getToken()).toBe('token-1')
    expect(auth.user.value?.email).toBe('demo@example.com')
    expect(auth.isLoggedIn.value).toBe(true)
  })

  it('clears token and user on logout', () => {
    const auth = useAuth()
    auth.login('token-2', {
      user_id: 'u-2',
      email: 'persist@example.com',
      display_name: 'Persist',
      org_id: 'org-2',
      org_name: 'Persist Org',
      role: 'Brand',
      brand_id: 'brand-1'
    })

    auth.logout()

    expect(auth.getToken()).toBeNull()
    expect(auth.user.value).toBeNull()
    expect(auth.isLoggedIn.value).toBe(false)
  })
})
