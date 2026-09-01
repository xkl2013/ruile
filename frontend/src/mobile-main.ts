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
import CustomerSpace from '@/views/service/CustomerSpace.vue'

installTDesignIconOfflineGuard()
initTheme()
initFont()

const mobileRouteMeta = { requiresInit: true, requiresAuth: true, mobileEntry: true, serviceTab: 'customers' }

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/mobile/service/customers',
    },
    {
      path: '/mobile',
      redirect: '/mobile/service/customers',
    },
    {
      path: '/mobile/service/customers',
      name: 'mobileServiceCustomers',
      component: CustomerSpace,
      meta: mobileRouteMeta,
    },
    {
      path: '/mobile/service/customers/:subjectId',
      name: 'mobileServiceCustomerDetail',
      component: CustomerSpace,
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
