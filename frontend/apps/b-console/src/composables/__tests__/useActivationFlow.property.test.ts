import { describe, it, expect } from 'vitest'
import { fc } from '@fast-check/vitest'
import { useActivationFlow } from '../useActivationFlow'
import type { UseApiReturn } from '../useTypedApi'

type ActivationApiMock = Pick<UseApiReturn, 'activateAsset' | 'entangleAsset' | 'confirmAssetActivation'>

describe('useActivationFlow - Property Tests', () => {
  it('Property 5: For any step S ∈ {1,2,3} where API fails, error.step === S, execution stops, and retry resumes from S', () => {
    fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 1, max: 3 }),
        fc.string({ minLength: 1, maxLength: 50 }),
        async (failingStep, errorMessage) => {
          const mockApi = {
            activateAsset: async () => {
              if (failingStep === 1) {
                throw { code: 'TEST_ERROR', message: errorMessage }
              }
              return { to_state: 'Activated' }
            },
            entangleAsset: async () => {
              if (failingStep === 2) {
                throw { code: 'TEST_ERROR', message: errorMessage }
              }
              return { to_state: 'Activated' }
            },
            confirmAssetActivation: async () => {
              if (failingStep === 3) {
                throw { code: 'TEST_ERROR', message: errorMessage }
              }
              return { to_state: 'Activated' }
            }
          } as ActivationApiMock as UseApiReturn

          const flow = useActivationFlow(mockApi)

          await flow.execute({
            asset_id: 'test-asset'
          })

          expect(flow.error.value).not.toBeNull()
          expect(flow.error.value?.step).toBe(failingStep)
          expect(flow.error.value?.message).toContain(errorMessage)

          const expectedStep = (['step1', 'step2', 'step3'] as const)[failingStep - 1]
          expect(flow.currentStep.value).toBe(expectedStep)

          const callCount = { activate: 0, entangle: 0, confirm: 0 }
          const retryMockApi = {
            activateAsset: async () => {
              callCount.activate++
              return { to_state: 'Activated' }
            },
            entangleAsset: async () => {
              callCount.entangle++
              return { to_state: 'Activated' }
            },
            confirmAssetActivation: async () => {
              callCount.confirm++
              return { to_state: 'Activated' }
            }
          } as ActivationApiMock as UseApiReturn

          const retryFlow = useActivationFlow(retryMockApi)
          retryFlow.currentStep.value = expectedStep
          await retryFlow.execute({ asset_id: 'test-asset' })

          if (failingStep === 2) {
            expect(callCount.activate).toBe(0)
          }
          if (failingStep === 3) {
            expect(callCount.activate).toBe(0)
            expect(callCount.entangle).toBe(0)
          }
        }
      ),
      { numRuns: 100 }
    )
  })
})
