<template>
  <view class="rc-loading-state" v-if="loading || error">
    <view v-if="loading"><text>加载中...</text></view>
    <view v-else-if="error" class="rc-loading-state__error">
      <text class="rc-loading-state__error-text">{{ error }}</text>
      <text class="rc-loading-state__retry" @tap="$emit('retry')">重试</text>
    </view>
  </view>
</template>

<script setup lang="ts">
defineProps<{ loading: boolean; error?: string }>()
defineEmits<{ retry: [] }>()
</script>

<style scoped>
.rc-loading-state { display: flex; justify-content: center; align-items: center; padding: 64rpx; }
.rc-loading-state__error { text-align: center; }
.rc-loading-state__error-text { color: #F44336; font-size: 28rpx; display: block; margin-bottom: 16rpx; }
.rc-loading-state__retry { color: #2196F3; font-size: 28rpx; }
</style>
