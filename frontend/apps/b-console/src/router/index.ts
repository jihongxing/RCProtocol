import { createRouter, createWebHashHistory } from 'vue-router'
import { useAuth } from '@rcprotocol/state'

const routes = [
  {
    path: '/',
    redirect: '/login'
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../pages/login.vue')
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('../pages/dashboard.vue')
  },
  {
    path: '/brands',
    name: 'BrandList',
    component: () => import('../pages/brands/index.vue'),
    meta: { roles: ['Platform', 'Brand'] }
  },
  {
    path: '/brands/detail',
    name: 'BrandDetail',
    component: () => import('../pages/brands/detail.vue'),
    meta: { roles: ['Platform', 'Brand'] }
  },
  {
    path: '/brands/create',
    name: 'BrandCreate',
    component: () => import('../pages/brands/create.vue'),
    meta: { roles: ['Platform'] }
  },
  {
    path: '/brands/api-keys',
    name: 'BrandApiKeys',
    component: () => import('../pages/brands/api-keys.vue'),
    meta: { roles: ['Platform', 'Brand'] }
  },
  {
    path: '/brands/external-sku-create',
    name: 'ExternalSkuCreate',
    component: () => import('../pages/brands/product-create.vue'),
    meta: { roles: ['Platform', 'Brand'] }
  },
  {
    path: '/audit',
    name: 'Audit',
    component: () => import('../pages/audit/index.vue'),
    meta: { roles: ['Platform', 'Moderator'] }
  },
  {
    path: '/batch',
    name: 'Batch',
    component: () => import('../pages/batch/index.vue'),
    meta: { roles: ['Platform', 'Brand', 'Factory'] }
  },
  {
    path: '/scan',
    name: 'Scan',
    component: () => import('../pages/scan/index.vue'),
    meta: { roles: ['Platform', 'Factory'] }
  },
  {
    path: '/activate',
    name: 'Activate',
    component: () => import('../pages/activate/index.vue'),
    meta: { roles: ['Platform', 'Brand'] }
  },
  {
    path: '/sell',
    name: 'Sell',
    component: () => import('../pages/sell/index.vue'),
    meta: { roles: ['Platform', 'Brand'] }
  },
  {
    path: '/authority-devices',
    name: 'AuthorityDevices',
    component: () => import('../pages/authority-devices/index.vue'),
    meta: { roles: ['Platform', 'Brand'] }
  },
  {
    path: '/authority-devices/register-physical',
    name: 'RegisterPhysicalMotherCard',
    component: () => import('../pages/authority-devices/register-physical.vue'),
    meta: { roles: ['Platform', 'Brand'] }
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

function getFallbackPathForRole(role?: string): string {
  switch (role) {
    case 'Factory':
      return '/batch'
    case 'Brand':
      return '/brands'
    case 'Moderator':
      return '/audit'
    default:
      return '/dashboard'
  }
}

router.beforeEach((to) => {
  const { isLoggedIn, user } = useAuth()

  if (to.path === '/login' && isLoggedIn.value) {
    return getFallbackPathForRole(user.value?.role)
  }

  if (to.path !== '/login' && !isLoggedIn.value) {
    return '/login'
  }

  if (to.meta.roles && Array.isArray(to.meta.roles)) {
    const allowedRoles = to.meta.roles as string[]
    const userRole = user.value?.role

    if (userRole && !allowedRoles.includes(userRole)) {
      return getFallbackPathForRole(userRole)
    }
  }
})

export default router
