import { ref, onUnmounted } from 'vue'

export interface NfcScanResult {
  uid: string
  ctr: string
  cmac: string
}

export interface NfcReaderState {
  scanning: boolean
  result: NfcScanResult | null
  error: string | null
}

type HceState = {
  available: boolean
  enabled: boolean
}

type HceMessage = {
  data: ArrayBuffer
}

declare const uni: {
  getHCEState(): Promise<HceState>
  startHCE(options: { aid_list: string[] }): Promise<void>
  onHCEMessage(listener: (res: HceMessage) => void): void
  offHCEMessage(listener: (res: HceMessage) => void): void
  stopHCE(): void
}

export function useNfcReader() {
  const state = ref<NfcReaderState>({
    scanning: false,
    result: null,
    error: null
  })

  let timeoutId: number | null = null
  let messageListener: ((res: HceMessage) => void) | null = null

  function parseSunMessage(data: ArrayBuffer): NfcScanResult | null {
    const bytes = new Uint8Array(data)

    if (bytes.length !== 18) {
      console.error('[NFC] Invalid data length:', bytes.length)
      return null
    }

    const uid = Array.from(bytes.slice(0, 7))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('')
      .toUpperCase()

    const ctr = Array.from(bytes.slice(7, 10))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('')
      .toUpperCase()

    const cmac = Array.from(bytes.slice(10, 18))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('')
      .toUpperCase()

    return { uid, ctr, cmac }
  }

  async function startScan() {
    try {
      const hceState = await uni.getHCEState()

      if (!hceState.available) {
        state.value.error = 'NFC_NOT_SUPPORTED'
        return
      }

      if (!hceState.enabled) {
        state.value.error = 'NFC_DISABLED'
        return
      }

      await uni.startHCE({
        aid_list: ['F0010203040506']
      })

      state.value.scanning = true
      state.value.error = null

      timeoutId = setTimeout(() => {
        state.value.error = 'SCAN_TIMEOUT'
        stopScan()
      }, 30000) as unknown as number

      messageListener = (res: HceMessage) => {
        console.log('[NFC] Received message:', res)

        const result = parseSunMessage(res.data)

        if (result) {
          state.value.result = result
          state.value.scanning = false
          stopScan()
        } else if (!state.value.error) {
          state.value.error = 'MALFORMED_DATA'
          console.warn('[NFC] Malformed data, waiting for retry...')
        }
      }

      uni.onHCEMessage(messageListener)
    } catch (err: unknown) {
      console.error('[NFC] Start scan failed:', err)
      const message = err && typeof err === 'object' && 'errMsg' in err ? String((err as { errMsg?: string }).errMsg) : 'UNKNOWN_ERROR'
      state.value.error = message || 'UNKNOWN_ERROR'
      state.value.scanning = false
    }
  }

  function stopScan() {
    if (timeoutId !== null) {
      clearTimeout(timeoutId)
      timeoutId = null
    }

    if (messageListener) {
      uni.offHCEMessage(messageListener)
      messageListener = null
    }

    try {
      uni.stopHCE()
    } catch (err) {
      console.warn('[NFC] Stop HCE failed:', err)
    }

    state.value.scanning = false
  }

  function reset() {
    state.value.result = null
    state.value.error = null
  }

  onUnmounted(() => {
    stopScan()
  })

  return {
    state,
    startScan,
    stopScan,
    reset,
    parseSunMessage
  }
}
