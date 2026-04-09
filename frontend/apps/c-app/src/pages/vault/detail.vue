<template>
  <RcPageLayout :title="detail?.brand_name || '资产详情'" showBack>
    <RcLoadingState :loading="loading" :error="errorMsg" @retry="loadDetail" />

    <view v-if="!loading && notFound" class="not-found">
      <text class="not-found__icon">🔍</text>
      <text class="not-found__text">资产不存在</text>
    </view>

    <view v-if="!loading && !errorMsg && detail" class="asset-detail">
      <view v-if="isHonorState" class="honor-banner" :class="`honor-banner--${detail.state}`">
        <text class="honor-banner__icon">{{ honorIcon }}</text>
        <text class="honor-banner__title">{{ honorTitle }}</text>
        <text class="honor-banner__desc">{{ honorDesc }}</text>
      </view>

      <view v-else class="asset-detail__status">
        <view class="asset-detail__status-badge">
          <RcStatusBadge :status="detail.state" />
        </view>
        <text class="asset-detail__state-label">{{ detail.state_label }}</text>
      </view>

      <RcRiskCard
        v-if="detail.state === 'Disputed'"
        level="risk"
        message="该资产当前处于争议冻结状态"
      />

      <view v-if="detail.display_badges?.length" class="asset-detail__badges">
        <text
          v-for="badge in detail.display_badges"
          :key="badge"
          class="badge-pill"
        >
          {{ badgeLabel(badge) }}
        </text>
      </view>

      <view class="asset-detail__card">
        <view class="detail-row">
          <text class="detail-row__label">品牌</text>
          <text class="detail-row__value">{{ detail.brand_name || detail.brand_id }}</text>
        </view>
        <view class="detail-row">
          <text class="detail-row__label">资产</text>
          <text class="detail-row__value">{{ detail.product_name || detail.product_id || '未命名资产' }}</text>
        </view>
        <view v-if="detail.external_product_id" class="detail-row">
          <text class="detail-row__label">外部 SKU</text>
          <text class="detail-row__value detail-row__value--mono">{{ detail.external_product_id }}</text>
        </view>
        <view v-if="detail.external_product_name" class="detail-row">
          <text class="detail-row__label">外部 SKU 名称</text>
          <text class="detail-row__value">{{ detail.external_product_name }}</text>
        </view>
        <view v-if="detail.external_product_url" class="detail-row">
          <text class="detail-row__label">外部 SKU 详情</text>
          <text class="detail-row__value detail-row__value--link" @tap="openExternalSkuUrl">查看详情 →</text>
        </view>
        <view v-if="detail.virtual_mother_card?.authority_uid" class="detail-row">
          <text class="detail-row__label">虚拟母卡 UID</text>
          <text class="detail-row__value detail-row__value--mono">{{ detail.virtual_mother_card?.authority_uid }}</text>
        </view>
        <view v-if="detail.virtual_mother_card?.credential_hash" class="detail-row">
          <text class="detail-row__label">虚拟母卡凭证</text>
          <view class="detail-row__action-group">
            <text class="detail-row__value detail-row__value--mono">{{ truncateId(detail.virtual_mother_card?.credential_hash || '') }}</text>
            <button class="detail-row__copy-btn" @tap="saveVirtualCredential">保存到本机</button>
          </view>
        </view>
        <view class="detail-row">
          <text class="detail-row__label">资产 ID</text>
          <text class="detail-row__value detail-row__value--mono">{{ truncateId(detail.asset_id) }}</text>
        </view>
        <view v-if="detail.uid" class="detail-row">
          <text class="detail-row__label">UID</text>
          <text class="detail-row__value detail-row__value--mono">{{ detail.uid }}</text>
        </view>
        <view class="detail-row detail-row--last">
          <text class="detail-row__label">创建时间</text>
          <text class="detail-row__value">{{ formatDate(detail.created_at) }}</text>
        </view>
      </view>

      <view v-if="showActions" class="action-bar-placeholder"></view>
    </view>

    <view v-if="showActions" class="action-bar">
      <button class="action-bar__btn action-bar__btn--primary" @tap="goTransfer">
        转让
      </button>
      <button class="action-bar__btn action-bar__btn--secondary" @tap="confirmConsume">
        标记已消费
      </button>
      <button class="action-bar__btn action-bar__btn--secondary" @tap="confirmLegacy">
        标记传承遗珍
      </button>
    </view>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'CAppAssetDetailPage' })

import { ref, computed, onMounted } from 'vue'
import { RcPageLayout, RcLoadingState, RcStatusBadge, RcRiskCard } from '@rcprotocol/ui/uni'
import { formatDate, truncateId } from '@rcprotocol/utils'
import type { AssetDetailVM } from '@rcprotocol/utils'
import { useAuth } from '@rcprotocol/state'
import { useTypedApi } from '../../composables/useTypedApi'
import { generateIdempotencyKey } from '../../composables/useIdempotency'

interface DetailPageError {
  status?: number
  code?: string
  message?: string
}

const { app: appApi } = useTypedApi()
const { isLoggedIn } = useAuth()

const loading = ref(false)
const errorMsg = ref('')
const notFound = ref(false)
const detail = ref<AssetDetailVM | null>(null)
const assetId = ref('')

