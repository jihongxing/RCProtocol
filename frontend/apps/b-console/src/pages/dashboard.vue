<template>
  <RcPageLayout title="Dashboard">
    <RcLoadingState :loading="loading" :error="errorMsg" @retry="loadDashboard" />

    <div v-if="!loading && !errorMsg" class="dashboard">
      <div v-if="degraded" class="dashboard__banner">
        当前数据可能不完整，请稍后刷新重试。
      </div>

      <div class="dashboard__flow-card">
        <div class="dashboard__flow-title">B 端最小运营闭环</div>
        <div class="dashboard__flow-steps">1. 创建品牌 → 2. 配置 API Key / 外部 SKU → 3. 盲扫登记 → 4. 资产激活 → 5. 售出确认 → 6. 审计核对</div>
      </div>

      <div v-if="stats.length === 0" class="dashboard__empty">
        暂无可展示的仪表盘数据
      </div>

      <div v-else class="dashboard__stats">
        <div v-for="stat in stats" :key="stat.label" class="stat-card">
          <div class="stat-card__value">{{ stat.value }}</div>
          <div class="stat-card__label">{{ stat.label }}</div>
        </div>
      </div>

      <div class="dashboard__quick-actions">
        <button class="dashboard__action" @click="router.push('/brands/create')">1. 创建品牌</button>
        <button class="dashboard__action" @click="router.push('/brands')">2. 配置 API Key / 外部 SKU</button>
        <button class="dashboard__action" @click="router.push('/scan')">3. 盲扫登记</button>
        <button class="dashboard__action" @click="router.push('/activate')">4. 资产激活</button>
        <button class="dashboard__action" @click="router.push('/sell')">5. 售出确认</button>
        <button class="dashboard__action" @click="openAudit">6. 审计核对</button>
      </div>

      <button class="dashboard__refresh" @click="loadDashboard">刷新</button>
    </div>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'BConsoleDashboardPage' })

import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { RcPageLayout, RcLoadingState } from '@rcprotocol/ui/web'
import { getErrorMessage } from '@rcprotocol/api'
import type { DashboardData } from '@rcprotocol/utils/types'
import { useTypedApi } from '../composables/useTypedApi'

const route = useRoute()
const router = useRouter()
const { console: consoleApi } = useTypedApi()
const loading = ref(false)
const errorMsg = ref('')
const degraded = ref(false)

const data = ref<DashboardData | null>(null)

const stats = computed(() => {
  if (!data.value) return []
  return [
    { label: '品牌数', value: normalizeCount(data.value.total_brands) },
    { label: '资产总数', value: normalizeCount(data.value.total_assets) },
    { label: '活跃资产', value: normalizeCount(data.value.active_assets) },
    { label: '待审批', value: normalizeCount(data.value.pending_approvals) }
  ]
})

function normalizeCount(value: number | null | undefined): number {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    degraded.value = true
    return 0
  }
  return value
}

function openAudit() {
  const resourceId = typeof route.query.assetId === 'string' ? route.query.assetId : ''
  if (resourceId) {
    router.push({ path: '/audit', query: { resourceType: 'asset', resourceId } })
    return
  }
  router.push('/audit')
}

async function loadDashboard() {
  loading.value = true
  errorMsg.value = ''
  degraded.value = false
  try {
    data.value = await consoleApi.getDashboard()
  } catch (error: unknown) {
    data.value = null
    errorMsg.value = getErrorMessage(error as never, '加载仪表盘')
  } finally {
    loading.value = false
  }
}

onMounted(loadDashboard)
</script>

<style scoped>
.dashboard__banner {
  margin-bottom: 16px;
  padding: 12px 16px;
  border-radius: 8px;
  background: rgba(255, 159, 10, 0.14);
  color: #8a5600;
  font-size: 14px;
}
.dashboard__flow-card {
  margin-bottom: 16px;
  padding: 16px;
  border-radius: 12px;
  background: rgba(0, 113, 227, 0.08);
  border: 1px solid rgba(0, 113, 227, 0.14);
}
.dashboard__flow-title {
  font-size: 14px;
  font-weight: 700;
  color: #0071e3;
  margin-bottom: 6px;
}
.dashboard__flow-steps {
  font-size: 13px;
  color: rgba(0,0,0,0.62);
  line-height: 1.6;
}
.dashboard__empty {
  padding: 40px 24px;
  border-radius: 12px;
  background: #fff;
  text-align: center;
  color: rgba(0,0,0,0.45);
  margin-bottom: 24px;
}
.dashboard__stats {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}
.stat-card {
  background: #fff;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
}
.stat-card__value {
  font-size: 24px;
  font-weight: 600;
  color: #1d1d1f;
  line-height: 1.25;
}
.stat-card__label {
  font-size: 14px;
  color: rgba(0,0,0,0.40);
  margin-top: 4px;
}
.dashboard__quick-actions {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
  margin-bottom: 24px;
}
.dashboard__action {
  padding: 14px 16px;
  border: 1px solid rgba(0,113,227,0.18);
  border-radius: 12px;
  background: #fff;
  color: #0071e3;
  font-size: 14px;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
  transition: background-color 0.15s, border-color 0.15s;
}
.dashboard__action:hover {
  background: rgba(0,113,227,0.06);
  border-color: rgba(0,113,227,0.32);
}
.dashboard__refresh {
  padding: 8px 20px;
  border: 1px solid #0071e3;
  border-radius: 8px;
  background: transparent;
  color: #0071e3;
  font-size: 14px;
  cursor: pointer;
  transition: background-color 0.15s, color 0.15s;
}
.dashboard__refresh:hover {
  background-color: #0071e3;
  color: #fff;
}
</style>
