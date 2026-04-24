import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/pages/LoginPage.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      name: 'home',
      component: () => import('@/pages/HomePage.vue'),
    },
    {
      path: '/services/:serviceId',
      name: 'service',
      component: () => import('@/pages/ServicePage.vue'),
      props: true,
    },
    {
      path: '/services/:serviceId/:protocol',
      name: 'protocol',
      component: () => import('@/pages/ProtocolPage.vue'),
      props: true,
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true

  const authStore = useAuthStore()

  if (!authStore.isAuthenticated) {
    const ok = await authStore.validate()
    if (!ok) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }
  }

  return true
})

export default router
