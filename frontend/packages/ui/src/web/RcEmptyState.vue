<template>
  <div class="rc-empty-state">
    <p class="rc-empty-state__text">{{ message || '暂无数据' }}</p>
  </div>
</template>

<script setup lang="ts">
defineProps<{ message?: string }>()
</script>

<style scoped>
.rc-empty-state { display: flex; justify-content: center; align-items: center; padding: 32px; }
.rc-empty-state__text { color: #9E9E9E; font-size: 14px; }
</style>
