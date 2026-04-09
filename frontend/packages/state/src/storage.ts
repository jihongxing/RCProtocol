import type { StorageAdapter } from './types'

declare const uni: {
  getStorageSync(key: string): string | null | undefined
  setStorageSync(key: string, value: string): void
  removeStorageSync(key: string): void
}

/** uni-app 存储适配器，供 c-app 使用 */
export function createUniStorageAdapter(): StorageAdapter {
  return {
    getItem(key: string) {
      return uni.getStorageSync(key) || null
    },
    setItem(key: string, value: string) {
      uni.setStorageSync(key, value)
    },
    removeItem(key: string) {
      uni.removeStorageSync(key)
    }
  }
}

/** Web localStorage 适配器，供 b-console 使用 */
export function createWebStorageAdapter(): StorageAdapter {
  return {
    getItem(key: string) {
      return localStorage.getItem(key)
    },
    setItem(key: string, value: string) {
      localStorage.setItem(key, value)
    },
    removeItem(key: string) {
      localStorage.removeItem(key)
    }
  }
}
