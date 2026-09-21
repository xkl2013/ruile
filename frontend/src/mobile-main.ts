import { createApp, h } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory, RouterView } from 'vue-router'
import TDesign from 'tdesign-vue-next'
import 'tdesign-vue-next/es/style/index.css'
import '@/assets/theme/theme.css'
import { installTDesignIconOfflineGuard } from '@/utils/tdesign-icon-offline'
import { initTheme } from '@/composables/useTheme'
import { initFont } from '@/composables/useFont'
import i18n from './i18n'
import ServiceHub from '@/views/service/ServiceHub.vue'

installTDesignIconOfflineGuard()
initTheme()
initFont()

const mobileRouteMeta = { requiresInit: true, requiresAuth: true, mobileEntry: true }

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/mobile/service',
    },
    {
      path: '/mobile',
      redirect: '/mobile/service',
    },
    {
      path: '/mobile/service',
      name: 'mobileServiceHub',
      component: ServiceHub,
      meta: mobileRouteMeta,
    },
    {
      path: '/mobile/service/:pathMatch(.*)*',
      redirect: '/mobile/service',
      meta: mobileRouteMeta,
    },
  ],
})

const app = createApp({ render: () => h(RouterView) })

app.use(TDesign)
app.use(createPinia())
app.use(router)
app.use(i18n)

router.isReady().finally(() => {
  app.mount('#mobile-app')
})
