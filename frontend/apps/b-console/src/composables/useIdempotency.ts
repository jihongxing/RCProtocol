import { ref } from 'vue'

/**
 * Composable for generating and managing idempotency keys for write operations.
 * Uses UUID v4 format for guaranteed uniqueness.
 */
export function useIdempotency() {
  const key = ref<string>(crypto.randomUUID())

  const regenerate = () => {
    key.value = crypto.randomUUID()
  }

  return {
    key,
    regenerate
  }
}
