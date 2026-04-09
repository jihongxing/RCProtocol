<template>
  <RcPageLayout :title="brand?.brand_name || '品牌详情'" showBack>
    <RcLoadingState :loading="loading" :error="errorMsg" @retry="loadData" />

    <div v-if="!loading && !errorMsg && brand">
      <div class="brand-info">
        <div class="brand-info__row">
          <span class="brand-info__label">品牌名称</span>
          <span class="brand-info__value">{{ brand.brand_name }}</span>
        </div>
        <div class="brand-info__row">
          <span class="brand-info__label">状态</span>
          <RcStatusBadge :status="brand.status" />
        </div>
        <div class="brand-info__row">
          <span class="brand-info__label">创建时间</span>
          <span class="brand-info__value">{{ formatDate(brand.created_at) }}</span>
        </div>
      </div>

      <div class="detail-tabs">
        <button
          class="detail-tabs__item"
          :class="{ 'detail-tabs__item--active': activeTab === 'products' }"
          @click="activeTab = 'products'"
        >
          外部 SKU 映射
        </button>
        <button
          class="detail-tabs__item"
          :class="{ 'detail-tabs__item--active': activeTab === 'apikeys' }"
          @click="activeTab = 'apikeys'"
        >
          API Key 管理
        </button>
      </div>

      <div v-if="activeTab === 'products'" class="products-section">
        <div class="products-section__header">
          <h3 class="products-section__title">外部 SKU 映射</h3>
          <button v-if="canCreateProduct" class="btn-primary" @click="goCreateProduct">
            新增外部 SKU 映射
          </button>
        </div>

        <RcEmptyState v-if="products.length === 0" message="暂无外部 SKU 映射" />

        <div v-else class="products__list">
          <div v-for="p in products" :key="p.product_id" class="product-item">
            <div class="product-item__main">
              <span class="product-item__name">{{ p.product_name }}</span>
              <span v-if="p.external_product_name" class="product-item__sub">外部 SKU 名称：{{ p.external_product_name }}</span>
              <span v-if="p.external_product_id" class="product-item__meta">外部 SKU：{{ p.external_product_id }}</span>
              <a
                v-if="p.external_product_url"
                :href="p.external_product_url"
                target="_blank"
                rel="noopener noreferrer"
                class="product-item__link"
              >
                查看外部 SKU 页面
              </a>
            </div>
            <span class="product-item__time">{{ formatDate(p.created_at) }}</span>
          </div>
        </div>
      </div>

      <div v-if="activeTab === 'apikeys'" class="apikeys-section">
        <div class="apikeys-section__header">
          <h3 class="apikeys-section__title">品牌 API Key 管理</h3>
        </div>
        <p class="apikeys-section__desc">
          该页面用于管理当前品牌的 API Key。
        </p>
        <button class="btn-primary" @click="goApiKeys">
          管理 API Key
        </button>
      </div>
    </div>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'BConsoleBrandDetailPage' })

import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '@rcprotocol/state'
import { RcPageLayout, RcLoadingState, RcEmptyState, RcStatusBadge } from '@rcprotocol/ui/web'
import { formatDate } from '@rcprotocol/utils'
import type { Brand, Product } from '@rcprotocol/utils'
import { useTypedApi } from '../../composables/useTypedApi'

interface BrandDetailError {
  code?: string
}

const route = useRoute()
const router = useRouter()
const { console: consoleApi } = useTypedApi()
const { user } = useAuth()

const brandId = computed(() => (route.query.brandId as string) || '')
const activeTab = ref<'products' | 'apikeys'>('products')
const loading = ref(false)
const errorMsg = ref('')
const brand = ref<Brand | null>(null)
const products = ref<Product[]>([])

const canCreateProduct = computed(() =>
  user.value?.role === 'Platform' || user.value?.role === 'Brand'
)

async function loadData() {
  if (!brandId.value) return
  loading.value = true
  errorMsg.value = ''
  try {
    const [brandRes, productRes] = await Promise.all([
      consoleApi.getBrand(brandId.value),
      consoleApi.listBrandProducts(brandId.value)
    ])
    brand.value = brandRes
    products.value = productRes.items || []
  } catch (error: unknown) {
    const e = error as BrandDetailError
    errorMsg.value = e.code === 'NOT_FOUND' ? '品牌不存在' : '加载失败'
  } finally {
    loading.value = false
  }
}

function goCreateProduct() {
  router.push(`/brands/external-sku-create?brandId=${brandId.value}`)
}

function goApiKeys() {
  router.push({ path: '/brands/api-keys', query: { brandId: brandId.value } })
}

onMounted(loadData)
</script>

<style scoped>
.brand-info {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
  margin-bottom: 24px;
}
.brand-info__row {
  display: flex;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid rgba(0,0,0,0.04);
}
.brand-info__row:last-child { border-bottom: none; }
.brand-info__label {
  width: 100px;
  font-size: 13px;
  color: rgba(0,0,0,0.40);
  flex-shrink: 0;
}
.brand-info__value {
  font-size: 14px;
  color: #1d1d1f;
}
.detail-tabs {
  display: flex;
  border-bottom: 1px solid rgba(0,0,0,0.08);
  margin-bottom: 20px;
}
.detail-tabs__item {
  padding: 10px 20px;
  font-size: 14px;
  font-weight: 500;
  color: rgba(0,0,0,0.65);
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: color 0.2s, border-color 0.2s;
}
.detail-tabs__item:hover {
  color: #0071e3;
}
.detail-tabs__item--active {
  color: #0071e3;
  border-bottom-color: #0071e3;
}
.apikeys-section,
.products__list {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
}
.apikeys-section__header,
.products-section__header {
  margin-bottom: 12px;
}
.products-section__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.apikeys-section__title,
.products-section__title {
  font-size: 18px;
  font-weight: 600;
  color: #1d1d1f;
  margin: 0;
}
.apikeys-section__desc {
  font-size: 14px;
  color: rgba(0,0,0,0.65);
  margin: 0 0 16px;
  line-height: 1.5;
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
}
.btn-primary:hover { opacity: 0.85; }
.product-item {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px solid rgba(0,0,0,0.04);
}
.product-item:last-child { border-bottom: none; }
.product-item__main {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.product-item__name { font-size: 14px; font-weight: 600; color: #1d1d1f; }
.product-item__sub,
.product-item__meta { font-size: 12px; color: rgba(0,0,0,0.55); }
.product-item__link { font-size: 12px; color: #0071e3; text-decoration: none; }
.product-item__time { font-size: 12px; color: rgba(0,0,0,0.40); white-space: nowrap; }
</style>
