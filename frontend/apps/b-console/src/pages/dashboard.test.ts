import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import DashboardPage from './dashboard.vue'

const mockGetDashboard = vi.fn()

vi.mock('../composables/useTypedApi', () => ({
  useTypedApi: () => ({
    console: { getDashboard: mockGetDashboard }
  })
}))

vi.mock('@rcprotocol/ui/web', () => ({
  RcPageLayout: { name: 'RcPageLayout', template: '<div><slot /></div>' },
  RcLoadingState: { name: 'RcLoadingState', props: ['loading', 'error'], template: '<div />' }
}))

describe('Dashboard Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetDashboard.mockResolvedValue({})
  })

  it('renders dashboard shell and triggers load', () => {
    const wrapper = mount(DashboardPage)
    expect(wrapper.text()).toContain('刷新')
    expect(mockGetDashboard).toHaveBeenCalled()
  })
})
