import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ApiKeysPage from './api-keys.vue'

const mockListApiKeys = vi.fn()

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: { brandId: 'brand-1' } })
}))

vi.mock('@rcprotocol/state', () => ({
  useAuth: () => ({ user: { value: { role: 'Platform', brand_id: 'brand-1' } } })
}))

vi.mock('../../composables/useTypedApi', () => ({
  useTypedApi: () => ({
    console: {
      listApiKeys: mockListApiKeys,
      createApiKey: vi.fn(),
      revokeApiKey: vi.fn()
    }
  })
}))

vi.mock('@rcprotocol/ui/web', () => ({
  RcPageLayout: { name: 'RcPageLayout', template: '<div><slot /></div>' },
  RcLoadingState: { name: 'RcLoadingState', template: '<div />' },
  RcEmptyState: { name: 'RcEmptyState', template: '<div>empty</div>' },
  RcStatusBadge: { name: 'RcStatusBadge', template: '<div />' },
  RcForbiddenState: { name: 'RcForbiddenState', props: ['message'], template: '<div>{{ message }}</div>' }
}))

describe('API Keys Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockListApiKeys.mockResolvedValue({
      items: [{
        key_id: 'k-1',
        description: 'desc',
        status: 'active',
        created_at: '2025-01-01T00:00:00Z'
      }]
    })
  })

  it('renders api key page shell and triggers typed api load', () => {
    const wrapper = mount(ApiKeysPage)
    expect(wrapper.text()).toContain('创建 API Key')
    expect(mockListApiKeys).toHaveBeenCalledWith('brand-1')
  })
})
