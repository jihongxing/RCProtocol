import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import VaultPage from './vault/index.vue'

const mockListMyAssets = vi.fn()
const mockNavigateTo = vi.fn()

vi.stubGlobal('uni', { navigateTo: mockNavigateTo })

vi.mock('../composables/useTypedApi', () => ({
  useTypedApi: () => ({ app: { listMyAssets: mockListMyAssets } })
}))

vi.mock('@rcprotocol/ui/uni', () => ({
  RcPageLayout: { name: 'RcPageLayout', template: '<div><slot /></div>' },
  RcLoadingState: { name: 'RcLoadingState', template: '<div />' },
  RcEmptyState: { name: 'RcEmptyState', props: ['message'], template: '<div>{{ message }}</div>' },
  RcStatusBadge: { name: 'RcStatusBadge', template: '<div />' }
}))

describe('Vault Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockListMyAssets.mockResolvedValue({ items: [], total: 0, page: 1 })
  })

  it('renders vault tabs and triggers load', () => {
    const wrapper = mount(VaultPage)
    expect(wrapper.text()).toContain('活跃资产')
    expect(wrapper.text()).toContain('荣誉典藏')
    expect(mockListMyAssets).toHaveBeenCalled()
  })
})
