import { ref } from 'vue'
import type { UseApiReturn } from './useTypedApi'

type Step = 'idle' | 'step1' | 'step2' | 'step3' | 'completed'

interface FlowError {
  step: number
  message: string
  code?: string
}

interface ActivationResult {
  asset_id: string
  final_state: string
  virtual_mother_card?: {
    authority_uid?: string
    authority_type?: string
    credential_hash?: string
    epoch?: number
  }
}

interface ActivationPayload {
  asset_id: string
  external_product_id?: string
  external_product_name?: string
  external_product_url?: string
}

/**
 * Composable for managing 3-step activation flow with independent idempotency keys per step.
 * Treats 409 as "step already done" and proceeds to next step.
 * Failed step records error.step for retry resumption.
 */
export function useActivationFlow(api: UseApiReturn) {
  const currentStep = ref<Step>('idle')
  const isRunning = ref(false)
  const error = ref<FlowError | null>(null)
  const result = ref<ActivationResult | null>(null)

  const generateStepKey = () => crypto.randomUUID()

  const execute = async (payload: ActivationPayload) => {
    isRunning.value = true
    error.value = null
    result.value = null

    try {
      const startStep = currentStep.value === 'idle' ? 1 :
                        currentStep.value === 'step1' ? 1 :
                        currentStep.value === 'step2' ? 2 :
                        currentStep.value === 'step3' ? 3 : 1

      if (startStep <= 1) {
        currentStep.value = 'step1'
        const step1Key = generateStepKey()

        try {
          const response = await api.activateAsset(payload.asset_id, {
            external_product_id: payload.external_product_id,
            external_product_name: payload.external_product_name,
            external_product_url: payload.external_product_url
          }, {
            'X-Idempotency-Key': step1Key
          }) as { virtual_mother_card?: ActivationResult['virtual_mother_card'] }

          if (response.virtual_mother_card) {
            result.value = {
              asset_id: payload.asset_id,
              final_state: 'RotatingKeys',
              virtual_mother_card: response.virtual_mother_card
            }
          }
        } catch (err: any) {
          if (err.code !== 'CONFLICT') {
            throw { step: 1, message: err.message || '激活失败', code: err.code }
          }
        }
      }

      if (startStep <= 2) {
        currentStep.value = 'step2'
        const step2Key = generateStepKey()

        try {
          await api.entangleAsset(payload.asset_id, {
            'X-Idempotency-Key': step2Key
          })
        } catch (err: any) {
          if (err.code !== 'CONFLICT') {
            throw { step: 2, message: err.message || '绑定失败', code: err.code }
          }
        }
      }

      if (startStep <= 3) {
        currentStep.value = 'step3'
        const step3Key = generateStepKey()

        try {
          const response = await api.confirmAssetActivation(payload.asset_id, {
            'X-Idempotency-Key': step3Key
          })

          result.value = {
            asset_id: payload.asset_id,
            final_state: (response as { to_state?: string }).to_state || 'Activated',
            virtual_mother_card: result.value?.virtual_mother_card,
          }
        } catch (err: any) {
          if (err.code !== 'CONFLICT') {
            throw { step: 3, message: err.message || '确认失败', code: err.code }
          }
          result.value = {
            asset_id: payload.asset_id,
            final_state: 'Activated',
            virtual_mother_card: result.value?.virtual_mother_card,
          }
        }
      }

      currentStep.value = 'completed'
    } catch (err: any) {
      error.value = err as FlowError
    } finally {
      isRunning.value = false
    }
  }

  const reset = () => {
    currentStep.value = 'idle'
    isRunning.value = false
    error.value = null
    result.value = null
  }

  return {
    currentStep,
    isRunning,
    error,
    result,
    execute,
    reset
  }
}
