<template>
  <RcPageLayout title="注册物理母卡" showBack>
    <view class="register-form">
      <!-- 芯片 UID -->
      <view class="form-field">
        <text class="form-label">芯片 UID *</text>
        <input
          v-model="formData.chip_uid"
          placeholder="请输入 14 位十六进制 UID"
          class="form-input"
          :class="{ 'form-input--error': uidError }"
          maxlength="14"
          @input="validateUid"
        />
        <text v-if="uidError" class="form-error">{{ uidError }}</text>
        <text class="form-hint">示例: 04E1A2B3C4D5E6</text>
      </view>

      <!-- 品牌选择 -->
      <view class="form-field">
        <text class="form-label">所属品牌 *</text>
        <picker
          mode="selector"
          :range="brandOptions"
          range-key="name"
          @change="onBrandChange"
        >
          <view class="form-picker">
            <text :class="{ 'form-picker__placeholder': !selectedBrand }">
              {{ selectedBrand ? selectedBrand.name : '请选择品牌' }}
            </text>
            <text class="form-picker__arrow">▼</text>
          </view>
        </picker>
      </view>

      <!-- 资产 ID -->
      <view class="form-field">
        <text class="form-label">绑定资产 ID *</text>
        <input
          v-model="formData.asset_id"
          placeholder="请输入资产 ID"
          class="form-input"
        />
      </view>

      <!-- 密钥纪元 -->
      <view class="form-field">
        <text class="form-label">密钥纪元 *</text>
        <input
          v-model.number="formData.key_epoch"
          type="number"
          placeholder="请输入密钥纪元（通常为 1）"
          class="form-input"
        />
        <text class="form-hint">密钥版本号，通常为 1</text>
      </view>

      <!-- 错误提示 -->
      <RcRiskCard v-if="errorMsg" level="risk" :message="errorMsg" />

      <!-- 提交按钮 -->
      <button
        class="submit-btn"
        :disabled="!canSubmit"
        @tap="handleSubmit"
      >
        {{ submitting ? '注册中...' : '注册物理母卡' }}
      </button>
    </view>
  </RcPageLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { RcPageLayout, RcRiskCard } from '@rcprotocol/ui/uni'
import { useTypedApi } from '../../composables/useTypedApi'

const { console: consoleApi } = useTypedApi()

interface BrandOption {
  brand_id: string
  name: string
}

const formData = ref({
  chip_uid: '',
  brand_id: '',
  asset_id: '',
  key_epoch: 1
})

const brandOptions = ref<BrandOption[]>([])
const selectedBrand = ref<BrandOption | null>(null)
const uidError = ref('')
const errorMsg = ref('')
const submitting = ref(false)

const canSubmit = computed(() => {
  return (
    !submitting.value &&
    formData.value.chip_uid.length === 14 &&
    !uidError.value &&
    formData.value.brand_id &&
    formData.value.asset_id &&
    formData.value.key_epoch > 0
  )
})

onMounted(async () => {
  try {
    // Load brand list
    const brandsRes = await consoleApi.listBrands({ page: 1, page_size: 100 })
    brandOptions.value = (brandsRes.items || []).map((item) => ({
      brand_id: item.brand_id,
      name: item.brand_name,
    }))
  } catch {
    errorMsg.value = '加载品牌列表失败'
  }
})

function validateUid() {
  const uid = formData.value.chip_uid.toUpperCase()
  formData.value.chip_uid = uid

  if (uid.length === 0) {
    uidError.value = ''
    return
  }

  if (uid.length !== 14) {
    uidError.value = 'UID 必须为 14 位'
    return
  }

  if (!/^[0-9A-F]{14}$/.test(uid)) {
    uidError.value = 'UID 必须为十六进制字符（0-9, A-F）'
    return
  }

  uidError.value = ''
}

function onBrandChange(e: any) {
  const index = e.detail.value
  selectedBrand.value = brandOptions.value[index]
  formData.value.brand_id = selectedBrand.value.brand_id
}

async function handleSubmit() {
  if (!canSubmit.value) return

  submitting.value = true
  errorMsg.value = ''

  try {
    await consoleApi.registerPhysicalAuthorityDevice(formData.value as Record<string, unknown>)

    uni.showToast({ title: '注册成功', icon: 'success' })
    setTimeout(() => {
      uni.navigateBack()
    }, 1000)
  } catch (e: any) {
    if (e.status === 409 || e.code === 'CONFLICT') {
      errorMsg.value = '该芯片 UID 已被注册'
    } else if (e.status === 403 || e.code === 'FORBIDDEN') {
      errorMsg.value = '您没有权限注册物理母卡'
    } else if (e.status === 400 || e.code === 'INVALID_INPUT') {
      errorMsg.value = e.message || '输入数据格式错误'
    } else if (e.code === 'NETWORK_ERROR') {
      errorMsg.value = '网络连接失败，请稍后重试'
    } else {
      errorMsg.value = e.message || '注册失败'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.register-form {
  display: flex;
  flex-direction: column;
  gap: 32rpx;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
}

.form-label {
  font-size: 28rpx;
  font-weight: 500;
  color: #1d1d1f;
}

.form-input {
  width: 100%;
  height: 80rpx;
  border: 1rpx solid rgba(0, 0, 0, 0.12);
  border-radius: 16rpx;
  padding: 0 24rpx;
  font-size: 28rpx;
  color: #1d1d1f;
  background-color: #ffffff;
  box-sizing: border-box;
}

.form-input--error {
  border-color: #ff3b30;
}

.form-picker {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  height: 80rpx;
  border: 1rpx solid rgba(0, 0, 0, 0.12);
  border-radius: 16rpx;
  padding: 0 24rpx;
  font-size: 28rpx;
  color: #1d1d1f;
  background-color: #ffffff;
  box-sizing: border-box;
}

.form-picker__placeholder {
  color: rgba(0, 0, 0, 0.3);
}

.form-picker__arrow {
  font-size: 20rpx;
  color: rgba(0, 0, 0, 0.45);
}

.form-error {
  font-size: 24rpx;
  color: #ff3b30;
}

.form-hint {
  font-size: 24rpx;
  color: rgba(0, 0, 0, 0.45);
}

.submit-btn {
  width: 100%;
  height: 88rpx;
  line-height: 88rpx;
  text-align: center;
  font-size: 32rpx;
  font-weight: 500;
  color: #ffffff;
  background-color: #0071e3;
  border-radius: 16rpx;
  border: none;
  padding: 0;
  margin-top: 16rpx;
}

.submit-btn[disabled] {
  opacity: 0.4;
}
</style>
