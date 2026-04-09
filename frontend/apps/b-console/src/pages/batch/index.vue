<template>
  <RcPageLayout title="批次管理">
    <div class="batch-placeholder">
      <div class="batch-placeholder__card">
        <h2 class="batch-placeholder__title">当前 MVP 未开放真实批次实体查询</h2>
        <p class="batch-placeholder__desc">
          本页不再使用资产列表伪装为批次视图。待后端开放真实批次接口后，再接入批次列表、筛选与分页能力。
        </p>
        <ul class="batch-placeholder__list">
          <li>当前资产查询请使用其他业务页按真实语义操作。</li>
          <li>如需补齐批次能力，请先定义 batch 实体与对应 API。</li>
        </ul>
      </div>
    </div>
  </RcPageLayout>
</template>

<script setup lang="ts">
defineOptions({ name: 'BConsoleBatchPage' })

import { RcPageLayout } from '@rcprotocol/ui/web'
</script>

<style scoped>
.batch-placeholder {
  display: flex;
  justify-content: center;
  padding: 32px 0;
}
.batch-placeholder__card {
  width: 100%;
  max-width: 760px;
  background: #fff;
  border-radius: 16px;
  padding: 28px;
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.08);
  border: 1px solid rgba(0,0,0,0.05);
}
.batch-placeholder__title {
  margin: 0 0 12px;
  font-size: 20px;
  color: #1d1d1f;
}
.batch-placeholder__desc {
  margin: 0 0 16px;
  line-height: 1.7;
  color: rgba(0,0,0,0.68);
}
.batch-placeholder__list {
  margin: 0;
  padding-left: 20px;
  color: rgba(0,0,0,0.62);
  line-height: 1.8;
}
</style>
