<template>
  <view class="rc-page-layout">
    <view class="rc-page-layout__header" v-if="title">
      <text v-if="showBack" class="rc-page-layout__back" @tap="goBack">←</text>
      <text class="rc-page-layout__title">{{ title }}</text>
    </view>
    <view class="rc-page-layout__content"><slot /></view>
    <view class="rc-page-layout__footer" v-if="$slots.footer"><slot name="footer" /></view>
  </view>
</template>

<script setup lang="ts">
defineProps<{ title?: string; showBack?: boolean }>()
function goBack() { uni.navigateBack() }
</script>

<style scoped>
.rc-page-layout { display: flex; flex-direction: column; min-height: 100vh; }
.rc-page-layout__header { display: flex; align-items: center; padding: 24rpx 32rpx; background-color: #fff; border-bottom: 1rpx solid #eee; }
.rc-page-layout__back { font-size: 36rpx; margin-right: 16rpx; }
.rc-page-layout__title { font-size: 34rpx; font-weight: 600; }
.rc-page-layout__content { flex: 1; padding: 24rpx; }
.rc-page-layout__footer { padding: 24rpx 32rpx; background-color: #fff; border-top: 1rpx solid #eee; }
</style>
