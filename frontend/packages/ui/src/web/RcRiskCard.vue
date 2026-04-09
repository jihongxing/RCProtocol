<template>
  <div class="rc-risk-card" :class="`rc-risk-card--${level}`">
    <span class="rc-risk-card__icon">{{ icon }}</span>
    <span class="rc-risk-card__message">{{ message }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
const props = defineProps<{ level: 'risk' | 'warning' | 'info'; message: string }>()
const icon = computed(() => ({ risk: '⚠', warning: '⚡', info: 'ℹ' }[props.level] || 'ℹ'))
</script>

<style scoped>
.rc-risk-card { display: flex; align-items: flex-start; padding: 12px; border-radius: 6px; margin-bottom: 8px; }
.rc-risk-card--risk { background-color: #FFEBEE; }
.rc-risk-card--warning { background-color: #FFF3E0; }
.rc-risk-card--info { background-color: #E3F2FD; }
.rc-risk-card__icon { font-size: 16px; margin-right: 8px; }
.rc-risk-card__message { font-size: 14px; flex: 1; }
</style>
