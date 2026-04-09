<template>
  <RcPageLayout title="资产验真">
    <view v-if="!result && !loading && !networkError && !scanError" class="verify-entry">
      <view class="verify-entry__icon">🔍</view>
      <text class="verify-entry__title">验证资产真伪</text>
      <text class="verify-entry__desc">扫描资产标签上的二维码，或手动输入 UID</text>

      <button class="verify-entry__scan-btn" @tap="doScan">扫码验真</button>

      <view class="verify-entry__manual">
        <input
          v-model="manualUid"
          placeholder="手动输入 UID（14 位 hex）"
          maxlength="14"
          class="verify-entry__input"
        />
        <button
          class="verify-entry__manual-btn"
          :disabled="!isValidUid"
          @tap="doManualVerify"
        >
          查询
        </button>
      </view>
    </view>

    <view v-if="scanError && !loading" class="verify-scan-error">
      <view class="verify-scan-error__icon">⚠</view>
      <text class="verify-scan-error__text">{{ scanError }}</text>
      <button class="verify-scan-error__btn" @tap="reset">返回重试</button>
    </view>

    <RcLoadingState :loading="loading" :error="networkError" @retry="retryVerify" />

    <view v-if="result && !loading" class="verify-result">
      <view class="verify-result__icon" :class="`verify-result__icon--${normalizedStatus}`">
        <text class="verify-result__icon-text">{{ statusIcon }}</text>
      </view>

      <text class="verify-result__title">{{ statusTitle }}</text>
      <text class="verify-result__desc">{{ statusDesc }}</text>

      <RcRiskCard
        v-if="hasReplayRisk"
        level="warning"
        message="检测到异常扫描记录，该标签可能存在安全风险"
      />
      <RcRiskCard
        v-if="hasFrozenRisk"
        level="risk"
        message="该资产当前处于争议冻结状态"
      />

      <view v-if="showAssetInfo && result.asset" class="verify-result__asset">
        <view class="asset-field">
          <text class="asset-field__label">状态</text>
          <RcStatusBadge :status="result.asset.current_state || result.asset.state || '-'" />
        </view>
        <view class="asset-field">
          <text class="asset-field__label">品牌</text>
          <text class="asset-field__value">{{ result.asset.brand_name || result.asset.brand_id || '-' }}</text>
        </view>
        <view class="asset-field">
          <text class="asset-field__label">资产</text>
          <text class="asset-field__value">{{ result.asset.product_name || result.asset.product_id || '-' }}</text>
        </view>
        <view class="asset-field">
          <text class="asset-field__label">UID</text>
          <text class="asset-field__value mono">{{ result.asset.uid || '-' }}</text>
        </view>
        <view v-if="typeof result.scan_metadata?.ctr === 'number'" class="asset-field">
          <text class="asset-field__label">扫描计数</text>
          <text class="asset-field__value">{{ result.scan_metadata?.ctr }}</text>
        </view>
      </view>

      <view class="verify-result__actions">
        <button v-if="canEnterVault" class="verify-result__primary" @tap="goVault">进入资产馆</button>
        <button class="verify-result__rescan" @tap="reset">重新扫码</button>
      </view>
    </view>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'CAppVerifyPage' })

import { ref, computed } from 'vue'
import { RcPageLayout, RcLoadingState, RcStatusBadge, RcRiskCard } from '@rcprotocol/ui/uni'
import type { VerifyResponse } from '@rcprotocol/utils'
import { getErrorMessage } from '@rcprotocol/api'
import { useTypedApi } from '../composables/useTypedApi'

interface VerifyUrlParams {
  uid: string
  ctr?: number
  cmac?: string
}

const { app: appApi } = useTypedApi()

const loading = ref(false)
const networkError = ref('')
const scanError = ref('')
const result = ref<VerifyResponse | null>(null)
const manualUid = ref('')
const lastParams = ref<VerifyUrlParams | null>(null)

const isValidUid = computed(() => /^[0-9a-fA-F]{14}$/.test(manualUid.value))
const normalizedStatus = computed(() => normalizeStatus(result.value))
const canEnterVault = computed(() => Boolean(result.value?.asset?.asset_id) && ['verified', 'restricted', 'unverified'].includes(normalizedStatus.value))

const statusIcon = computed(() => {
  const map: Record<string, string> = {
    verified: '✓',
    failed: '✗',
    unknown: '?',
    restricted: '⚠',
    unverified: 'ℹ'
  }
  return map[normalizedStatus.value] || '?'
})

const statusTitle = computed(() => {
  const map: Record<string, string> = {
    verified: '认证通过',
    failed: '认证失败',
    unknown: '未知标签',
    restricted: '受限状态',
    unverified: '未认证查询'
  }
  return map[normalizedStatus.value] || '未知'
})

