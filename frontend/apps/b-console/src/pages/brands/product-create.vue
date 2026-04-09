<template>
  <RcPageLayout title="新增外部 SKU 映射" showBack>
    <div class="create-form">
      <div class="create-form__field">
        <label class="create-form__label">资产名称</label>
        <input
          v-model="productName"
          placeholder="请输入资产名称"
          class="create-form__input"
          :class="{ 'create-form__input--error': !!fieldErrors.productName }"
        />
        <span v-if="fieldErrors.productName" class="create-form__hint">{{ fieldErrors.productName }}</span>
      </div>

      <div class="create-form__field">
        <label class="create-form__label">外部 SKU</label>
        <input
          v-model="externalProductId"
          placeholder="请输入 external_sku"
          class="create-form__input"
        />
      </div>

      <div class="create-form__field">
        <label class="create-form__label">外部 SKU 名称</label>
        <input
          v-model="externalProductName"
          placeholder="请输入 external_sku_name"
          class="create-form__input"
        />
      </div>

      <div class="create-form__field">
        <label class="create-form__label">外部 SKU URL</label>
        <input
          v-model="externalProductUrl"
          placeholder="请输入 external_sku_url"
          class="create-form__input"
          :class="{ 'create-form__input--error': !!fieldErrors.externalProductUrl }"
        />
        <span v-if="fieldErrors.externalProductUrl" class="create-form__hint">{{ fieldErrors.externalProductUrl }}</span>
      </div>

      <RcRiskCard v-if="errorMsg" level="risk" :message="errorMsg" />

      <button
        class="create-form__btn"
        :disabled="submitting"
        @click="doCreate"
      >
        {{ submitting ? '提交中...' : '保存映射' }}
      </button>
    </div>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'BConsoleExternalSkuCreatePage' })

import { ref, reactive, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { RcPageLayout, RcRiskCard } from '@rcprotocol/ui/web'
import { useTypedApi } from '../../composables/useTypedApi'

interface ExternalSkuCreateError {
  code?: string
  message?: string
}

const route = useRoute()
const router = useRouter()
const { console: consoleApi } = useTypedApi()

const brandId = computed(() => (route.query.brandId as string) || '')
const productName = ref('')
const externalProductId = ref('')
const externalProductName = ref('')
const externalProductUrl = ref('')
const submitting = ref(false)
const errorMsg = ref('')
const fieldErrors = reactive({
  productName: '',
  externalProductUrl: '',
})

function validate(): boolean {
  fieldErrors.productName = ''
  fieldErrors.externalProductUrl = ''

  let valid = true
  if (!productName.value.trim()) {
    fieldErrors.productName = '资产名称不能为空'
    valid = false
  }

  if (externalProductUrl.value.trim()) {
    try {
      new URL(externalProductUrl.value.trim())
    } catch {
      fieldErrors.externalProductUrl = '请输入有效的 URL'
      valid = false
    }
  }

  return valid
}

async function doCreate() {
  errorMsg.value = ''
  if (!validate()) return

  submitting.value = true
  try {
    await consoleApi.createBrandProduct(brandId.value, {
      product_name: productName.value.trim(),
      external_product_id: externalProductId.value.trim() || undefined,
      external_product_name: externalProductName.value.trim() || undefined,
      external_product_url: externalProductUrl.value.trim() || undefined,
    })
    router.back()
  } catch (error: unknown) {
    const e = error as ExternalSkuCreateError
    if (e.code === 'FORBIDDEN') {
      errorMsg.value = '无权限保存映射'
    } else if (e.code === 'NETWORK_ERROR') {
      errorMsg.value = '网络连接失败，请稍后重试'
    } else {
      errorMsg.value = e.message || '保存失败'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.create-form { max-width: 480px; }
.create-form__field { margin-bottom: 20px; }
.create-form__label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: rgba(0,0,0,0.65);
  margin-bottom: 6px;
}
.create-form__input {
  width: 100%;
  height: 40px;
  border: 1px solid rgba(0,0,0,0.12);
  border-radius: 8px;
  padding: 0 12px;
  font-size: 14px;
  color: #1d1d1f;
  outline: none;
  transition: border-color 0.2s;
}
.create-form__input:focus { border-color: #0071e3; }
.create-form__input--error { border-color: #ff3b30; }
.create-form__hint { display: block; font-size: 12px; color: #ff3b30; margin-top: 4px; }
.create-form__btn {
  padding: 8px 20px;
  background-color: #0071e3;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 500;
  cursor: pointer;
}
.create-form__btn:hover:not(:disabled) { opacity: 0.85; }
.create-form__btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
