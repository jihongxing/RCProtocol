<template>
  <div class="login-page">
    <div class="login-card">
      <h1 class="login-card__title">RCProtocol 治理后台</h1>

      <div class="login-form">
        <div class="login-form__field">
          <label class="login-form__label">邮箱</label>
          <input
            v-model="email"
            type="text"
            placeholder="请输入邮箱"
            class="login-form__input"
            :class="{ 'login-form__input--error': emailError }"
          />
          <span v-if="emailError" class="login-form__hint">{{ emailError }}</span>
        </div>

        <div class="login-form__field">
          <label class="login-form__label">密码</label>
          <input
            v-model="password"
            type="password"
            placeholder="请输入密码"
            class="login-form__input"
            :class="{ 'login-form__input--error': passwordError }"
            @keyup.enter="doLogin"
          />
          <span v-if="passwordError" class="login-form__hint">{{ passwordError }}</span>
        </div>

        <div v-if="orgList.length > 0" class="login-form__field">
          <label class="login-form__label">选择组织</label>
          <select v-model="selectedOrgId" class="login-form__input">
            <option value="" disabled>请选择组织</option>
            <option v-for="org in orgList" :key="org.org_id" :value="org.org_id">
              {{ org.org_name }} · {{ org.role }}
            </option>
          </select>
        </div>

        <RcRiskCard v-if="errorMsg" level="risk" :message="errorMsg" />

        <button
          class="login-form__btn"
          :disabled="submitting || (orgList.length > 0 && !selectedOrgId)"
          @click="doLogin"
        >
          {{ submitting ? '登录中...' : orgList.length > 0 ? '确认组织并登录' : '登录' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'BConsoleLoginPage' })

import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '@rcprotocol/state'
import { getErrorMessage } from '@rcprotocol/api'
import { RcRiskCard } from '@rcprotocol/ui/web'
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

const router = useRouter()
const { auth: authApi } = useTypedApi()
const { login } = useAuth()

const email = ref('')
const password = ref('')
const submitting = ref(false)
const errorMsg = ref('')
const emailError = ref('')
const passwordError = ref('')
const orgList = ref<LoginOrgOption[]>([])
const selectedOrgId = ref('')

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
  selectedOrgId.value = ''
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

    const res = selectedOrgId.value
      ? await authApi.selectOrg({ ...basePayload, org_id: selectedOrgId.value })
      : await authApi.login(basePayload)

    clearOrgSelection()
    login(res.token, res.user)
    router.replace('/dashboard')
  } catch (error: unknown) {
    const e = error as LoginApiError
    const organizations = (e.available_orgs || e.organizations || []) as LoginOrgOption[]
    if ((e.code === 'ORG_SELECTION_REQUIRED' || e.requires_org_selection) && organizations.length > 0) {
      orgList.value = organizations
      selectedOrgId.value = organizations[0]?.org_id || ''
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
.login-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background-color: #f5f5f7;
}
.login-card {
  width: 100%;
  max-width: 400px;
  background: #fff;
  border-radius: 8px;
  padding: 48px 32px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
}
.login-card__title {
  font-size: 24px;
  font-weight: 600;
  text-align: center;
  color: #1d1d1f;
  margin: 0 0 32px;
}
.login-form__field {
  margin-bottom: 20px;
}
.login-form__label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: rgba(0,0,0,0.65);
  margin-bottom: 6px;
}
.login-form__input {
  width: 100%;
  height: 40px;
  border: 1px solid rgba(0,0,0,0.12);
  border-radius: 8px;
  padding: 0 12px;
  font-size: 14px;
  color: #1d1d1f;
  background: #fff;
  box-sizing: border-box;
  outline: none;
  transition: border-color 0.2s;
}
.login-form__input:focus {
  border-color: #0071e3;
}
.login-form__input--error {
  border-color: #ff3b30;
}
.login-form__hint {
  display: block;
  font-size: 12px;
  color: #ff3b30;
  margin-top: 4px;
}
.login-form__btn {
  width: 100%;
  height: 40px;
  border: none;
  border-radius: 8px;
  background-color: #0071e3;
  color: #fff;
  font-size: 16px;
  font-weight: 500;
  cursor: pointer;
  margin-top: 8px;
  transition: opacity 0.2s;
}
.login-form__btn:hover:not(:disabled) {
  opacity: 0.85;
}
.login-form__btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
