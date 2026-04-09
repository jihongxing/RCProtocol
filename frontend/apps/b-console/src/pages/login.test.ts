import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import LoginPage from './login.vue'

const mockReplace = vi.fn()
const mockLogin = vi.fn()
const mockAuthApi = {
  login: vi.fn(),
  selectOrg: vi.fn(),
}

vi.mock('vue-router', () => ({
  useRouter: () => ({ replace: mockReplace })
}))

vi.mock('@rcprotocol/state', () => ({
  useAuth: () => ({ login: mockLogin })
}))

vi.mock('../composables/useTypedApi', () => ({
  useTypedApi: () => ({ auth: mockAuthApi })
}))

vi.mock('@rcprotocol/ui/web', () => ({
  RcRiskCard: defineComponent({
    name: 'RcRiskCard',
    props: {
      message: {
        type: String,
        default: ''
      }
    },
    setup(props) {
      return () => h('div', props.message)
    }
  })
}))

describe('B Console Login Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders email, password and submit controls', () => {
    const wrapper = mount(LoginPage)
    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
    expect(wrapper.find('input[type="password"]').exists()).toBe(true)
    expect(wrapper.find('button').text()).toContain('登录')
  })
})
