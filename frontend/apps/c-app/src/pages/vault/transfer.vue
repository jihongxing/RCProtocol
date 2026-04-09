<template>
  <RcPageLayout title="发起转让" showBack>
    <view v-if="assetInfo" class="transfer-summary">
      <view class="transfer-summary__row">
        <text class="transfer-summary__label">品牌</text>
        <text class="transfer-summary__value">{{ assetInfo.brand_name }}</text>
      </view>
      <view class="transfer-summary__row">
        <text class="transfer-summary__label">资产</text>
        <text class="transfer-summary__value">{{ assetInfo.product_name }}</text>
      </view>
      <view class="transfer-summary__row transfer-summary__row--last">
        <text class="transfer-summary__label">状态</text>
        <RcStatusBadge :status="assetInfo.state" />
      </view>
    </view>

    <view v-if="authorityMode === 'VirtualToken' && virtualCredential" class="credential-banner">
      <text class="credential-banner__title">已检测到虚拟母卡凭证</text>
      <text class="credential-banner__desc">当前设备已保存该资产的虚拟母卡凭证，可直接完成虚拟授权转让。</text>
    </view>

    <view class="transfer-form">
      <view class="transfer-form__field">
        <text class="transfer-form__label">接收方</text>
        <input
          v-model="targetUserId"
          placeholder="请输入接收方用户 ID 或邮箱"
          class="transfer-form__input"
        />
      </view>

      <view class="transfer-form__field">
        <text class="transfer-form__label">授权方式</text>
        <view class="authority-mode">
          <view
            class="authority-mode__option"
            :class="{ 'authority-mode__option--active': authorityMode === 'VirtualToken' }"
            @tap="authorityMode = 'VirtualToken'"
          >
            <view class="authority-mode__radio">
              <view v-if="authorityMode === 'VirtualToken'" class="authority-mode__radio-inner" />
            </view>
            <view class="authority-mode__content">
              <view class="authority-mode__title-row">
                <text class="authority-mode__title">虚拟母卡授权</text>
                <text class="authority-mode__badge">推荐</text>
              </view>
              <text class="authority-mode__desc">使用已激活的虚拟母卡凭证进行授权</text>
            </view>
          </view>
          <view
            class="authority-mode__option"
            :class="{ 'authority-mode__option--active': authorityMode === 'PhysicalNfc' }"
            @tap="authorityMode = 'PhysicalNfc'"
          >
            <view class="authority-mode__radio">
              <view v-if="authorityMode === 'PhysicalNfc'" class="authority-mode__radio-inner" />
            </view>
            <view class="authority-mode__content">
              <text class="authority-mode__title">物理母卡授权</text>
              <text class="authority-mode__desc">使用物理母卡 NFC 扫描进行授权</text>
            </view>
          </view>
        </view>
      </view>

      <RcRiskCard
        v-if="authorityMode === 'VirtualToken' && !virtualCredential"
        level="risk"
        message="虚拟母卡凭证不存在，请先在资产详情中保存虚拟母卡凭证"
      />

      <view v-if="authorityMode === 'PhysicalNfc'" class="authority-nfc-status">
        <view v-if="nfcState.scanning" class="nfc-status nfc-status--scanning">
          <text class="nfc-status__icon">📡</text>
          <text class="nfc-status__text">正在扫描物理母卡...</text>
        </view>
        <view v-else-if="nfcState.result" class="nfc-status nfc-status--success">
          <text class="nfc-status__icon">✅</text>
          <view class="nfc-status__content">
            <text class="nfc-status__text">母卡扫描成功</text>
            <text class="nfc-status__detail">UID: ...{{ nfcState.result.uid.slice(-4) }}</text>
          </view>
        </view>
        <view v-else-if="nfcState.error" class="nfc-status nfc-status--error">
          <text class="nfc-status__icon">⚠️</text>
          <text class="nfc-status__text">{{ getNfcErrorMessage(nfcState.error) }}</text>
        </view>
        <view v-else class="nfc-status nfc-status--idle">
          <text class="nfc-status__icon">📡</text>
          <text class="nfc-status__text">请将物理母卡贴近手机 NFC 感应区</text>
        </view>
      </view>

      <RcRiskCard v-if="errorMsg" level="risk" :message="errorMsg" />

      <button
        class="transfer-form__btn"
        :disabled="!canSubmit"
        @tap="confirmTransfer"
      >
        {{ submitting ? '提交中...' : '确认转让' }}
      </button>
    </view>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'CAppTransferPage' })

import { ref, computed, onMounted, watch } from 'vue'
import { RcPageLayout, RcStatusBadge, RcRiskCard } from '@rcprotocol/ui/uni'
import type { AssetSummary, InitiateTransferRequest } from '@rcprotocol/api'
import { useAuth } from '@rcprotocol/state'
import { useTypedApi } from '../../composables/useTypedApi'
import { generateIdempotencyKey } from '../../composables/useIdempotency'
import { useNfcReader } from '../../composables/useNfcReader'
import { buildTransferPayload as buildTransferPayloadValue, getNfcErrorMessage, resolveTransferError, type AuthorityMode, type TransferPageError } from './transfer.logic'

