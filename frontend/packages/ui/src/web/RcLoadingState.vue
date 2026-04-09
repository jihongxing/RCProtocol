<template>
  <div class="rc-loading-state" v-if="loading || error">
    <div v-if="loading"><span>加载中...</span></div>
    <div v-else-if="error" class="rc-loading-state__error">
      <p class="rc-loading-state__error-text">{{ error }}</p>
      <button class="rc-loading-state__retry" @click="$emit('retry')">重试</button>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ loading: boolean; error?: string }>()
defineEmits<{ retry: [] }>()
</script>

<style scoped>
.rc-loading-state { display: flex; justify-content: center; align-items: center; padding: 32px; }
.rc-loading-state__error { text-align: center; }
.rc-loading-state__error-text { color: #F44336; font-size: 14px; margin-bottom: 8px; }
.rc-loading-state__retry { color: #2196F3; font-size: 14px; background: none; border: none; cursor: pointer; }
</style>
