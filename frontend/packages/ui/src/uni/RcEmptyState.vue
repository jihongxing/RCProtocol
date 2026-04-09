<template>
  <view class="rc-empty-state">
    <text class="rc-empty-state__text">{{ message || '暂无数据' }}</text>
  </view>
</template>

<script setup lang="ts">
defineProps<{ message?: string }>()
</script>

<style scoped>
.rc-empty-state { display: flex; justify-content: center; align-items: center; padding: 64rpx; }
.rc-empty-state__text { color: #9E9E9E; font-size: 28rpx; }
</style>
