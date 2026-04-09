<template>
  <div class="console-layout">
    <ConsoleNav />
    <div class="console-layout__right">
      <header class="console-layout__topbar">
        <span class="console-layout__topbar-title">RCProtocol</span>
      </header>
      <div class="console-layout__main">
        <div class="console-layout__content">
          <slot />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import ConsoleNav from './ConsoleNav.vue'
</script>

<style scoped>
.console-layout {
  display: flex;
  min-height: 100vh;
}
.console-layout__right {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.console-layout__topbar {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 24px;
  background-color: #fff;
  border-bottom: 1px solid rgba(0,0,0,0.08);
  flex-shrink: 0;
}
.console-layout__topbar-title {
  font-size: 14px;
  font-weight: 500;
  color: rgba(0,0,0,0.65);
}
.console-layout__main {
  flex: 1;
  overflow-y: auto;
  background-color: #f5f5f7;
}
.console-layout__content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0;
}
</style>
