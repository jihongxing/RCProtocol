import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import DetailPage from './vault/detail.vue'

const mockGetMyAsset = vi.fn()
const mockShowModal = vi.fn()
const mockNavigateTo = vi.fn()

vi.stubGlobal('getCurrentPages', () => ([{ options: { assetId: 'asset-1' } }]))
vi.stubGlobal('uni', {
  showModal: mockShowModal,
  navigateTo: mockNavigateTo,
  showToast: vi.fn()
})

vi.mock('@rcprotocol/state', () => ({
  useAuth: () => ({ isLoggedIn: { value: true } })
}))

vi.mock('../composables/useTypedApi', () => ({
  useTypedApi: () => ({
    app: {
      getMyAsset: mockGetMyAsset,
      consumeAsset: vi.fn(),
      legacyAsset: vi.fn()
    }
  })
}))

vi.mock('@rcprotocol/ui/uni', () => ({
  RcPageLayout: { name: 'RcPageLayout', template: '<div><slot /></div>' },
  RcLoadingState: { name: 'RcLoadingState', template: '<div />' },
  RcStatusBadge: { name: 'RcStatusBadge', template: '<div />' },
  RcRiskCard: { name: 'RcRiskCard', template: '<div />' }
}))

describe('Vault Detail Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetMyAsset.mockResolvedValue({
      asset_id: 'asset-1',
      state: 'LegallySold',
      state_label: '已售出',
      brand_id: 'b-1',
      brand_name: 'Brand A',
      product_id: 'p-1',
      product_name: 'Product A',
      created_at: '2025-01-01T00:00:00Z',
      updated_at: '2025-01-01T00:00:00Z'
    })
  })

  it('triggers detail load on mount', () => {
    mount(DetailPage)
    expect(mockGetMyAsset).toHaveBeenCalledWith('asset-1')
  })
})
