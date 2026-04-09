<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useAuth } from '@rcprotocol/state'
import ConsoleLayout from './components/ConsoleLayout.vue'

const { loadFromStorage } = useAuth()
loadFromStorage()

const route = useRoute()
const showNav = computed(() => route.path !== '/login')
</script>

<template>
  <ConsoleLayout v-if="showNav">
    <router-view />
  </ConsoleLayout>
  <router-view v-else />
</template>

<style>
body {
  margin: 0;
  font-family: Inter, 'Noto Sans SC', 'Helvetica Neue', Helvetica, Arial, sans-serif;
  font-size: 14px;
  color: #1d1d1f;
  background-color: #f5f5f7;
  -webkit-font-smoothing: antialiased;
}
* {
  box-sizing: border-box;
}
button:focus-visible {
  outline: 2px solid #0071e3;
  outline-offset: 2px;
}
input:focus-visible {
  outline: none;
}
</style>
