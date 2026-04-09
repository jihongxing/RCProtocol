<template>
  <view class="rc-risk-card" :class="`rc-risk-card--${level}`">
    <text class="rc-risk-card__icon">{{ icon }}</text>
    <text class="rc-risk-card__message">{{ message }}</text>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
const props = defineProps<{ level: 'risk' | 'warning' | 'info'; message: string }>()
const icon = computed(() => ({ risk: '⚠', warning: '⚡', info: 'ℹ' }[props.level] || 'ℹ'))
</script>

<style scoped>
.rc-risk-card { display: flex; flex-direction: row; align-items: flex-start; padding: 24rpx; border-radius: 12rpx; margin-bottom: 16rpx; }
.rc-risk-card--risk { background-color: #FFEBEE; }
.rc-risk-card--warning { background-color: #FFF3E0; }
.rc-risk-card--info { background-color: #E3F2FD; }
.rc-risk-card__icon { font-size: 32rpx; margin-right: 12rpx; }
.rc-risk-card__message { font-size: 28rpx; flex: 1; }
</style>
