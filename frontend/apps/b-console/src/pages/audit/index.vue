<template>
  <RcPageLayout title="审计查询">
    <div class="audit">
      <div class="audit__search">
        <select v-model="resourceType" class="audit__select">
          <option value="asset">资产</option>
          <option value="brand">品牌</option>
          <option value="product">外部 SKU 映射</option>
        </select>
        <input
          v-model="resourceId"
          placeholder="请输入资源 ID"
          class="audit__input"
          @keyup.enter="doSearch"
        />
        <button class="audit__btn" :disabled="searching" @click="doSearch">
          {{ searching ? '查询中...' : '查询' }}
        </button>
      </div>

      <RcLoadingState :loading="searching" :error="errorMsg" @retry="doSearch" />

      <RcEmptyState
        v-if="!searching && !errorMsg && searched && logs.length === 0"
        message="暂无审计日志"
      />

      <div v-if="!searching && !errorMsg && logs.length > 0" class="audit__result">
        <div class="audit__summary">
          共 {{ total }} 条日志 · 当前第 {{ page }} 页
        </div>

        <div v-for="log in logs" :key="log.log_id" class="audit-log-item">
          <div class="audit-log-item__header">
            <strong>{{ log.event_type }}</strong>
            <span>{{ formatDate(log.created_at) }}</span>
          </div>
          <div class="audit-log-item__meta">
            <span>actor: {{ log.actor_id }}</span>
            <span>resource: {{ log.resource_type }} / {{ log.resource_id }}</span>
          </div>
          <pre class="audit-log-item__details">{{ JSON.stringify(log.details, null, 2) }}</pre>
        </div>

        <div v-if="total > pageSize" class="audit__pagination">
          <button class="audit__btn audit__btn--ghost" :disabled="page <= 1" @click="changePage(page - 1)">上一页</button>
          <span>{{ page }} / {{ totalPages }}</span>
          <button class="audit__btn audit__btn--ghost" :disabled="page >= totalPages" @click="changePage(page + 1)">下一页</button>
        </div>
      </div>
    </div>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'BConsoleAuditPage' })

import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { RcPageLayout, RcLoadingState, RcEmptyState } from '@rcprotocol/ui/web'
import { formatDate } from '@rcprotocol/utils'
import type { AuditLog } from '@rcprotocol/api'
import { useTypedApi } from '../../composables/useTypedApi'

interface AuditPageError {
  code?: string
  message?: string
}

const route = useRoute()
const { console: consoleApi } = useTypedApi()
const resourceType = ref('asset')
const resourceId = ref('')
const searching = ref(false)
const errorMsg = ref('')
const searched = ref(false)
const logs = ref<AuditLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

onMounted(() => {
  if (typeof route.query.resourceType === 'string') {
    resourceType.value = route.query.resourceType
  }
  if (typeof route.query.resourceId === 'string') {
    resourceId.value = route.query.resourceId
    doSearch()
  }
})

async function doSearch() {
  if (!resourceId.value.trim()) return

  searching.value = true
  errorMsg.value = ''
  logs.value = []
  searched.value = true

  try {
    const response = await consoleApi.listAuditLogs({
      resource_type: resourceType.value,
      resource_id: resourceId.value.trim(),
      page: page.value,
      page_size: pageSize,
    })
    logs.value = response.items || []
    total.value = response.total || 0
  } catch (error: unknown) {
    const e = error as AuditPageError
    if (e.code === 'NETWORK_ERROR') {
      errorMsg.value = '网络连接失败'
    } else {
      errorMsg.value = e.message || '查询失败'
    }
  } finally {
    searching.value = false
  }
}

function changePage(nextPage: number) {
  page.value = nextPage
  doSearch()
}
</script>

<style scoped>
.audit__search {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
}
.audit__select,
.audit__input {
  height: 40px;
  border: 1px solid rgba(0,0,0,0.12);
  border-radius: 8px;
  padding: 0 12px;
  font-size: 14px;
  color: #1d1d1f;
  outline: none;
}
.audit__select {
  width: 140px;
}
.audit__input {
  flex: 1;
  max-width: 420px;
}
.audit__btn {
  height: 40px;
  padding: 0 20px;
  background-color: #0071e3;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}
.audit__btn--ghost {
  background: transparent;
  color: #0071e3;
  border: 1px solid #0071e3;
}
.audit__btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.audit__result {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.audit__summary {
  color: rgba(0,0,0,0.55);
  font-size: 13px;
}
.audit-log-item {
  background: #fff;
  border-radius: 10px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
}
.audit-log-item__header,
.audit-log-item__meta {
  display: flex;
  justify-content: space-between;
  gap: 16px;
}
.audit-log-item__header {
  margin-bottom: 8px;
}
.audit-log-item__meta {
  color: rgba(0,0,0,0.55);
  font-size: 12px;
  margin-bottom: 12px;
}
.audit-log-item__details {
  margin: 0;
  padding: 12px;
  border-radius: 8px;
  background: #f6f7fb;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-word;
}
.audit__pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-top: 8px;
}
</style>
