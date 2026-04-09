<template>
  <RcPageLayout title="登录">
    <view class="login-card">
      <text class="login-card__title">RCProtocol</text>

      <view class="login-form">
        <view class="login-form__field">
          <text class="login-form__label">邮箱</text>
          <input
            v-model="email"
            type="text"
            placeholder="请输入邮箱"
            class="login-form__input"
            :class="{ 'login-form__input--error': emailError }"
          />
          <text v-if="emailError" class="login-form__hint">{{ emailError }}</text>
        </view>

        <view class="login-form__field">
          <text class="login-form__label">密码</text>
          <input
            v-model="password"
            type="safe-password"
            placeholder="请输入密码"
            class="login-form__input"
            :class="{ 'login-form__input--error': passwordError }"
            @confirm="doLogin"
          />
          <text v-if="passwordError" class="login-form__hint">{{ passwordError }}</text>
        </view>

        <view v-if="orgList.length > 0" class="login-form__field">
          <text class="login-form__label">选择组织</text>
          <picker :range="orgNames" @change="onOrgSelect">
            <view class="login-form__input login-form__picker">
              <text :class="{ 'login-form__picker-placeholder': !selectedOrg }">
                {{ selectedOrg ? `${selectedOrg.org_name} · ${selectedOrg.role}` : '请选择组织' }}
              </text>
              <text class="login-form__picker-arrow">▾</text>
            </view>
          </picker>
        </view>

        <RcRiskCard v-if="errorMsg" level="risk" :message="errorMsg" />

        <button
          class="login-form__btn"
          :disabled="submitting || (orgList.length > 0 && !selectedOrg)"
          @tap="doLogin"
        >
          {{ submitting ? '登录中...' : orgList.length > 0 ? '确认组织并登录' : '登录' }}
        </button>
      </view>
    </view>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'CAppLoginPage' })

import { ref, computed } from 'vue'
import { useAuth } from '@rcprotocol/state'
import { RcPageLayout, RcRiskCard } from '@rcprotocol/ui/uni'
import { getErrorMessage } from '@rcprotocol/api'
import { useTypedApi } from '../composables/useTypedApi'

interface LoginOrgOption {
  org_id: string
  org_name: string
  role: string
}

interface LoginApiError {
  code?: string
  requires_org_selection?: boolean
  available_orgs?: LoginOrgOption[]
  organizations?: LoginOrgOption[]
}

const { auth: authApi } = useTypedApi()
const { login } = useAuth()

const email = ref('')
const password = ref('')
const submitting = ref(false)
const errorMsg = ref('')
const emailError = ref('')
const passwordError = ref('')
const orgList = ref<LoginOrgOption[]>([])
const selectedOrg = ref<LoginOrgOption | null>(null)

const orgNames = computed(() => orgList.value.map((o) => `${o.org_name} · ${o.role}`))

function onOrgSelect(event: { detail: { value: number | string } }) {
  const idx = Number(event.detail.value)
  selectedOrg.value = orgList.value[idx] || null
}

function validate(): boolean {
  emailError.value = ''
  passwordError.value = ''
  let valid = true

  if (!email.value.includes('@')) {
    emailError.value = '请输入有效的邮箱地址'
    valid = false
  }
  if (password.value.length < 8) {
    passwordError.value = '密码至少 8 个字符'
    valid = false
  }
  return valid
}

function clearOrgSelection() {
  orgList.value = []
  selectedOrg.value = null
}

async function doLogin() {
  errorMsg.value = ''
  if (!validate()) return

  submitting.value = true
  try {
    const basePayload = {
      email: email.value.trim(),
      password: password.value,
    }

    const res = selectedOrg.value
      ? await authApi.selectOrg({ ...basePayload, org_id: selectedOrg.value.org_id })
      : await authApi.login(basePayload)

    clearOrgSelection()
    login(res.token, res.user)
    uni.switchTab({ url: '/pages/vault/index' })
  } catch (error: unknown) {
    const e = error as LoginApiError
    const organizations = (e.available_orgs || e.organizations || []) as LoginOrgOption[]

    if ((e.code === 'ORG_SELECTION_REQUIRED' || e.requires_org_selection) && organizations.length > 0) {
      orgList.value = organizations
      selectedOrg.value = organizations[0] || null
      errorMsg.value = '请选择要登录的组织'
    } else if (e.code === 'AUTH_REQUIRED') {
      clearOrgSelection()
      errorMsg.value = '邮箱或密码错误'
    } else if (e.code === 'FORBIDDEN') {
      clearOrgSelection()
      errorMsg.value = '账号已禁用或未绑定组织'
    } else {
      errorMsg.value = getErrorMessage(error as never, '登录')
    }
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.login-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin: 80rpx 32rpx 0;
  background: #ffffff;
  border-radius: 16rpx;
  padding: 64rpx 40rpx 48rpx;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.login-card__title {
  font-size: 48rpx;
  font-weight: 600;
  color: #0071e3;
  text-align: center;
  margin-bottom: 56rpx;
}

.login-form {
  width: 100%;
}

.login-form__field {
  margin-bottom: 32rpx;
}

.login-form__label {
  display: block;
  font-size: 28rpx;
  font-weight: 500;
  color: rgba(0, 0, 0, 0.65);
  margin-bottom: 12rpx;
}

.login-form__input {
  width: 100%;
  height: 80rpx;
  border: 2rpx solid rgba(0, 0, 0, 0.12);
  border-radius: 16rpx;
  padding: 0 24rpx;
  font-size: 28rpx;
  color: #1d1d1f;
  background: #ffffff;
  box-sizing: border-box;
}

.login-form__input--error {
  border-color: #ff3b30;
}

.login-form__hint {
  display: block;
  font-size: 24rpx;
  color: #ff3b30;
  margin-top: 8rpx;
}

.login-form__picker {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
}

.login-form__picker-placeholder {
  color: rgba(0, 0, 0, 0.4);
}

.login-form__picker-arrow {
  font-size: 24rpx;
  color: rgba(0, 0, 0, 0.4);
}

.login-form__btn {
  width: 100%;
  height: 88rpx;
  line-height: 88rpx;
  border: none;
  border-radius: 16rpx;
  background-color: #0071e3;
  color: #ffffff;
  font-size: 32rpx;
  font-weight: 500;
  text-align: center;
  margin-top: 16rpx;
}

.login-form__btn::after {
  border: none;
}

.login-form__btn[disabled] {
  opacity: 0.5;
}
</style>
