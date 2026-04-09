declare const uni: {
  getStorageSync(key: string): string | null | undefined
  setStorageSync(key: string, value: string): void
  removeStorageSync(key: string): void
}
