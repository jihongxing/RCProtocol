import { config } from '@vue/test-utils'

const ignoredVueWarnings = [
  'withDirectives can only be used inside render functions.',
  'resolveComponent can only be used in render() or setup().',
]

const originalWarn = console.warn.bind(console)
const originalError = console.error.bind(console)

console.warn = (...args: unknown[]) => {
  const firstArg = typeof args[0] === 'string' ? args[0] : ''
  const secondArg = typeof args[1] === 'string' ? args[1] : ''
  const combined = `${firstArg} ${secondArg}`

  if (ignoredVueWarnings.some((warning) => combined.includes(warning))) {
    return
  }

  originalWarn(...args)
}

console.error = (...args: unknown[]) => {
  const firstArg = typeof args[0] === 'string' ? args[0] : ''
  const secondArg = typeof args[1] === 'string' ? args[1] : ''
  const combined = `${firstArg} ${secondArg}`

  if (ignoredVueWarnings.some((warning) => combined.includes(warning))) {
    return
  }

  originalError(...args)
}

config.global.config = {
  ...(config.global.config ?? {}),
  warnHandler(msg) {
    if (ignoredVueWarnings.some((warning) => msg.includes(warning))) {
      return
    }
  }
}
