<template>
  <RcPageLayout title="确认接收" showBack>
    <RcLoadingState :loading="loading" :error="loadError" @retry="loadTransfer" />

    <view v-if="!loading && expired" class="confirm-expired">
      <RcRiskCard level="info" message="该转让请求已过期或已处理" />
    </view>

    <view v-if="!loading && !loadError && transferInfo" class="confirm">
      <view class="confirm-card">
        <view class="confirm-card__row">
          <text class="confirm-card__label">发起方</text>
          <text class="confirm-card__value">{{ transferInfo.from_user_id }}</text>
        </view>
        <view class="confirm-card__row">
          <text class="confirm-card__label">资产 ID</text>
          <text class="confirm-card__value">{{ transferInfo.asset_id }}</text>
        </view>
        <view v-if="transferInfo.asset_summary?.brand_name" class="confirm-card__row">
          <text class="confirm-card__label">品牌</text>
          <text class="confirm-card__value">{{ transferInfo.asset_summary.brand_name }}</text>
        </view>
        <view v-if="transferInfo.asset_summary?.product_name || transferInfo.asset_summary?.external_product_name" class="confirm-card__row">
          <text class="confirm-card__label">资产</text>
          <text class="confirm-card__value">{{ transferInfo.asset_summary?.product_name || transferInfo.asset_summary?.external_product_name }}</text>
        </view>
        <view class="confirm-card__row confirm-card__row--last">
          <text class="confirm-card__label">发起时间</text>
          <text class="confirm-card__value">{{ formatDate(transferInfo.created_at) }}</text>
        </view>
      </view>

      <RcRiskCard v-if="errorMsg" level="risk" :message="errorMsg" />

      <view class="confirm-actions">
        <button
          class="confirm-actions__btn confirm-actions__btn--primary"
          :disabled="submitting"
          @tap="doConfirm"
        >
          {{ submitting ? '处理中...' : '确认接收' }}
        </button>
        <button
          class="confirm-actions__btn confirm-actions__btn--secondary"
          :disabled="submitting"
          @tap="doReject"
        >
          {{ submitting ? '处理中...' : '拒绝' }}
        </button>
      </view>
    </view>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'CAppTransferConfirmPage' })

import { ref, onMounted } from 'vue'
import { RcPageLayout, RcLoadingState, RcRiskCard } from '@rcprotocol/ui/uni'
import { formatDate } from '@rcprotocol/utils'
import type { TransferInfo } from '@rcprotocol/utils'
import { useTypedApi } from '../../composables/useTypedApi'
import { generateIdempotencyKey } from '../../composables/useIdempotency'
import { normalizeTransferInfo, resolveTransferActionError, resolveTransferLoadError } from './transfer-confirm.logic'

interface TransferConfirmError {
  status?: number
  code?: string
  message?: string
}

const { transfer: transferApi } = useTypedApi()

const transferId = ref('')
const transferInfo = ref<TransferInfo | null>(null)
const loading = ref(false)
const loadError = ref('')
const submitting = ref(false)
const errorMsg = ref('')
const expired = ref(false)

onMounted(() => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as { options?: { transferId?: string } }
  transferId.value = currentPage?.options?.transferId || ''
  if (transferId.value) {
    loadTransfer()
  }
})

async function loadTransfer() {
  loading.value = true
  loadError.value = ''
  expired.value = false
  try {
    const normalized = normalizeTransferInfo(await transferApi.getTransfer(transferId.value))
    expired.value = normalized.expired
    transferInfo.value = normalized.transferInfo
  } catch (error: unknown) {
    const resolved = resolveTransferLoadError(error as TransferConfirmError)
    expired.value = resolved.expired
    loadError.value = resolved.loadError
  } finally {
    loading.value = false
  }
}

async function doConfirm() {
  if (!transferInfo.value) return
  submitting.value = true
  errorMsg.value = ''
  try {
    await transferApi.confirm({ transfer_id: transferId.value }, {
      'X-Idempotency-Key': generateIdempotencyKey()
    })
    uni.showToast({ title: '接收成功', icon: 'success' })
    setTimeout(() => {
      uni.switchTab({ url: '/pages/vault/index' })
    }, 1000)
  } catch (error: unknown) {
    handleActionError(error, '确认失败')
  } finally {
    submitting.value = false
  }
}

async function doReject() {
  if (!transferInfo.value) return
  submitting.value = true
  errorMsg.value = ''
  try {
    await transferApi.reject({ transfer_id: transferId.value }, {
      'X-Idempotency-Key': generateIdempotencyKey()
    })
    uni.showToast({ title: '已拒绝', icon: 'none' })
    setTimeout(() => {
      uni.navigateBack()
    }, 600)
  } catch (error: unknown) {
    handleActionError(error, '拒绝失败')
  } finally {
    submitting.value = false
  }
}

function handleActionError(error: unknown, fallbackMessage: string) {
  const resolved = resolveTransferActionError(error as TransferConfirmError, fallbackMessage)
  errorMsg.value = resolved.message
  expired.value = resolved.expired
  if (resolved.clearTransferInfo) {
    transferInfo.value = null
  }
}
</script>

<style scoped>
.confirm-expired {
  padding: 48rpx 0;
}

.confirm {
  display: flex;
  flex-direction: column;
  gap: 32rpx;
}

.confirm-card {
  background-color: #ffffff;
  border-radius: 24rpx;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  padding: 0 32rpx;
}

.confirm-card__row {
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
  padding: 28rpx 0;
  border-bottom: 1rpx solid rgba(0, 0, 0, 0.08);
}

.confirm-card__row--last {
  border-bottom: none;
}

.confirm-card__label {
  font-size: 28rpx;
  color: rgba(0, 0, 0, 0.65);
  flex-shrink: 0;
  margin-right: 24rpx;
}

.confirm-card__value {
  font-size: 28rpx;
  color: #1d1d1f;
  text-align: right;
  word-break: break-all;
}
</style>
