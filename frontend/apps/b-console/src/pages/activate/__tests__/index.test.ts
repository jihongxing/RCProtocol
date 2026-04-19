import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import ActivatePage from '../index.vue'

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ push: vi.fn() }),
}))

type Step = 'idle' | 'step1' | 'step2' | 'step3' | 'completed'
type FlowError = { step: number; message: string; code?: string }
type FlowResult = { asset_id: string; final_state: string }

type MockedFlow = {
  currentStep: { value: Step }
  isRunning: { value: boolean }
  error: { value: FlowError | null }
  result: { value: FlowResult | null }
  execute: ReturnType<typeof vi.fn>
  reset: ReturnType<typeof vi.fn>
}

const mockExecute = vi.fn()
const mockReset = vi.fn()

function createMockedFlow(overrides?: Partial<{
  currentStep: Step
  isRunning: boolean
  error: FlowError | null
  result: FlowResult | null
}>): MockedFlow {
  return {
    currentStep: { value: overrides?.currentStep ?? 'idle' },
    isRunning: { value: overrides?.isRunning ?? false },
    error: { value: overrides?.error ?? null },
    result: { value: overrides?.result ?? null },
    execute: mockExecute,
    reset: mockReset,
  }
}

let currentFlow = createMockedFlow()

vi.mock('../../../composables/useTypedApi', () => ({
  useTypedApi: () => ({
    console: {
      activateAsset: vi.fn(),
      entangleAsset: vi.fn(),
      confirmAssetActivation: vi.fn(),
    }
  })
}))

vi.mock('../../../composables/useActivationFlow', () => ({
  useActivationFlow: () => currentFlow
}))

vi.mock('@rcprotocol/ui/web', () => ({
  RcPageLayout: defineComponent({
    name: 'RcPageLayout',
    setup(_, { slots }) {
      return () => h('div', slots.default ? slots.default() : [])
    }
  })
}))

describe('Activate Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    currentFlow = createMockedFlow()
  })

  it('renders form with required and optional fields', () => {
    const wrapper = mount(ActivatePage)

    expect(wrapper.find('#asset-id').exists()).toBe(true)
    expect(wrapper.find('#external-product-id').exists()).toBe(true)
    expect(wrapper.find('#external-product-name').exists()).toBe(true)
    expect(wrapper.find('#external-product-url').exists()).toBe(true)
  })

  it('shows step indicator during execution', () => {
    currentFlow = createMockedFlow({ currentStep: 'step2', isRunning: true })
    const wrapper = mount(ActivatePage)

    expect(wrapper.find('.step-indicator').exists()).toBe(true)
    expect(wrapper.findAll('.step').length).toBe(3)
  })

  it('displays error with retry state', () => {
    currentFlow = createMockedFlow({ error: { step: 2, message: '绑定失败', code: 'TEST_ERROR' }, currentStep: 'step2' })
    const wrapper = mount(ActivatePage)

    expect(wrapper.text()).toContain('步骤 2 失败')
    expect(wrapper.find('.retry-btn').exists()).toBe(true)
  })

  it('displays success result after completion', () => {
    currentFlow = createMockedFlow({ currentStep: 'completed', result: { asset_id: 'asset-203', final_state: 'Activated' } })
    const wrapper = mount(ActivatePage)

    expect(wrapper.text()).toContain('激活成功')
    expect(wrapper.text()).toContain('asset-203')
  })
})
