import { ref, computed } from 'vue'
import type { User, StorageAdapter } from './types'

const TOKEN_KEY = 'rc_token'
const USER_KEY = 'rc_user'

const token = ref<string | null>(null)
const user = ref<User | null>(null)

let _storage: StorageAdapter | null = null

export function initAuth(storage: StorageAdapter) {
  _storage = storage
}

export function useAuth() {
  const isLoggedIn = computed(() => !!token.value)

  function login(newToken: string, newUser: User) {
    token.value = newToken
    user.value = newUser
    _storage?.setItem(TOKEN_KEY, newToken)
    _storage?.setItem(USER_KEY, JSON.stringify(newUser))
  }

  function logout(redirect?: () => void) {
    token.value = null
    user.value = null
    _storage?.removeItem(TOKEN_KEY)
    _storage?.removeItem(USER_KEY)
    redirect?.()
  }

  function loadFromStorage() {
    if (!_storage) return
    const savedToken = _storage.getItem(TOKEN_KEY)
    const savedUser = _storage.getItem(USER_KEY)
    if (savedToken) {
      token.value = savedToken
    }
    if (savedUser) {
      try {
        user.value = JSON.parse(savedUser)
      } catch {
        user.value = null
      }
    }
  }

  function getToken(): string | null {
    return token.value
  }

  return { token, user, isLoggedIn, login, logout, loadFromStorage, getToken }
}
