import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import BatchPage from '../index.vue'

vi.mock('@rcprotocol/ui/web', () => ({
  RcPageLayout: {
    name: 'RcPageLayout',
    template: '<div><slot /></div>'
  }
}))

describe('Batch Page', () => {
  it('shows MVP placeholder instead of asset table', () => {
    const wrapper = mount(BatchPage)

    expect(wrapper.text()).toContain('当前 MVP 未开放真实批次实体查询')
    expect(wrapper.text()).toContain('不再使用资产列表伪装为批次视图')
  })
})
