import { describe, it, expect } from 'vitest'
import { fc } from '@fast-check/vitest'
import { useOperationLog } from '../useOperationLog'
import type { OperationLogEntry } from '../useOperationLog'

describe('useOperationLog - Property Tests', () => {
  it('Property 2: After M appends, log length ≤ maxSize, newest entries preserved, ordering is newest-first', () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 50 }), // maxSize
        fc.integer({ min: 1, max: 100 }), // number of appends
        (maxSize, numAppends) => {
          const { logs, append } = useOperationLog(maxSize)
          const appendedEntries: OperationLogEntry[] = []

          // Append M entries
          for (let i = 0; i < numAppends; i++) {
            const entry: OperationLogEntry = {
              id: `op-${i}`,
              asset_id: `asset-${i}`,
              action: 'test',
              from_state: 'A',
              to_state: 'B',
              timestamp: new Date(Date.now() + i).toISOString(),
              success: true
            }
            appendedEntries.push(entry)
            append(entry)
          }

          // Verify bounded size
          expect(logs.value.length).toBeLessThanOrEqual(maxSize)

          // Verify newest entries preserved
          const expectedSize = Math.min(numAppends, maxSize)
          expect(logs.value.length).toBe(expectedSize)

          // Verify newest-first ordering
          for (let i = 0; i < logs.value.length; i++) {
            const expectedEntry = appendedEntries[numAppends - 1 - i]
            expect(logs.value[i].id).toBe(expectedEntry.id)
          }
        }
      ),
      { numRuns: 100 }
    )
  })
})
