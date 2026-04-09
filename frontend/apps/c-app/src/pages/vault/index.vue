<template>
  <RcPageLayout title="我的资产馆">
    <view class="vault-tabs">
      <view
        class="vault-tab"
        :class="{ 'vault-tab--active': activeTab === 'active' }"
        @tap="switchTab('active')"
      >
        <text class="vault-tab__text">活跃资产</text>
      </view>
      <view
        class="vault-tab"
        :class="{ 'vault-tab--active': activeTab === 'honor' }"
        @tap="switchTab('honor')"
      >
        <text class="vault-tab__text">荣誉典藏</text>
      </view>
    </view>

    <RcLoadingState :loading="loading" :error="errorMsg" @retry="loadAssets" />

    <view v-if="!loading && !errorMsg">
      <RcEmptyState
        v-if="filteredAssets.length === 0"
        :message="activeTab === 'active' ? '您还没有持有的资产' : '您还没有荣誉典藏资产'"
      />

      <view v-else class="vault-list">
        <view
          v-for="asset in filteredAssets"
          :key="asset.asset_id"
          class="vault-item"
          @tap="goDetail(asset.asset_id)"
        >
          <view class="vault-item__header">
            <text class="vault-item__brand">{{ asset.brand_name || asset.brand_id }}</text>
            <RcStatusBadge :status="asset.state" />
          </view>
          <text class="vault-item__asset">{{ asset.product_name || asset.product_id || '未命名资产' }}</text>

          <view v-if="asset.display_badges?.length" class="vault-item__badges">
            <text
              v-for="badge in asset.display_badges"
              :key="badge"
              class="vault-item__badge"
              :class="{ 'vault-item__badge--honor': isHonor(asset.state) }"
            >
              {{ badgeLabel(badge) }}
            </text>
          </view>

          <view v-else-if="isHonor(asset.state)" class="vault-item__badges">
            <text class="vault-item__badge vault-item__badge--honor">
              {{ asset.state === 'Legacy' ? '👑 传承遗珍' : '🏆 已消费' }}
            </text>
          </view>
        </view>
      </view>

      <view v-if="page < totalPages" class="vault-pagination">
        <button
          class="vault-pagination__btn"
          :disabled="loadingMore"
          @tap="loadMore"
        >
          {{ loadingMore ? '加载中...' : '加载更多' }}
        </button>
      </view>
    </view>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'CAppVaultPage' })

import { ref, computed, onMounted } from 'vue'
import { RcPageLayout, RcLoadingState, RcEmptyState, RcStatusBadge } from '@rcprotocol/ui/uni'
import type { AssetVM } from '@rcprotocol/utils'
import { useTypedApi } from '../../composables/useTypedApi'

interface VaultPageError {
  code?: string
}

const { app: appApi } = useTypedApi()

const ACTIVE_STATES = new Set(['LegallySold', 'Transferred', 'Disputed'])
const HONOR_STATES = new Set(['Consumed', 'Legacy'])

const activeTab = ref<'active' | 'honor'>('active')
const loading = ref(false)
const loadingMore = ref(false)
const errorMsg = ref('')
const allAssets = ref<AssetVM[]>([])
const page = ref(1)
const pageSize = 50
const total = ref(0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const filteredAssets = computed(() => {
  const states = activeTab.value === 'active' ? ACTIVE_STATES : HONOR_STATES
  return allAssets.value.filter((asset) => states.has(asset.state))
})

function isHonor(state: string): boolean {
  return HONOR_STATES.has(state)
}

function badgeLabel(badge: string): string {
  const map: Record<string, string> = {
    verified: '✓ 已认证',
    frozen: '❄ 冻结',
    legacy: '👑 传承遗珍',
    consumed: '🏆 已消费'
  }
  return map[badge] || badge
}

function switchTab(tab: 'active' | 'honor') {
  activeTab.value = tab
}

async function fetchPage(nextPage: number, append: boolean) {
  if (append) {
    loadingMore.value = true
  } else {
    loading.value = true
    errorMsg.value = ''
  }

  try {
    const res = await appApi.listMyAssets({
      page: nextPage,
      page_size: pageSize
    })
    const items = res.items || []
    allAssets.value = append ? [...allAssets.value, ...items] : items
    total.value = res.total
    page.value = res.page
  } catch (error: unknown) {
    const e = error as VaultPageError
    errorMsg.value = e.code === 'NETWORK_ERROR' ? '网络连接失败' : '加载失败，请重试'
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

async function loadAssets() {
  await fetchPage(1, false)
}

async function loadMore() {
  if (page.value >= totalPages.value || loadingMore.value) return
  await fetchPage(page.value + 1, true)
}

function goDetail(assetId: string) {
  uni.navigateTo({ url: `/pages/vault/detail?assetId=${assetId}` })
}

onMounted(loadAssets)
</script>

<style scoped>
.vault-tabs {
  display: flex;
  flex-direction: row;
  height: 88rpx;
  background-color: #ffffff;
  border-bottom: 1rpx solid rgba(0, 0, 0, 0.08);
  margin-bottom: 16rpx;
}

.vault-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 88rpx;
  position: relative;
  box-sizing: border-box;
}

.vault-tab__text {
  font-size: 28rpx;
  color: rgba(0, 0, 0, 0.65);
  font-weight: 400;
}

.vault-tab--active .vault-tab__text {
  color: #1d1d1f;
  font-weight: 600;
}

.vault-tab--active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 64rpx;
  height: 4rpx;
  background-color: #4CAF50;
  border-radius: 2rpx;
}

.vault-list {
  display: flex;
  flex-direction: column;
}

.vault-item {
  background-color: #ffffff;
  border-radius: 16rpx;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  padding: 24rpx;
  margin-bottom: 16rpx;
}

.vault-item__header {
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8rpx;
}

.vault-item__brand {
  font-size: 30rpx;
  font-weight: 600;
  color: #1d1d1f;
}

.vault-item__product {
  font-size: 26rpx;
  color: rgba(0, 0, 0, 0.65);
  margin-bottom: 12rpx;
}

.vault-item__badges {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  gap: 8rpx;
  margin-top: 8rpx;
}

.vault-item__badge {
  display: inline-flex;
  align-items: center;
  font-size: 22rpx;
  color: #0071e3;
  background-color: rgba(0, 113, 227, 0.1);
  padding: 4rpx 16rpx;
  border-radius: 999rpx;
}

.vault-item__badge--honor {
  color: #8e6d00;
  background-color: rgba(255, 159, 10, 0.12);
}

.vault-pagination {
  display: flex;
  justify-content: center;
  padding: 24rpx 0 40rpx;
}

.vault-pagination__btn {
  font-size: 26rpx;
  color: #0071e3;
  background-color: #ffffff;
  border: 1rpx solid #0071e3;
  border-radius: 8rpx;
  padding: 12rpx 32rpx;
  line-height: 1.4;
}
</style>
