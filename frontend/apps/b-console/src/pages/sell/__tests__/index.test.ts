import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import SellPage from '../index.vue'

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ push: vi.fn() }),
}))

const mockLegalSellAsset = vi.fn()

vi.mock('../../../composables/useTypedApi', () => ({
  useTypedApi: () => ({
    console: {
      legalSellAsset: mockLegalSellAsset,
    }
  })
}))

vi.mock('../../../composables/useIdempotency', () => ({
  useIdempotency: () => ({
    key: { value: 'test-key-456' },
    regenerate: vi.fn()
  })
}))

vi.mock('../../../composables/useOperationLog', () => ({
  useOperationLog: () => ({
    logs: { value: [] },
    append: vi.fn(),
    clear: vi.fn()
  })
}))

vi.mock('@rcprotocol/ui/web', () => ({
  RcPageLayout: defineComponent({
    name: 'RcPageLayout',
    setup(_, { slots }) {
      return () => h('div', slots.default ? slots.default() : [])
    }
  })
}))

describe('Sell Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders form with asset id and buyer id inputs', () => {
    const wrapper = mount(SellPage)

    expect(wrapper.find('#asset-id').exists()).toBe(true)
    expect(wrapper.find('#buyer-id').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').exists()).toBe(true)
  })
})