const statusDesc = computed(() => {
  const map: Record<string, string> = {
    verified: '该资产已通过动态认证校验',
    failed: '该标签可能为仿冒品，请谨慎对待',
    unknown: '该标签未在系统中注册',
    restricted: '该资产当前处于受限或冻结状态',
    unverified: '仅通过 UID 查询，未进行密码学认证'
  }
  return map[normalizedStatus.value] || ''
})

const hasReplayRisk = computed(() => result.value?.risk_flags?.includes('replay_suspected') ?? false)
const hasFrozenRisk = computed(() => result.value?.risk_flags?.includes('frozen_asset') ?? false)
const showAssetInfo = computed(() => ['verified', 'restricted', 'unverified'].includes(normalizedStatus.value))

function normalizeStatus(value: VerifyResponse | null): 'verified' | 'failed' | 'unknown' | 'restricted' | 'unverified' {
  const status = value?.verification_status
  if (status === 'authentication_failed' || status === 'failed') return 'failed'
  if (status === 'unknown_tag' || status === 'unknown') return 'unknown'
  if (status === 'restricted') return 'restricted'
  if (status === 'unverified') return 'unverified'
  return 'verified'
}

function parseVerifyUrl(url: string): VerifyUrlParams | null {
  try {
    const parsed = new URL(url.startsWith('http') ? url : `https://placeholder.com${url}`)
    const uid = parsed.searchParams.get('uid')
    const ctr = parsed.searchParams.get('ctr')
    const cmac = parsed.searchParams.get('cmac')
    if (uid && ctr && cmac) {
      return { uid, ctr: Number(ctr), cmac }
    }
    return null
  } catch {
    return null
  }
}

async function doScan() {
  scanError.value = ''
  try {
    const scanResult = await new Promise<UniApp.ScanCodeSuccessRes>((resolve, reject) => {
      uni.scanCode({
        onlyFromCamera: false,
        success: resolve,
        fail: reject
      })
    })

    const params = parseVerifyUrl(scanResult.result)
    if (!params) {
      scanError.value = '无效的验真码，请检查标签'
      return
    }

    lastParams.value = params
    await callVerify(params)
  } catch {
    // ignore user cancel
  }
}

async function doManualVerify() {
  if (!isValidUid.value) return
  scanError.value = ''
  lastParams.value = { uid: manualUid.value }
  await callVerify({ uid: manualUid.value })
}

async function callVerify(params: VerifyUrlParams) {
  loading.value = true
  networkError.value = ''
  scanError.value = ''
  result.value = null

  try {
    result.value = await appApi.verify({
      uid: params.uid,
      ctr: params.ctr ?? 0,
      cmac: params.cmac || ''
    })
  } catch (error: unknown) {
    networkError.value = getErrorMessage(error as never, '验真')
  } finally {
    loading.value = false
  }
}

function retryVerify() {
  if (lastParams.value) {
    callVerify(lastParams.value)
  }
}

function goVault() {
  const assetId = result.value?.asset?.asset_id
  if (!assetId) return
  uni.navigateTo({ url: `/pages/vault/detail?assetId=${assetId}` })
}

function reset() {
  result.value = null
  networkError.value = ''
  scanError.value = ''
  manualUid.value = ''
  lastParams.value = null
}
</script>

<style scoped>
.verify-entry {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80rpx 32rpx;
}

.verify-entry__icon {
  font-size: 96rpx;
  margin-bottom: 24rpx;
}

.verify-entry__title {
  font-size: 40rpx;
  font-weight: 600;
  color: #1d1d1f;
  margin-bottom: 12rpx;
}

.verify-entry__desc {
  font-size: 28rpx;
  color: rgba(0, 0, 0, 0.65);
  text-align: center;
  line-height: 1.5;
  margin-bottom: 40rpx;
}

.verify-entry__scan-btn,
.verify-entry__manual-btn,
.verify-result__rescan,
.verify-scan-error__btn,
.verify-result__primary {
  background-color: #0071e3;
  color: #ffffff;
  border: none;
  border-radius: 16rpx;
}

.verify-entry__manual {
  display: flex;
  width: 100%;
  gap: 16rpx;
  margin-top: 32rpx;
}

.verify-entry__input {
  flex: 1;
  height: 80rpx;
  border: 1rpx solid rgba(0, 0, 0, 0.12);
  border-radius: 16rpx;
  padding: 0 24rpx;
  background: #ffffff;
  box-sizing: border-box;
}

.verify-result,
.verify-scan-error {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.verify-result__actions {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.asset-field {
  display: flex;
  justify-content: space-between;
  gap: 24rpx;
}

.asset-field__label {
  color: rgba(0, 0, 0, 0.65);
}

.asset-field__value {
  color: #1d1d1f;
  word-break: break-all;
}

.mono {
  font-family: monospace;
}
</style>
