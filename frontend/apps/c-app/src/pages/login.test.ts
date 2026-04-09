import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import LoginPage from './login.vue'

const mockSwitchTab = vi.fn()
const mockLogin = vi.fn()
const mockAuthApi = {
  login: vi.fn(),
  selectOrg: vi.fn(),
}

vi.stubGlobal('uni', {
  switchTab: mockSwitchTab
})

vi.mock('@rcprotocol/state', () => ({
  useAuth: () => ({ login: mockLogin })
}))

vi.mock('../composables/useTypedApi', () => ({
  useTypedApi: () => ({ auth: mockAuthApi })
}))

vi.mock('@rcprotocol/ui/uni', () => ({
  RcPageLayout: defineComponent({
    name: 'RcPageLayout',
    setup(_, { slots }) {
      return () => h('div', slots.default ? slots.default() : [])
    }
  }),
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

describe('C App Login Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders email, password and submit controls', () => {
    const wrapper = mount(LoginPage, {
      global: {
        config: {
          compilerOptions: {
            isCustomElement: (tag) => ['view', 'text', 'picker', 'input', 'button'].includes(tag)
          }
        }
      }
    })
    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
    expect(wrapper.find('input[type="safe-password"]').exists()).toBe(true)
    expect(wrapper.find('button').text()).toContain('登录')
  })
})
