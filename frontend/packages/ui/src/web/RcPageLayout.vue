<template>
  <div class="rc-page-layout">
    <header class="rc-page-layout__header" v-if="title">
      <button v-if="showBack" class="rc-page-layout__back" @click="goBack">←</button>
      <h1 class="rc-page-layout__title">{{ title }}</h1>
    </header>
    <main class="rc-page-layout__content"><slot /></main>
    <footer class="rc-page-layout__footer" v-if="$slots.footer"><slot name="footer" /></footer>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

defineProps<{ title?: string; showBack?: boolean }>()
const router = useRouter()
function goBack() { router.back() }
</script>

<style scoped>
.rc-page-layout { display: flex; flex-direction: column; min-height: 100vh; }
.rc-page-layout__header { display: flex; align-items: center; padding: 12px 16px; background-color: #fff; border-bottom: 1px solid #eee; }
.rc-page-layout__back { font-size: 18px; margin-right: 8px; background: none; border: none; cursor: pointer; }
.rc-page-layout__title { font-size: 18px; font-weight: 600; margin: 0; }
.rc-page-layout__content { flex: 1; padding: 16px; }
.rc-page-layout__footer { padding: 12px 16px; background-color: #fff; border-top: 1px solid #eee; }
</style>
