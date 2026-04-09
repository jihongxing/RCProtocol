<template>
  <RcPageLayout title="品牌管理">
    <RcLoadingState :loading="loading" :error="errorMsg" @retry="loadBrands" />

    <div v-if="!loading && !errorMsg">
      <div class="brands__toolbar">
        <button v-if="canCreate" class="btn-primary" @click="goCreate">
          创建品牌
        </button>
      </div>

      <RcEmptyState v-if="brands.length === 0" message="暂无品牌数据" />

      <div v-else class="brands__list">
        <div
          v-for="brand in brands"
          :key="brand.brand_id"
          class="brand-item"
          @click="goDetail(brand.brand_id)"
        >
          <div class="brand-item__info">
            <span class="brand-item__name">{{ brand.brand_name }}</span>
            <span class="brand-item__time">{{ formatDate(brand.created_at) }}</span>
          </div>
          <RcStatusBadge :status="brand.status" />
        </div>
      </div>

      <div v-if="total > pageSize" class="brands__pagination">
        <button :disabled="page <= 1" class="btn-ghost" @click="prevPage">上一页</button>
        <span class="brands__page-info">{{ page }} / {{ totalPages }}</span>
        <button :disabled="page >= totalPages" class="btn-ghost" @click="nextPage">下一页</button>
      </div>
    </div>
  </RcPageLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '@rcprotocol/state'
import { RcPageLayout, RcLoadingState, RcEmptyState, RcStatusBadge } from '@rcprotocol/ui/web'
import { formatDate } from '@rcprotocol/utils'
import type { Brand } from '@rcprotocol/utils'
import { useTypedApi } from '../../composables/useTypedApi'

const router = useRouter()
const { console: consoleApi } = useTypedApi()
const { user } = useAuth()

const loading = ref(false)
const errorMsg = ref('')
const brands = ref<Brand[]>([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const totalPages = computed(() => Math.ceil(total.value / pageSize))

const canCreate = computed(() =>
  user.value?.role === 'Platform' || user.value?.role === 'Brand'
)

async function loadBrands() {
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await consoleApi.listBrands({
      page: page.value,
      page_size: pageSize
    })
    brands.value = res.items || []
    total.value = res.total
  } catch (e: any) {
    errorMsg.value = e.code === 'NETWORK_ERROR' ? '网络连接失败' : '加载失败'
  } finally {
    loading.value = false
  }
}

function goCreate() { router.push('/brands/create') }
function goDetail(brandId: string) { router.push(`/brands/detail?brandId=${brandId}`) }
function prevPage() { if (page.value > 1) { page.value--; loadBrands() } }
function nextPage() { if (page.value < totalPages.value) { page.value++; loadBrands() } }

onMounted(loadBrands)
</script>

<style scoped>
.brands__toolbar {
  margin-bottom: 16px;
}
.btn-primary {
  padding: 8px 20px;
  background-color: #0071e3;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
}
.btn-primary:hover { opacity: 0.85; }
.brands__list {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
  overflow: hidden;
}
.brand-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  transition: background-color 0.15s;
  border-bottom: 1px solid rgba(0,0,0,0.04);
}
.brand-item:last-child { border-bottom: none; }
.brand-item:hover { background-color: #f0f0f5; }
.brand-item__info { display: flex; flex-direction: column; gap: 2px; }
.brand-item__name { font-size: 14px; font-weight: 600; color: #1d1d1f; }
.brand-item__time { font-size: 12px; color: rgba(0,0,0,0.40); }
.brands__pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-top: 16px;
}
.brands__page-info { font-size: 14px; color: rgba(0,0,0,0.65); }
.btn-ghost {
  padding: 6px 16px;
  background: transparent;
  border: none;
  color: #0071e3;
  font-size: 14px;
  cursor: pointer;
}
.btn-ghost:disabled { color: rgba(0,0,0,0.25); cursor: not-allowed; }
</style>
