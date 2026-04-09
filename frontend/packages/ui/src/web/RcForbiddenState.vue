<template>
  <div class="rc-feedback-state rc-feedback-state--forbidden">
    <div class="rc-feedback-state__icon">🔒</div>
    <p class="rc-feedback-state__title">{{ title || '无权限访问' }}</p>
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
  padding: 32px;
  border-radius: 12px;
  background: #fff;
  border: 1px dashed rgba(0,0,0,0.1);
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
