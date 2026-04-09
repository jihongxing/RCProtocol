<template>
  <RcPageLayout title="物理母卡管理">
    <div class="authority-devices-page">
      <div class="page-header">
        <div>
          <h1>物理母卡管理</h1>
          <p class="page-header__desc">当前 MVP 仅闭环物理母卡注册，列表查询接口尚未开放。</p>
        </div>
        <button @click="navigateToRegister" class="btn-primary">
          注册新母卡
        </button>
      </div>

      <div class="empty-state">
        暂无可查询的物理母卡列表。请先通过“注册新母卡”完成录入，待后端开放列表接口后再补齐此页。
      </div>
    </div>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'BConsoleAuthorityDevicesPage' })

import { useRouter } from 'vue-router'
import { RcPageLayout } from '@rcprotocol/ui/web'

const router = useRouter()

const navigateToRegister = () => {
  router.push('/authority-devices/register-physical')
}
</script>

<style scoped>
.authority-devices-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}
.page-header h1 {
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 8px;
}
.page-header__desc {
  margin: 0;
  color: rgba(0,0,0,0.58);
}
.btn-primary {
  padding: 8px 16px;
  background: #1890ff;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
}
.btn-primary:hover {
  background: #40a9ff;
}
.empty-state {
  padding: 48px 24px;
  border-radius: 12px;
  background: #fff;
  color: rgba(0,0,0,0.62);
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
}
</style>
