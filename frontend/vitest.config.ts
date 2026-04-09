import { defineConfig, type UserWorkspaceConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

type VitestPlugin = NonNullable<UserWorkspaceConfig['plugins']>[number]

const projectRoot = new URL('.', import.meta.url)
const vuePlugin = vue() as unknown as VitestPlugin

export default defineConfig({
  plugins: [vuePlugin],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./test/setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        '**/*.config.ts',
        '**/*.d.ts',
        '**/dist/',
        '**/build/',
        '**/__tests__/',
      ],
    },
  },
  resolve: {
    alias: {
      '@rcprotocol/api': new URL('./packages/api/src', projectRoot).pathname,
      '@rcprotocol/state': new URL('./packages/state/src', projectRoot).pathname,
      '@rcprotocol/ui': new URL('./packages/ui/src', projectRoot).pathname,
      '@rcprotocol/utils': new URL('./packages/utils/src', projectRoot).pathname,
    },
  },
})