const { app: appApi, transfer: transferApi } = useTypedApi()
const { state, startScan, stopScan, reset: resetNfc } = useNfcReader()
const { user } = useAuth()

const assetId = ref('')
const assetInfo = ref<AssetSummary | null>(null)
const targetUserId = ref('')
const submitting = ref(false)
const errorMsg = ref('')
const authorityMode = ref<AuthorityMode>('VirtualToken')

const nfcState = computed(() => state.value)
const virtualCredential = computed<string | null>(() => {
  if (!assetId.value) return null
  try {
    return uni.getStorageSync(`rc_virtual_credential_${assetId.value}`) || null
  } catch {
    return null
  }
})

const canSubmit = computed(() => {
  if (submitting.value) return false
  if (!targetUserId.value.trim()) return false
  if (authorityMode.value === 'VirtualToken' && (!virtualCredential.value || !user.value?.user_id)) return false
  if (authorityMode.value === 'PhysicalNfc' && (!nfcState.value.result || nfcState.value.scanning)) return false
  return true
})

watch(authorityMode, (newMode, oldMode) => {
  if (newMode === 'PhysicalNfc' && oldMode !== 'PhysicalNfc') {
    startScan()
  } else if (newMode === 'VirtualToken' && oldMode === 'PhysicalNfc') {
    stopScan()
    resetNfc()
  }
})

onMounted(async () => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as { options?: { assetId?: string } }
  assetId.value = currentPage?.options?.assetId || ''

  if (assetId.value) {
    try {
      const detail = await appApi.getMyAsset(assetId.value)
      assetInfo.value = {
        asset_id: detail.asset_id,
        state: detail.state,
        state_label: detail.state_label,
        brand_name: detail.brand_name || detail.brand_id,
        product_name: detail.product_name || detail.product_id,
        external_product_name: detail.external_product_name,
      }
    } catch {
      // ignore summary load failure
    }
  }
})

function buildTransferPayload(): InitiateTransferRequest {
  return buildTransferPayloadValue({
    authorityMode: authorityMode.value,
    targetUserId: targetUserId.value,
    assetId: assetId.value,
    userId: user.value?.user_id || '',
    virtualCredential: virtualCredential.value || '',
    nfcResult: nfcState.value.result,
  })
}

function confirmTransfer() {
  if (!canSubmit.value) return

  uni.showModal({
    title: '确认转让',
    content: `确认将此资产转让给 ${targetUserId.value.trim()}？此操作不可撤销。`,
    confirmColor: '#ff3b30',
    success: (res) => {
      if (res.confirm) {
        doTransfer()
      }
    }
  })
}

async function doTransfer() {
  submitting.value = true
  errorMsg.value = ''
  try {
    const transfer = await transferApi.initiate(assetId.value, buildTransferPayload(), {
      'X-Idempotency-Key': generateIdempotencyKey(),
    })

    uni.showToast({ title: '转让请求已创建', icon: 'success' })
    setTimeout(() => {
      uni.navigateTo({ url: `/pages/vault/transfer-confirm?transferId=${transfer.transfer_id}` })
    }, 500)
  } catch (error: unknown) {
    const resolved = resolveTransferError(error as TransferPageError)
    errorMsg.value = resolved.message
    if (resolved.resetNfc) {
      resetNfc()
    }
    if (resolved.restartScan && authorityMode.value === 'PhysicalNfc') {
      startScan()
    }
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.transfer-summary {
  background-color: #ffffff;
  border-radius: 24rpx;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  padding: 0 32rpx;
  margin-bottom: 32rpx;
}

.transfer-summary__row {
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
  padding: 28rpx 0;
  border-bottom: 1rpx solid rgba(0, 0, 0, 0.08);
}

.transfer-summary__row--last {
  border-bottom: none;
}

.transfer-summary__label {
  font-size: 28rpx;
  color: rgba(0, 0, 0, 0.65);
  flex-shrink: 0;
  margin-right: 24rpx;
}

.transfer-summary__value {
  font-size: 28rpx;
  color: #1d1d1f;
  text-align: right;
  word-break: break-all;
}

.credential-banner {
  margin-bottom: 24rpx;
  padding: 24rpx 28rpx;
  border-radius: 20rpx;
  background: rgba(0, 113, 227, 0.08);
  border: 1rpx solid rgba(0, 113, 227, 0.14);
}

.credential-banner__title {
  display: block;
  font-size: 28rpx;
  font-weight: 600;
  color: #0071e3;
  margin-bottom: 8rpx;
}

.credential-banner__desc {
  display: block;
  font-size: 24rpx;
  color: rgba(0, 0, 0, 0.66);
  line-height: 1.6;
}
</style>
