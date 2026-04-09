<template>
  <view class="rc-feedback-state rc-feedback-state--forbidden">
    <text class="rc-feedback-state__icon">🔒</text>
    <text class="rc-feedback-state__title">{{ title || '无权限访问' }}</text>
    <text v-if="message" class="rc-feedback-state__message">{{ message }}</text>
  </view>
</template>

<script setup lang="ts">
defineProps<{ title?: string; message?: string }>()
</script>

<style scoped>
.rc-feedback-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 64rpx 32rpx;
  border-radius: 24rpx;
  background: #ffffff;
  border: 1rpx dashed rgba(0,0,0,0.1);
}
.rc-feedback-state__icon {
  font-size: 56rpx;
  margin-bottom: 16rpx;
}
.rc-feedback-state__title {
  font-size: 32rpx;
  font-weight: 600;
  color: #1d1d1f;
}
.rc-feedback-state__message {
  margin-top: 12rpx;
  font-size: 28rpx;
  color: rgba(0,0,0,0.6);
  text-align: center;
}
</style>
