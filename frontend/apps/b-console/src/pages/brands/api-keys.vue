<template>
  <RcPageLayout title="API Key 管理" showBack>
    <div class="api-keys__intro">
      <p class="api-keys__intro-text">
        当前页面管理的是品牌 API Key，用于品牌侧集成与控制台授权。
      </p>
    </div>

    <div v-if="forbidden" class="forbidden">
      <RcForbiddenState message="无权限访问该品牌的 API Key" />
    </div>

    <template v-else>
      <RcLoadingState :loading="loading" :error="errorMsg" @retry="loadApiKeys" />

      <div v-if="!loading && !errorMsg">
        <button class="btn-primary api-keys__create-btn" @click="showCreateDialog">
          创建 API Key
        </button>

        <RcEmptyState v-if="apiKeys.length === 0" message="暂无 API Key" />

        <div v-else class="api-keys__list">
          <div class="api-keys__table">
            <div class="api-keys__thead">
              <span class="api-keys__th api-keys__th--id">Key ID</span>
              <span class="api-keys__th api-keys__th--desc">描述</span>
              <span class="api-keys__th api-keys__th--time">创建时间</span>
              <span class="api-keys__th api-keys__th--time">最后使用</span>
              <span class="api-keys__th api-keys__th--status">状态</span>
              <span class="api-keys__th api-keys__th--action">操作</span>
            </div>
            <div
              v-for="key in apiKeys"
              :key="key.key_id"
              class="api-keys__row"
            >
              <span class="api-keys__td api-keys__td--id">{{ key.key_id }}</span>
              <span class="api-keys__td api-keys__td--desc">{{ key.description || '-' }}</span>
              <span class="api-keys__td api-keys__td--time">{{ formatDate(key.created_at) }}</span>
              <span class="api-keys__td api-keys__td--time">{{ key.last_used_at ? formatDate(key.last_used_at) : '-' }}</span>
              <span class="api-keys__td api-keys__td--status">
                <RcStatusBadge :status="key.status" />
              </span>
              <span class="api-keys__td api-keys__td--action">
                <button
                  v-if="key.status === 'active'"
                  class="btn-danger btn-small"
                  @click="revokeKey(key.key_id)"
                >
                  撤销
                </button>
              </span>
            </div>
          </div>
        </div>
      </div>
    </template>

    <Teleport to="body">
      <div v-if="showDialog" class="dialog-mask" @click="closeDialog">
        <div class="dialog" @click.stop>
          <h3 class="dialog__title">创建品牌 API Key</h3>
          <div class="dialog__field">
            <label class="dialog__label">描述（可选）</label>
            <input
              v-model="newKeyDescription"
              placeholder="例如：品牌 ERP 集成"
              class="dialog__input"
            />
          </div>
          <div class="dialog__actions">
            <button class="btn-secondary" @click="closeDialog">取消</button>
            <button class="btn-primary" :disabled="creating" @click="doCreate">
              {{ creating ? '创建中...' : '创建' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="newApiKey" class="dialog-mask" @click.stop>
        <div class="dialog" @click.stop>
          <h3 class="dialog__title">API Key 创建成功</h3>
          <p class="dialog__warning">⚠️ 请妥善保存以下密钥，此密钥仅显示一次，关闭后无法再次查看。</p>
          <div class="dialog__key-display">
            <code class="dialog__key-text">{{ newApiKey }}</code>
          </div>
          <div class="dialog__actions">
            <button class="btn-secondary" @click="copyKey">复制密钥</button>
            <button class="btn-primary" @click="closeNewKeyDialog">我已保存，关闭</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="revokeConfirmId" class="dialog-mask" @click="cancelRevoke">
        <div class="dialog" @click.stop>
          <h3 class="dialog__title">确认撤销</h3>
          <p class="dialog__warning">撤销后该 API Key 将无法使用，此操作不可恢复。</p>
          <div class="dialog__actions">
            <button class="btn-secondary" @click="cancelRevoke">取消</button>
            <button class="btn-danger" :disabled="revoking" @click="confirmRevoke">
              {{ revoking ? '撤销中...' : '确认撤销' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'BConsoleApiKeysPage' })

import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuth } from '@rcprotocol/state'
import { RcPageLayout, RcLoadingState, RcEmptyState, RcStatusBadge, RcForbiddenState } from '@rcprotocol/ui/web'
import { formatDate } from '@rcprotocol/utils'
import type { ApiKey } from '@rcprotocol/utils'
import { useTypedApi } from '../../composables/useTypedApi'

interface ApiKeyPageError {
  code?: string
  status?: number
  message?: string
}

const route = useRoute()
const { console: consoleApi } = useTypedApi()
const { user } = useAuth()

const brandId = computed(() => (route.query.brandId as string) || '')
const forbidden = ref(false)
const loading = ref(false)
const errorMsg = ref('')
const apiKeys = ref<ApiKey[]>([])

const showDialog = ref(false)
const newKeyDescription = ref('')
const creating = ref(false)
const newApiKey = ref('')

const revokeConfirmId = ref('')
const revoking = ref(false)

onMounted(() => {
  if (!brandId.value) {
    errorMsg.value = '缺少品牌 ID'
    return
  }
  const u = user.value as (typeof user.value & { brand_id?: string })
  if (u?.role === 'Brand' && u.brand_id && u.brand_id !== brandId.value) {
    forbidden.value = true
    return
  }
  loadApiKeys()
})

async function loadApiKeys() {
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await consoleApi.listApiKeys(brandId.value)
    apiKeys.value = res.items || []
  } catch (error: unknown) {
    const e = error as ApiKeyPageError
    if (e.status === 403) {
      forbidden.value = true
    } else {
      errorMsg.value = e.code === 'NETWORK_ERROR' ? '网络连接失败' : '加载失败'
    }
  } finally {
    loading.value = false
  }
}