const ACTIVE_STATES = new Set(['LegallySold', 'Transferred'])
const showActions = computed(() => isLoggedIn.value && detail.value !== null && ACTIVE_STATES.has(detail.value.state))
const isHonorState = computed(() => detail.value !== null && (detail.value.state === 'Consumed' || detail.value.state === 'Legacy'))
const honorIcon = computed(() => detail.value?.state === 'Legacy' ? '👑' : '🏆')
const honorTitle = computed(() => detail.value?.state === 'Legacy' ? '传承遗珍' : '已消费')
const honorDesc = computed(() => detail.value?.state === 'Legacy' ? '该资产已成为传承遗珍' : '该资产已完成使命')

function badgeLabel(badge: string): string {
  const map: Record<string, string> = {
    verified: '✓ 已认证',
    frozen: '❄ 冻结',
  }
  return map[badge] || badge
}

function openExternalSkuUrl() {
  const url = detail.value?.external_product_url
  if (!url) return
  // #ifdef H5
  window.open(url, '_blank')
  // #endif
  // #ifndef H5
  uni.navigateTo({ url: `/pages/webview?url=${encodeURIComponent(url)}` })
  // #endif
}

function saveVirtualCredential() {
  const credential = detail.value?.virtual_mother_card?.credential_hash
  if (!credential || !assetId.value) return
  try {
    uni.setStorageSync(`rc_virtual_credential_${assetId.value}`, credential)
    uni.showToast({ title: '已保存虚拟母卡凭证', icon: 'success' })
  } catch {
    uni.showToast({ title: '保存失败', icon: 'none' })
  }
}

onMounted(() => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as { options?: { assetId?: string } }
  assetId.value = currentPage?.options?.assetId || ''
  if (assetId.value) {
    loadDetail()
  }
})

async function loadDetail() {
  loading.value = true
  errorMsg.value = ''
  notFound.value = false
  try {
    detail.value = await appApi.getMyAsset(assetId.value)
  } catch (error: unknown) {
    const e = error as DetailPageError
    if (e.status === 404 || e.code === 'NOT_FOUND') {
      notFound.value = true
      errorMsg.value = ''
    } else if (e.code === 'NETWORK_ERROR') {
      errorMsg.value = '网络连接失败'
    } else {
      errorMsg.value = e.message || '加载失败'
    }
  } finally {
    loading.value = false
  }
}

function goTransfer() {
  uni.navigateTo({ url: `/pages/vault/transfer?assetId=${assetId.value}` })
}

function confirmConsume() {
  uni.showModal({
    title: '标记已消费',
    content: '确认标记该资产为已消费？此操作不可逆。',
    confirmColor: '#ff3b30',
    success: (res) => {
      if (res.confirm) {
        doStateAction('consume', '已标记为已消费')
      }
    }
  })
}

function confirmLegacy() {
  uni.showModal({
    title: '标记传承遗珍',
    content: '确认标记该资产为传承遗珍？此操作不可逆。',
    confirmColor: '#9C27B0',
    success: (res) => {
      if (res.confirm) {
        doStateAction('legacy', '已标记为传承遗珍')
      }
    }
  })
}

async function doStateAction(action: 'consume' | 'legacy', successMsg: string) {
  try {
    if (action === 'consume') {
      await appApi.consumeAsset(assetId.value, { 'X-Idempotency-Key': generateIdempotencyKey() })
    } else {
      await appApi.legacyAsset(assetId.value, { 'X-Idempotency-Key': generateIdempotencyKey() })
    }
    uni.showToast({ title: successMsg, icon: 'success' })
    loadDetail()
  } catch (error: unknown) {
    handleWriteError(error as DetailPageError)
  }
}

function handleWriteError(error: DetailPageError) {
  const statusMap: Record<number, string> = {
    403: '您没有权限执行此操作',
    409: '该操作已执行或状态不允许',
  }
  if (error.status && statusMap[error.status]) {
    uni.showToast({ title: statusMap[error.status], icon: 'none' })
  } else if (error.code === 'NETWORK_ERROR') {
    uni.showToast({ title: '网络连接失败，请稍后重试', icon: 'none' })
  } else {
    uni.showToast({ title: error.message || '操作失败', icon: 'none' })
  }
}
</script>

<style scoped>
.not-found {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 120rpx 48rpx;
}

.not-found__icon {
  font-size: 80rpx;
  margin-bottom: 24rpx;
}

.not-found__text {
  font-size: 32rpx;
  color: rgba(0, 0, 0, 0.65);
}

.asset-detail {
  display: flex;
  flex-direction: column;
}

.honor-banner {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 64rpx 32rpx 48rpx;
  border-radius: 24rpx;
  margin-bottom: 32rpx;
}

.honor-banner--Consumed {
  background: linear-gradient(180deg, #FFF8E1 0%, #FFE0B2 100%);
}

.honor-banner--Legacy {
  background: linear-gradient(180deg, #F3E5F5 0%, #E1BEE7 100%);
}

.asset-detail__card {
  background-color: #ffffff;
  border-radius: 24rpx;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  padding: 0 32rpx;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 28rpx 0;
  border-bottom: 1rpx solid rgba(0, 0, 0, 0.08);
}

.detail-row--last {
  border-bottom: none;
}

.detail-row__label {
  font-size: 28rpx;
  color: rgba(0, 0, 0, 0.65);
}

.detail-row__value {
  font-size: 28rpx;
  color: #1d1d1f;
  text-align: right;
  word-break: break-all;
}

.detail-row__value--mono {
  font-family: monospace;
}

.detail-row__action-group {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 10rpx;
}

.detail-row__copy-btn {
  padding: 0 20rpx;
  height: 56rpx;
  line-height: 56rpx;
  border-radius: 28rpx;
  background: #0071e3;
  color: #fff;
  font-size: 24rpx;
}
</style>
