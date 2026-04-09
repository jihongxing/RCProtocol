<template>
  <span class="rc-status-badge" :style="{ backgroundColor: bgColor }">
    <span class="rc-status-badge__text" :style="{ color: textColor }">{{ label }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ASSET_STATE_LABELS, ASSET_STATE_COLORS } from '@rcprotocol/utils'

const props = defineProps<{ status: string }>()

const label = computed(() => ASSET_STATE_LABELS[props.status] || props.status)
const bgColor = computed(() => (ASSET_STATE_COLORS[props.status] || '#9E9E9E') + '1A')
const textColor = computed(() => ASSET_STATE_COLORS[props.status] || '#9E9E9E')
</script>

<style scoped>
.rc-status-badge { display: inline-flex; padding: 2px 8px; border-radius: 4px; }
.rc-status-badge__text { font-size: 12px; font-weight: 500; }
</style>
