<template>
  <div class="rc-feedback-state rc-feedback-state--success">
    <div class="rc-feedback-state__icon">✅</div>
    <p class="rc-feedback-state__title">{{ title || '操作成功' }}</p>
    <p v-if="message" class="rc-feedback-state__message">{{ message }}</p>
  </div>
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
  padding: 24px;
  border-radius: 12px;
  background: #f4fbf5;
  border: 1px solid rgba(48, 209, 88, 0.18);
}
.rc-feedback-state__icon {
  font-size: 28px;
  margin-bottom: 8px;
}
.rc-feedback-state__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #1d1d1f;
}
.rc-feedback-state__message {
  margin: 6px 0 0;
  font-size: 14px;
  color: rgba(0,0,0,0.6);
}
</style>
