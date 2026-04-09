import { ref } from 'vue'

export interface OperationLogEntry {
  id: string
  asset_id: string
  action: string
  from_state: string
  to_state: string
  timestamp: string
  success: boolean
}

/**
 * Composable for managing a bounded operation log.
 * Maintains newest-first ordering and drops oldest entries when exceeding maxSize.
 */
export function useOperationLog(maxSize = 20) {
  const logs = ref<OperationLogEntry[]>([])

  const append = (entry: OperationLogEntry) => {
    logs.value.unshift(entry) // Add to front (newest first)
    if (logs.value.length > maxSize) {
      logs.value = logs.value.slice(0, maxSize) // Keep only newest maxSize entries
    }
  }

  const clear = () => {
    logs.value = []
  }

  return {
    logs,
    append,
    clear
  }
}
