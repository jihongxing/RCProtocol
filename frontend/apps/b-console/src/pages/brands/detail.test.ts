import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import BrandDetailPage from './detail.vue'

const mockGetBrand = vi.fn()
const mockListBrandProducts = vi.fn()

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: { brandId: 'brand-1' } }),
  useRouter: () => ({ push: vi.fn() })
}))

vi.mock('@rcprotocol/state', () => ({
  useAuth: () => ({ user: { value: { role: 'Platform' } } })
}))

vi.mock('../../composables/useTypedApi', () => ({
  useTypedApi: () => ({
    console: {
      getBrand: mockGetBrand,
      listBrandProducts: mockListBrandProducts
    }
  })
}))

vi.mock('@rcprotocol/ui/web', () => ({
  RcPageLayout: { name: 'RcPageLayout', template: '<div><slot /></div>' },
  RcLoadingState: { name: 'RcLoadingState', template: '<div />' },
  RcEmptyState: { name: 'RcEmptyState', template: '<div>empty</div>' },
  RcStatusBadge: { name: 'RcStatusBadge', template: '<div />' }
}))

describe('Brand Detail Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetBrand.mockResolvedValue({
      brand_id: 'brand-1',
      brand_name: 'Brand A',
      status: 'active',
      created_at: '2025-01-01T00:00:00Z'
    })
    mockListBrandProducts.mockResolvedValue({
      items: [{
        product_id: 'p-1',
        product_name: 'Product A',
        brand_id: 'brand-1',
        status: 'active',
        created_at: '2025-01-01T00:00:00Z'
      }]
    })
  })

  it('triggers typed api calls on mount', () => {
    mount(BrandDetailPage)
    expect(mockGetBrand).toHaveBeenCalledWith('brand-1')
    expect(mockListBrandProducts).toHaveBeenCalledWith('brand-1')
  })
})
