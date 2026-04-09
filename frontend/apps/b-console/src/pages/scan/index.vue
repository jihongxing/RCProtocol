<template>
  <RcPageLayout title="盲扫任务">
    <div class="scan-container">
      <div class="form-card">
        <h3>扫描操作</h3>
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
            <label for="operation-type">操作类型</label>
            <select id="operation-type" v-model="operationType" required>
              <option value="blind-log">盲扫登记</option>
              <option value="stock-in">入库</option>
            </select>
          </div>

          <button
            type="submit"
            class="submit-btn"
            :disabled="isLoading || !assetId"
          >
            {{ isLoading ? '提交中...' : '提交' }}
          </button>
        </form>

        <div v-if="result" class="result success">
          <strong>操作成功</strong>
          <p>{{ result.from_state }} → {{ result.to_state }}</p>
          <div class="result__actions">
            <button v-if="operationType === 'blind-log'" type="button" class="result__btn" @click="goActivate">
              继续去激活
            </button>
            <button type="button" class="result__btn result__btn--ghost" @click="goAudit">
              查看审计
            </button>
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
import { ref, onMounted } from 'vue'
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
const operationType = ref<'blind-log' | 'stock-in'>('blind-log')
const isLoading = ref(false)
const result = ref<{ from_state: string; to_state: string } | null>(null)
const errorMessage = ref('')
const lastSuccessAssetId = ref('')

const errorCodeMap: Record<string, string> = {
  CONFLICT: '该操作已提交',
  FORBIDDEN: '权限不足',
  UNPROCESSABLE: '当前状态不允许此操作',
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

function goActivate() {
  const nextAssetId = lastSuccessAssetId.value || assetId.value.trim()
  if (!nextAssetId) return
  router.push({ path: '/activate', query: { assetId: nextAssetId } })
}

function goAudit() {
  const nextAssetId = lastSuccessAssetId.value || assetId.value.trim()
  if (!nextAssetId) return
  router.push({ path: '/audit', query: { resourceType: 'asset', resourceId: nextAssetId } })
}

const handleSubmit = async () => {
  if (!assetId.value.trim()) return

  isLoading.value = true
  result.value = null
  errorMessage.value = ''

  const currentAssetId = assetId.value.trim()
  const actionName = operationType.value === 'blind-log' ? '盲扫登记' : '入库'

  try {
    const response = operationType.value === 'blind-log'
      ? await consoleApi.blindLogAsset(currentAssetId, { 'X-Idempotency-Key': idempotency.key.value })
      : await consoleApi.stockInAsset(currentAssetId, { 'X-Idempotency-Key': idempotency.key.value })

    const stateResponse = response as { from_state?: string; to_state?: string }
    lastSuccessAssetId.value = currentAssetId
    result.value = {
      from_state: stateResponse.from_state || '-',
      to_state: stateResponse.to_state || '-'
    }

    const logEntry: OperationLogEntry = {
      id: crypto.randomUUID(),
      asset_id: currentAssetId,
      action: actionName,
      from_state: stateResponse.from_state || '-',
      to_state: stateResponse.to_state || '-',
      timestamp: new Date().toISOString(),
      success: true
    }
    operationLog.append(logEntry)

    idempotency.regenerate()
    assetId.value = ''
  } catch (err: any) {
    errorMessage.value = getErrorMessage(err.code)

    const logEntry: OperationLogEntry = {
      id: crypto.randomUUID(),
      asset_id: currentAssetId,
      action: actionName,
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
.scan-container {
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

input,
select {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
}

input:focus,
select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
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

.result__btn--ghost {
  background: transparent;
  color: #065f46;
  border: 1px solid #6ee7b7;
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
