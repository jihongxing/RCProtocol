import { describe, it, expect } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { useNfcReader } from '../useNfcReader'

const Harness = defineComponent({
  setup() {
    return useNfcReader()
  },
  template: '<div />'
})

describe('useNfcReader - parseSunMessage', () => {
  it('should parse valid 18-byte SUN message', () => {
    const wrapper = mount(Harness)
    const { parseSunMessage } = wrapper.vm

    const data = new Uint8Array([
      0x04, 0xE1, 0xA2, 0xB3, 0xC4, 0xD5, 0xE6,
      0x00, 0x00, 0x01,
      0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF
    ]).buffer

    const result = parseSunMessage(data)

    expect(result).toEqual({
      uid: '04E1A2B3C4D5E6',
      ctr: '000001',
      cmac: '0123456789ABCDEF'
    })
  })

  it('should reject data with incorrect length', () => {
    const wrapper = mount(Harness)
    const { parseSunMessage } = wrapper.vm

    const shortData = new Uint8Array(10).buffer
    const longData = new Uint8Array(20).buffer

    expect(parseSunMessage(shortData)).toBeNull()
    expect(parseSunMessage(longData)).toBeNull()
  })

  it('should correctly convert bytes to uppercase hex', () => {
    const wrapper = mount(Harness)
    const { parseSunMessage } = wrapper.vm

    const zeroData = new Uint8Array(18).fill(0).buffer
    const zeroResult = parseSunMessage(zeroData)

    expect(zeroResult).toEqual({
      uid: '00000000000000',
      ctr: '000000',
      cmac: '0000000000000000'
    })

    const maxData = new Uint8Array(18).fill(0xFF).buffer
    const maxResult = parseSunMessage(maxData)

    expect(maxResult).toEqual({
      uid: 'FFFFFFFFFFFFFF',
      ctr: 'FFFFFF',
      cmac: 'FFFFFFFFFFFFFFFF'
    })
  })

  it('should handle mixed case hex conversion', () => {
    const wrapper = mount(Harness)
    const { parseSunMessage } = wrapper.vm

    const mixedData = new Uint8Array([
      0x0A, 0x1B, 0x2C, 0x3D, 0x4E, 0x5F, 0x60,
      0x71, 0x82, 0x93,
      0xA4, 0xB5, 0xC6, 0xD7, 0xE8, 0xF9, 0x00, 0x11
    ]).buffer

    const result = parseSunMessage(mixedData)

    expect(result).toEqual({
      uid: '0A1B2C3D4E5F60',
      ctr: '718293',
      cmac: 'A4B5C6D7E8F90011'
    })
  })
})
