import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { initAuth, createWebStorageAdapter } from '@rcprotocol/state'

initAuth(createWebStorageAdapter())

const app = createApp(App)
app.use(router)
app.mount('#app')
