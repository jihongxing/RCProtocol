<template>
  <RcPageLayout title="资产激活">
    <div class="activate-container">
      <div class="form-card">
        <h3>资产激活</h3>
        <div class="flow-hint">
          <div class="flow-hint__title">最小运营闭环</div>
          <div class="flow-hint__steps">品牌创建 → API Key / 外部 SKU → 盲扫 → 激活 → 售出 → 审计</div>
        </div>

        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label for="asset-id">资产 ID <span class="required">*</span></label>
            <input
              id="asset-id"
              v-model="assetId"
              type="text"
              placeholder="输入资产 ID"
              required
              :disabled="flow.isRunning.value"
            />
          </div>

          <div class="form-group">
            <label for="external-product-id">外部 SKU</label>
            <input
              id="external-product-id"
              v-model="externalProductId"
              type="text"
              placeholder="可选"
              :disabled="flow.isRunning.value"
            />
          </div>

          <div class="form-group">
            <label for="external-product-name">外部 SKU 名称</label>
            <input
              id="external-product-name"
              v-model="externalProductName"
              type="text"
              placeholder="可选"
              :disabled="flow.isRunning.value"
            />
          </div>

          <div class="form-group">
            <label for="external-product-url">外部 SKU URL</label>
            <input
              id="external-product-url"
              v-model="externalProductUrl"
              type="url"
              placeholder="可选"
              :disabled="flow.isRunning.value"
            />
          </div>

          <div v-if="flow.currentStep.value !== 'idle'" class="step-indicator">
            <div class="step" :class="{ active: flow.currentStep.value === 'step1', completed: isStepCompleted(1) }">
              <div class="step-number">1</div>
              <div class="step-label">密钥轮换</div>
            </div>
            <div class="step-divider"></div>
            <div class="step" :class="{ active: flow.currentStep.value === 'step2', completed: isStepCompleted(2) }">
              <div class="step-number">2</div>
              <div class="step-label">绑定建立</div>
            </div>
            <div class="step-divider"></div>
            <div class="step" :class="{ active: flow.currentStep.value === 'step3', completed: isStepCompleted(3) }">
              <div class="step-number">3</div>
              <div class="step-label">激活确认</div>
            </div>
          </div>

          <button
            v-if="!flow.error.value"
            type="submit"
            class="submit-btn"
            :disabled="flow.isRunning.value || !assetId"
          >
            {{ flow.isRunning.value ? '执行中...' : '开始激活' }}
          </button>

          <button
            v-if="flow.error.value"
            type="button"
            class="retry-btn"
            @click="handleRetry"
            :disabled="flow.isRunning.value"
          >
            {{ flow.isRunning.value ? '重试中...' : '重试' }}
          </button>
        </form>

        <div v-if="flow.result.value" class="result success">
          <strong>激活成功</strong>
          <p>资产 {{ flow.result.value.asset_id }} 已成功激活</p>
          <p class="state-info">最终状态: {{ flow.result.value.final_state }}</p>
          <div v-if="flow.result.value.virtual_mother_card?.authority_uid" class="credential-card">
            <div class="credential-card__title">虚拟母卡已生成</div>
            <div class="credential-card__row">
              <span>母卡 UID</span>
              <span>{{ flow.result.value.virtual_mother_card?.authority_uid }}</span>
            </div>
            <div class="credential-card__row">
              <span>凭证哈希</span>
              <span class="credential-card__mono">{{ flow.result.value.virtual_mother_card?.credential_hash }}</span>
            </div>
            <div class="credential-card__hint">该凭证将用于 C 端虚拟母卡授权链路。</div>
          </div>
          <div class="result__actions">
            <button type="button" class="result__btn" @click="copyCredential">复制虚拟母卡凭证</button>
            <button type="button" class="result__btn" @click="goSell">继续去售出</button>
            <button type="button" class="result__btn result__btn--ghost" @click="goAudit">查看审计</button>
          </div>
        </div>

        <div v-if="flow.error.value" class="result error">
          <strong>步骤 {{ flow.error.value.step }} 失败</strong>
          <p>{{ flow.error.value.message }}</p>
          <p class="error-hint">点击“重试”按钮将从失败的步骤继续执行</p>
        </div>
      </div>
    </div>
  </RcPageLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { RcPageLayout } from '@rcprotocol/ui/web'
import { useTypedApi } from '../../composables/useTypedApi'
import { useActivationFlow } from '../../composables/useActivationFlow'

const route = useRoute()
const router = useRouter()
const { console: consoleApi } = useTypedApi()
const flow = useActivationFlow(consoleApi)

const assetId = ref('')
const externalProductId = ref('')
const externalProductName = ref('')
const externalProductUrl = ref('')

onMounted(() => {
  if (typeof route.query.assetId === 'string') {
    assetId.value = route.query.assetId
  }
})

const isStepCompleted = (step: number): boolean => {
  const currentStepMap: Record<string, number> = {
    idle: 0,
    step1: 1,
    step2: 2,
    step3: 3,
    completed: 4
  }

  const currentStepNum = currentStepMap[flow.currentStep.value] || 0
  return currentStepNum > step
}

