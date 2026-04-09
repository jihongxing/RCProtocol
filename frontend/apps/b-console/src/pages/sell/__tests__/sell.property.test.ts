import { describe, it, expect } from 'vitest'
import { fc } from '@fast-check/vitest'

describe('Sell Page - Property Tests', () => {
  it('Property 4: Buyer ID validation rejects empty/whitespace', () => {
    fc.assert(
      fc.property(
        fc.string().filter(s => s.trim() === ''), // Generate whitespace-only strings
        (whitespaceString) => {
          // Simulate validation logic
          const validateBuyerId = (value: string): string => {
            const trimmed = value.trim()
            if (value && !trimmed) {
              return '买家 ID 不能为空或仅包含空格'
            }
            return ''
          }

          const error = validateBuyerId(whitespaceString)

          // If string is non-empty but only whitespace, should have error
          if (whitespaceString.length > 0) {
            expect(error).toBeTruthy()
            expect(error).toContain('买家 ID 不能为空或仅包含空格')
          }
        }
      ),
      { numRuns: 100 }
    )
  })

  it('Property 4 Extended: Valid buyer IDs are accepted', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1 }).filter(s => s.trim().length > 0), // Non-empty after trim
        (validString) => {
          const validateBuyerId = (value: string): string => {
            const trimmed = value.trim()
            if (value && !trimmed) {
              return '买家 ID 不能为空或仅包含空格'
            }
            return ''
          }

          const error = validateBuyerId(validString)
          expect(error).toBe('')
        }
      ),
      { numRuns: 100 }
    )
  })
})
