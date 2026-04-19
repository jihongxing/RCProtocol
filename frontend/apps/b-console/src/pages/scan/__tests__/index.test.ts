import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import ScanPage from '../index.vue'

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ push: vi.fn() }),
}))

const mockBlindLogAsset = vi.fn()
const mockStockInAsset = vi.fn()

vi.mock('../../../composables/useTypedApi', () => ({
  useTypedApi: () => ({
    console: {
      blindLogAsset: mockBlindLogAsset,
      stockInAsset: mockStockInAsset,
    }
  })
}))

vi.mock('../../../composables/useIdempotency', () => ({
  useIdempotency: () => ({
    key: { value: 'test-key-123' },
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

describe('Scan Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders form with asset id input and operation selector', () => {
    const wrapper = mount(ScanPage)

    expect(wrapper.find('#asset-id').exists()).toBe(true)
    expect(wrapper.find('#operation-type').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').exists()).toBe(true)
  })
})
