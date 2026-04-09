import { createSSRApp } from 'vue'
import App from './App.vue'
import { initAuth, createUniStorageAdapter } from '@rcprotocol/state'

initAuth(createUniStorageAdapter())

export function createApp() {
  const app = createSSRApp(App)
  return { app }
}
