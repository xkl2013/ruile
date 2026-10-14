import { createApp, h } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory, RouterView } from 'vue-router'
import TDesign from 'tdesign-vue-next'
import 'tdesign-vue-next/es/style/index.css'
import '@/assets/theme/theme.css'
import '@/views/service/serviceHub.less'
import { installTDesignIconOfflineGuard } from '@/utils/tdesign-icon-offline'
import { initTheme } from '@/composables/useTheme'
import { initFont } from '@/composables/useFont'
import i18n from './i18n'
import ServiceMobileEntryView from '@/views/service/ServiceMobileEntryView.vue'
import ServiceOverviewView from '@/views/service/ServiceOverviewView.vue'
import ServiceActivityView from '@/views/service/ServiceActivityView.vue'
import ServiceSettingsView from '@/views/service/ServiceSettingsView.vue'

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
      component: ServiceMobileEntryView,
      meta: mobileRouteMeta,
    },
    {
      path: '/mobile/service/:serviceId/overview',
      name: 'mobileServiceOverview',
      component: ServiceOverviewView,
      meta: mobileRouteMeta,
    },
    {
      path: '/mobile/service/:serviceId/subjects',
      redirect: (to) => ({
        path: '/mobile/service',
        query: { service: String(to.params.serviceId || '') },
      }),
      meta: mobileRouteMeta,
    },
    {
      path: '/mobile/service/:serviceId/activity',
      name: 'mobileServiceActivity',
      component: ServiceActivityView,
      meta: mobileRouteMeta,
    },
    {
      path: '/mobile/service/:serviceId/settings/:section?',
      name: 'mobileServiceSettings',
      component: ServiceSettingsView,
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
