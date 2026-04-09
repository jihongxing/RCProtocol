<template>
  <nav class="console-nav">
    <div class="console-nav__brand">
      <span class="console-nav__logo">RC</span>
      <span class="console-nav__app-name">治理后台</span>
    </div>

    <div class="console-nav__user">
      <div class="console-nav__name">{{ user?.display_name || '未登录' }}</div>
      <div class="console-nav__role">{{ roleLabel }}</div>
    </div>

    <div class="console-nav__flow">
      <div class="console-nav__flow-title">最小运营闭环</div>
      <div class="console-nav__flow-steps">品牌 → API Key / SKU → 盲扫 → 激活 → 售出 → 审计</div>
    </div>

    <div class="console-nav__menu">
      <a
        v-for="item in visibleMenuItems"
        :key="item.path"
        class="console-nav__item"
        :class="{ 'console-nav__item--active': isActive(item.path) }"
        @click="navigate(item.path)"
      >
        {{ item.label }}
      </a>
    </div>

    <div class="console-nav__footer">
      <button class="console-nav__logout" @click="doLogout">退出登录</button>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '@rcprotocol/state'
import { ROLE_LABELS } from '@rcprotocol/utils'
import type { MenuItem } from '@rcprotocol/utils'

const route = useRoute()
const router = useRouter()
const { user, logout } = useAuth()

const roleLabel = computed(() =>
  user.value ? (ROLE_LABELS[user.value.role] || user.value.role) : ''
)

interface MenuItemWithRoles extends MenuItem {
  roles?: string[]
}

const menuItems: MenuItemWithRoles[] = [
  { label: 'Dashboard', path: '/dashboard' },
  { label: '品牌管理', path: '/brands', roles: ['Platform', 'Brand'] },
  { label: '批次管理', path: '/batch', roles: ['Platform', 'Brand', 'Factory'] },
  { label: '盲扫任务', path: '/scan', roles: ['Platform', 'Factory'] },
  { label: '激活操作', path: '/activate', roles: ['Platform', 'Brand'] },
  { label: '售出确认', path: '/sell', roles: ['Platform', 'Brand'] },
  { label: '物理母卡管理', path: '/authority-devices', roles: ['Platform', 'Brand'] },
  { label: '审计查询', path: '/audit', roles: ['Platform', 'Moderator'] }
]

const visibleMenuItems = computed(() =>
  menuItems.filter(item =>
    !item.roles || (user.value && item.roles.includes(user.value.role))
  )
)

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(path + '/')
}

function navigate(path: string) {
  router.push(path)
}

function doLogout() {
  logout(() => router.replace('/login'))
}
</script>

<style scoped>
.console-nav {
  width: 240px;
  min-height: 100vh;
  background-color: #1d1d1f;
  color: #fff;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}
.console-nav__brand {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 20px 16px 12px;
  border-bottom: 1px solid rgba(255,255,255,0.12);
}
.console-nav__logo {
  font-size: 18px;
  font-weight: 700;
  color: #0071e3;
}
.console-nav__app-name {
  font-size: 14px;
  font-weight: 500;
  color: rgba(255,255,255,0.7);
}
.console-nav__user {
  padding: 16px;
  border-bottom: 1px solid rgba(255,255,255,0.12);
}
.console-nav__name {
  font-size: 14px;
  font-weight: 500;
  color: #fff;
}
.console-nav__role {
  font-size: 12px;
  color: rgba(255,255,255,0.5);
  margin-top: 2px;
}
.console-nav__flow {
  margin: 12px 16px 8px;
  padding: 12px;
  border-radius: 10px;
  background: rgba(0,113,227,0.12);
  border: 1px solid rgba(0,113,227,0.18);
}
.console-nav__flow-title {
  font-size: 12px;
  font-weight: 700;
  color: #78b8ff;
  margin-bottom: 4px;
}
.console-nav__flow-steps {
  font-size: 12px;
  line-height: 1.5;
  color: rgba(255,255,255,0.72);
}
.console-nav__menu {
  flex: 1;
  padding: 8px 0;
}
.console-nav__item {
  display: block;
  padding: 10px 16px;
  font-size: 14px;
  color: rgba(255,255,255,0.7);
  cursor: pointer;
  text-decoration: none;
  transition: background-color 0.15s, color 0.15s;
  border-left: 3px solid transparent;
}
.console-nav__item:hover {
  background-color: rgba(255,255,255,0.06);
  color: #fff;
}
.console-nav__item--active {
  color: #fff;
  background-color: rgba(255,255,255,0.1);
  border-left-color: #0071e3;
}
.console-nav__footer {
  padding: 16px;
  border-top: 1px solid rgba(255,255,255,0.12);
}
.console-nav__logout {
  width: 100%;
  padding: 8px;
  font-size: 13px;
  color: rgba(255,255,255,0.6);
  background: none;
  border: 1px solid rgba(255,255,255,0.15);
  border-radius: 8px;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}
.console-nav__logout:hover {
  color: #ff3b30;
  border-color: #ff3b30;
}
</style>
