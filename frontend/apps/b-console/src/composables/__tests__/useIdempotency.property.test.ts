import { describe, it, expect } from 'vitest'
import { fc } from '@fast-check/vitest'
import { useIdempotency } from '../useIdempotency'

describe('useIdempotency - Property Tests', () => {
  it('Property 1: All generated keys are valid UUID v4 and N generations produce N distinct keys', () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 200 }),
        (n) => {
          const keys = new Set<string>()
          const uuidV4Regex = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

          for (let i = 0; i < n; i++) {
            const { key, regenerate } = useIdempotency()
            const currentKey = key.value

            // Verify UUID v4 format
            expect(currentKey).toMatch(uuidV4Regex)
            keys.add(currentKey)

            // Test regenerate
            regenerate()
            const newKey = key.value
            expect(newKey).toMatch(uuidV4Regex)
            keys.add(newKey)
          }

          // Verify all keys are distinct (N calls + N regenerates = 2N keys)
          expect(keys.size).toBe(n * 2)
        }
      ),
      { numRuns: 100 }
    )
  })
})