function goSell() {
  const nextAssetId = flow.result.value?.asset_id || assetId.value.trim()
  if (!nextAssetId) return
  router.push({ path: '/sell', query: { assetId: nextAssetId } })
}

function goAudit() {
  const nextAssetId = flow.result.value?.asset_id || assetId.value.trim()
  if (!nextAssetId) return
  router.push({ path: '/audit', query: { resourceType: 'asset', resourceId: nextAssetId } })
}

async function copyCredential() {
  const credential = flow.result.value?.virtual_mother_card?.credential_hash
  if (!credential) return

  try {
    await navigator.clipboard.writeText(credential)
    window.alert('虚拟母卡凭证已复制，可用于 C 端调试/联调。')
  } catch {
    window.alert(`复制失败，请手动复制：${credential}`)
  }
}

const handleSubmit = async () => {
  if (!assetId.value.trim()) return

  await flow.execute({
    asset_id: assetId.value.trim(),
    external_product_id: externalProductId.value.trim() || undefined,
    external_product_name: externalProductName.value.trim() || undefined,
    external_product_url: externalProductUrl.value.trim() || undefined
  })
}

const handleRetry = async () => {
  await flow.execute({
    asset_id: assetId.value.trim(),
    external_product_id: externalProductId.value.trim() || undefined,
    external_product_name: externalProductName.value.trim() || undefined,
    external_product_url: externalProductUrl.value.trim() || undefined
  })
}
</script>

<style scoped>
.activate-container {
  max-width: 800px;
}

.form-card {
  background: white;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
}

.flow-hint {
  margin-bottom: 16px;
  padding: 14px 16px;
  border-radius: 10px;
  background: rgba(0, 113, 227, 0.08);
  border: 1px solid rgba(0, 113, 227, 0.12);
}

.flow-hint__title {
  font-size: 13px;
  font-weight: 700;
  color: #0071e3;
  margin-bottom: 4px;
}

.flow-hint__steps {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.66);
}

.form-group {
  margin-bottom: 16px;
}

label {
  display: block;
  margin-bottom: 8px;
  font-weight: 500;
  color: #374151;
}

.required {
  color: #dc2626;
}

input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
}

input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

input:disabled {
  background: #f3f4f6;
  cursor: not-allowed;
}

.step-indicator {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 24px 0;
  padding: 16px;
  background: #f9fafb;
  border-radius: 8px;
}

.step {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.step-number {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: #e5e7eb;
  color: #6b7280;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 16px;
  transition: all 0.3s;
}

.step.active .step-number {
  background: #3b82f6;
  color: white;
}

.step.completed .step-number {
  background: #10b981;
  color: white;
}

.step-label {
  font-size: 13px;
  color: #6b7280;
  font-weight: 500;
}

.step.active .step-label {
  color: #3b82f6;
}

.step.completed .step-label {
  color: #10b981;
}

.step-divider {
  flex: 0 0 40px;
  height: 2px;
  background: #e5e7eb;
  margin: 0 8px;
  margin-bottom: 24px;
}

.submit-btn,
.retry-btn {
  width: 100%;
  padding: 10px;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.submit-btn {
  background: #3b82f6;
}

.submit-btn:hover:not(:disabled) {
  background: #2563eb;
}

.submit-btn:disabled {
  background: #9ca3af;
  cursor: not-allowed;
}

.retry-btn {
  background: #f59e0b;
}

.retry-btn:hover:not(:disabled) {
  background: #d97706;
}

.retry-btn:disabled {
  background: #9ca3af;
  cursor: not-allowed;
}

.result {
  margin-top: 16px;
  padding: 12px;
  border-radius: 6px;
}

.result.success {
  background: #d1fae5;
  border: 1px solid #6ee7b7;
  color: #065f46;
}

.result.error {
  background: #fee2e2;
  border: 1px solid #fca5a5;
  color: #991b1b;
}

.result strong {
  display: block;
  margin-bottom: 4px;
}

.state-info,
.error-hint {
  margin-top: 4px;
  font-size: 13px;
}

.error-hint {
  font-style: italic;
}

.credential-card {
  margin-top: 12px;
  padding: 12px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(6, 95, 70, 0.16);
}

.credential-card__title {
  font-size: 13px;
  font-weight: 700;
  margin-bottom: 8px;
}

.credential-card__row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 12px;
  margin-bottom: 6px;
}

.credential-card__mono {
  font-family: monospace;
  word-break: break-all;
  text-align: right;
}

.credential-card__hint {
  font-size: 12px;
  color: rgba(6, 95, 70, 0.8);
}

.result__actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  flex-wrap: wrap;
}

.result__btn {
  padding: 8px 12px;
  border: none;
  border-radius: 6px;
  background: #059669;
  color: #fff;
  cursor: pointer;
}

.result__btn--ghost {
  background: transparent;
  color: #065f46;
  border: 1px solid #6ee7b7;
}
</style>