function showCreateDialog() {
  showDialog.value = true
  newKeyDescription.value = ''
}

function closeDialog() {
  showDialog.value = false
}

async function doCreate() {
  creating.value = true
  try {
    const body: Record<string, string> = {}
    if (newKeyDescription.value.trim()) {
      body.description = newKeyDescription.value.trim()
    }
    const res = await consoleApi.createApiKey(brandId.value, body)
    newApiKey.value = res.api_key
    showDialog.value = false
    await loadApiKeys()
  } catch (error: unknown) {
    const e = error as ApiKeyPageError
    if (e.code === 'FORBIDDEN') {
      errorMsg.value = '无权限创建 API Key'
    } else if (e.code === 'NETWORK_ERROR') {
      errorMsg.value = '网络连接失败，请稍后重试'
    } else {
      errorMsg.value = e.message || '创建失败'
    }
    showDialog.value = false
  } finally {
    creating.value = false
  }
}

function closeNewKeyDialog() {
  newApiKey.value = ''
}

function copyKey() {
  navigator.clipboard.writeText(newApiKey.value).catch(() => {
    // ignore clipboard failure
  })
}

function revokeKey(keyId: string) {
  revokeConfirmId.value = keyId
}

function cancelRevoke() {
  revokeConfirmId.value = ''
}

async function confirmRevoke() {
  revoking.value = true
  try {
    await consoleApi.revokeApiKey(brandId.value, revokeConfirmId.value)
    revokeConfirmId.value = ''
    await loadApiKeys()
  } catch (error: unknown) {
    const e = error as ApiKeyPageError
    if (e.code === 'FORBIDDEN') {
      errorMsg.value = '无权限撤销 API Key'
    } else if (e.code === 'NETWORK_ERROR') {
      errorMsg.value = '网络连接失败，请稍后重试'
    } else {
      errorMsg.value = e.message || '撤销失败'
    }
    revokeConfirmId.value = ''
  } finally {
    revoking.value = false
  }
}
</script>

<style scoped>
.api-keys__intro {
  margin-bottom: 16px;
}
.api-keys__intro-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: rgba(0,0,0,0.68);
}
.forbidden {
  margin-top: 12px;
}
.api-keys__create-btn {
  margin-bottom: 16px;
}
.api-keys__list {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
}
.api-keys__table {
  display: flex;
  flex-direction: column;
}
.api-keys__thead,
.api-keys__row {
  display: grid;
  grid-template-columns: 1.4fr 1.4fr 1fr 1fr 0.8fr 0.8fr;
  gap: 12px;
  align-items: center;
}
.api-keys__thead {
  padding: 0 0 12px;
  border-bottom: 1px solid rgba(0,0,0,0.08);
}
.api-keys__row {
  padding: 14px 0;
  border-bottom: 1px solid rgba(0,0,0,0.04);
}
.api-keys__row:last-child { border-bottom: none; }
.api-keys__th {
  font-size: 12px;
  color: rgba(0,0,0,0.45);
}
.api-keys__td {
  font-size: 13px;
  color: #1d1d1f;
  word-break: break-all;
}
.dialog-mask {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.dialog {
  width: min(480px, calc(100vw - 32px));
  background: #fff;
  border-radius: 12px;
  padding: 20px;
}
.dialog__title { margin: 0 0 16px; }
.dialog__label { display: block; margin-bottom: 8px; font-size: 14px; }
.dialog__input {
  width: 100%;
  height: 40px;
  border: 1px solid rgba(0,0,0,0.12);
  border-radius: 8px;
  padding: 0 12px;
  box-sizing: border-box;
}
.dialog__warning { color: rgba(0,0,0,0.65); line-height: 1.6; }
.dialog__key-display {
  margin: 12px 0 20px;
  padding: 12px;
  border-radius: 8px;
  background: #f6f7fb;
}
.dialog__key-text { word-break: break-all; }
.dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
.btn-primary,
.btn-secondary,
.btn-danger {
  padding: 8px 16px;
  border-radius: 8px;
  border: none;
  cursor: pointer;
}
.btn-primary { background: #0071e3; color: #fff; }
.btn-secondary { background: #f2f2f7; color: #1d1d1f; }
.btn-danger { background: #ff3b30; color: #fff; }
.btn-small { padding: 6px 12px; font-size: 12px; }
</style>
