<template>
  <RcPageLayout title="售出确认">
    <div class="sell-container">
      <div class="form-card">
        <h3>售出操作</h3>
        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label for="asset-id">资产 ID</label>
            <input
              id="asset-id"
              v-model="assetId"
              type="text"
              placeholder="输入资产 ID"
              required
            />
          </div>

          <div class="form-group">
            <label for="buyer-id">买家 ID</label>
            <input
              id="buyer-id"
              v-model="buyerId"
              type="text"
              placeholder="输入买家 ID"
              required
              @blur="validateBuyerId"
            />
            <span v-if="buyerIdError" class="error-text">{{ buyerIdError }}</span>
          </div>

          <button
            type="submit"
            class="submit-btn"
            :disabled="isLoading || !assetId || !buyerId || !!buyerIdError"
          >
            {{ isLoading ? '提交中...' : '确认售出' }}
          </button>
        </form>

        <div v-if="result" class="result success">
          <strong>售出成功</strong>
          <p>{{ result.from_state }} → {{ result.to_state }}</p>
          <p class="buyer-info">买家: {{ result.buyer_id }}</p>
          <div class="result__actions">
            <button type="button" class="result__btn" @click="goAudit">查看审计</button>
          </div>
        </div>

        <div v-if="errorMessage" class="result error">
          <strong>操作失败</strong>
          <p>{{ errorMessage }}</p>
        </div>
      </div>

      <div class="log-card">
        <h3>操作记录（本次会话）</h3>
        <div v-if="operationLog.logs.value.length === 0" class="empty-log">
          暂无操作记录
        </div>
        <table v-else class="log-table">
          <thead>
            <tr>
              <th>资产 ID</th>
              <th>操作</th>
              <th>状态变更</th>
              <th>时间</th>
              <th>结果</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in operationLog.logs.value" :key="log.id">
              <td>{{ log.asset_id }}</td>
              <td>{{ log.action }}</td>
              <td>{{ log.from_state }} → {{ log.to_state }}</td>
              <td>{{ formatTime(log.timestamp) }}</td>
              <td>
                <span :class="log.success ? 'status-success' : 'status-fail'">
                  {{ log.success ? '成功' : '失败' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </RcPageLayout>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { RcPageLayout } from '@rcprotocol/ui/web'
import { useTypedApi } from '../../composables/useTypedApi'
import { useIdempotency } from '../../composables/useIdempotency'
import { useOperationLog } from '../../composables/useOperationLog'
import type { OperationLogEntry } from '../../composables/useOperationLog'

const route = useRoute()
const router = useRouter()
const { console: consoleApi } = useTypedApi()
const idempotency = useIdempotency()
const operationLog = useOperationLog(20)

const assetId = ref('')
const buyerId = ref('')
const buyerIdError = ref('')
const isLoading = ref(false)
const result = ref<{ from_state: string; to_state: string; buyer_id: string } | null>(null)
const errorMessage = ref('')
const lastSuccessAssetId = ref('')

const errorCodeMap: Record<string, string> = {
  CONFLICT: '该操作已提交',
  UNPROCESSABLE: '仅已激活资产可售出',
  NETWORK_ERROR: '网络连接失败'
}

const getErrorMessage = (code?: string): string => {
  return code ? errorCodeMap[code] || '操作失败，请稍后重试' : '操作失败，请稍后重试'
}

const formatTime = (timestamp: string): string => {
  return new Date(timestamp).toLocaleTimeString('zh-CN')
}

onMounted(() => {
  if (typeof route.query.assetId === 'string') {
    assetId.value = route.query.assetId
  }
})

function goAudit() {
  const nextAssetId = lastSuccessAssetId.value || assetId.value.trim()
  if (!nextAssetId) return
  router.push({ path: '/audit', query: { resourceType: 'asset', resourceId: nextAssetId } })
}

const validateBuyerId = () => {
  const trimmed = buyerId.value.trim()
  if (buyerId.value && !trimmed) {
    buyerIdError.value = '买家 ID 不能为空或仅包含空格'
  } else {
    buyerIdError.value = ''
  }
}

watch(buyerId, () => {
  if (buyerIdError.value) {
    validateBuyerId()
  }
})

const handleSubmit = async () => {
  validateBuyerId()
  if (buyerIdError.value || !assetId.value.trim() || !buyerId.value.trim()) {
    return
  }

  isLoading.value = true
  result.value = null
  errorMessage.value = ''

  const currentAssetId = assetId.value.trim()
  const currentBuyerId = buyerId.value.trim()

  try {
    const response = await consoleApi.legalSellAsset(
      currentAssetId,
      { buyer_id: currentBuyerId },
      { 'X-Idempotency-Key': idempotency.key.value }
    ) as { from_state?: string; to_state?: string }

    lastSuccessAssetId.value = currentAssetId
    result.value = {
      from_state: response.from_state || 'Activated',
      to_state: response.to_state || 'LegallySold',
      buyer_id: currentBuyerId
    }

    const logEntry: OperationLogEntry = {
      id: crypto.randomUUID(),
      asset_id: currentAssetId,
      action: '售出确认',
      from_state: response.from_state || 'Activated',
      to_state: response.to_state || 'LegallySold',
      timestamp: new Date().toISOString(),
      success: true
    }
    operationLog.append(logEntry)

    idempotency.regenerate()
    assetId.value = ''
    buyerId.value = ''
  } catch (err: any) {
    errorMessage.value = getErrorMessage(err.code)

    const logEntry: OperationLogEntry = {
      id: crypto.randomUUID(),
      asset_id: currentAssetId,
      action: '售出确认',
      from_state: '-',
      to_state: '-',
      timestamp: new Date().toISOString(),
      success: false
    }
    operationLog.append(logEntry)
  } finally {
    isLoading.value = false
  }
}
</script>

<style scoped>
.sell-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-width: 1200px;
}

.form-card,
.log-card {
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

.form-group {
  margin-bottom: 16px;
}

label {
  display: block;
  margin-bottom: 8px;
  font-weight: 500;
  color: #374151;
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

.error-text {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: #dc2626;
}

.submit-btn {
  width: 100%;
  padding: 10px;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.submit-btn:hover:not(:disabled) {
  background: #2563eb;
}

.submit-btn:disabled {
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

.buyer-info {
  margin-top: 4px;
  font-size: 13px;
}

.result__actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.result__btn {
  padding: 8px 12px;
  border: none;
  border-radius: 6px;
  background: #059669;
  color: #fff;
  cursor: pointer;
}

.empty-log {
  text-align: center;
  color: #6b7280;
  padding: 32px;
}

.log-table {
  width: 100%;
  border-collapse: collapse;
}

.log-table th,
.log-table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #e5e7eb;
}

.log-table th {
  background: #f9fafb;
  font-weight: 600;
  color: #374151;
}

.log-table tbody tr:hover {
  background: #f9fafb;
}

.status-success {
  color: #059669;
  font-weight: 500;
}

.status-fail {
  color: #dc2626;
  font-weight: 500;
}
</style>
