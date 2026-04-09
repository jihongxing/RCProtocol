<template>
  <RcPageLayout title="创建品牌" showBack>
    <div class="create-form">
      <div class="create-form__field">
        <label class="create-form__label">品牌名称</label>
        <input
          v-model="brandName"
          placeholder="请输入品牌名称"
          class="create-form__input"
          :class="{ 'create-form__input--error': !!fieldErrors.brandName }"
        />
        <span v-if="fieldErrors.brandName" class="create-form__hint">{{ fieldErrors.brandName }}</span>
      </div>

      <div class="create-form__field">
        <label class="create-form__label">联系邮箱</label>
        <input
          v-model="contactEmail"
          placeholder="请输入联系邮箱"
          class="create-form__input"
          :class="{ 'create-form__input--error': !!fieldErrors.contactEmail }"
        />
        <span v-if="fieldErrors.contactEmail" class="create-form__hint">{{ fieldErrors.contactEmail }}</span>
      </div>

      <div class="create-form__field">
        <label class="create-form__label">联系电话</label>
        <input
          v-model="contactPhone"
          placeholder="请输入联系电话"
          class="create-form__input"
          :class="{ 'create-form__input--error': !!fieldErrors.contactPhone }"
        />
        <span v-if="fieldErrors.contactPhone" class="create-form__hint">{{ fieldErrors.contactPhone }}</span>
      </div>

      <RcRiskCard v-if="errorMsg" level="risk" :message="errorMsg" />

      <RcSuccessState
        v-if="createdBrandId"
        title="品牌创建成功"
        :message="`品牌 ID：${createdBrandId}，即将跳转到品牌详情页...`"
      />

      <button
        v-if="!createdBrandId"
        class="create-form__btn"
        :disabled="submitting"
        @click="doCreate"
      >
        {{ submitting ? '提交中...' : '创建品牌' }}
      </button>
    </div>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'BConsoleBrandCreatePage' })

import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { RcPageLayout, RcRiskCard, RcSuccessState } from '@rcprotocol/ui/web'
import { useTypedApi } from '../../composables/useTypedApi'

interface CreateBrandError {
  code?: string
  message?: string
}

const router = useRouter()
const { console: consoleApi } = useTypedApi()

const brandName = ref('')
const contactEmail = ref('')
const contactPhone = ref('')
const submitting = ref(false)
const errorMsg = ref('')
const createdBrandId = ref('')
const fieldErrors = reactive({
  brandName: '',
  contactEmail: '',
  contactPhone: ''
})

function clearFieldErrors() {
  fieldErrors.brandName = ''
  fieldErrors.contactEmail = ''
  fieldErrors.contactPhone = ''
}

function validate(): boolean {
  clearFieldErrors()
  let valid = true

  if (!brandName.value.trim()) {
    fieldErrors.brandName = '品牌名称不能为空'
    valid = false
  }

  if (!contactEmail.value.trim()) {
    fieldErrors.contactEmail = '联系邮箱不能为空'
    valid = false
  } else if (!contactEmail.value.includes('@')) {
    fieldErrors.contactEmail = '请输入有效的邮箱地址'
    valid = false
  }

  if (!contactPhone.value.trim()) {
    fieldErrors.contactPhone = '联系电话不能为空'
    valid = false
  }

  return valid
}

async function doCreate() {
  errorMsg.value = ''
  createdBrandId.value = ''

  if (!validate()) return

  submitting.value = true
  try {
    const res = await consoleApi.createBrand({
      brand_name: brandName.value.trim(),
      contact_email: contactEmail.value.trim(),
      contact_phone: contactPhone.value.trim()
    })

    createdBrandId.value = res.brand_id

    setTimeout(() => {
      router.push({ path: '/brands/detail', query: { brandId: res.brand_id } })
    }, 2000)
  } catch (error: unknown) {
    const e = error as CreateBrandError
    if (e.code === 'CONFLICT') {
      errorMsg.value = '该品牌名称已存在'
    } else if (e.code === 'FORBIDDEN') {
      errorMsg.value = '无权限创建品牌'
    } else if (e.code === 'NETWORK_ERROR') {
      errorMsg.value = '网络连接失败，请稍后重试'
    } else {
      errorMsg.value = e.message || '创建失败'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.create-form {
  max-width: 480px;
}
.create-form__field {
  margin-bottom: 20px;
}
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
.create-form__hint {
  display: block;
  font-size: 12px;
  color: #ff3b30;
  margin-top: 4px;
}
.create-form__btn {
  padding: 8px 20px;
  background-color: #0071e3;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
}
.create-form__btn:hover:not(:disabled) { opacity: 0.85; }
.create-form__btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
