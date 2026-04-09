import { describe, it, expect } from 'vitest'
import { formatDate, formatRelativeTime, truncateId } from '../format'

describe('utils formatters', () => {
  it('truncates ids deterministically', () => {
    expect(truncateId('abcdef123456', 6)).toBe('abcdef...')
    expect(truncateId('short', 8)).toBe('short')
  })

  it('formats absolute dates', () => {
    expect(formatDate('2025-01-02T03:04:00.000Z')).toMatch(/2025-01-02/)
  })

  it('formats relative times', () => {
    expect(typeof formatRelativeTime(new Date().toISOString())).toBe('string')
  })
})
