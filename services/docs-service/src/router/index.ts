import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
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
  // TODO: auth guard
})

export default router
