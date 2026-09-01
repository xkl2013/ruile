import { createApp } from 'vue'
import { createPinia } from 'pinia'
import TDesign from 'tdesign-vue-next'
import App from './App.vue'
import router from './router'
import i18n from '@/i18n'
import { initFont } from '@/composables/useFont'
import { initTheme } from '@/composables/useTheme'
import { installAutofillGuard } from '@/utils/disable-autofill'
import { installTDesignIconOfflineGuard } from '@/utils/tdesign-icon-offline'
import 'tdesign-vue-next/es/style/index.css'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import '@/assets/fonts.css'
import '@/assets/theme/theme.css'
import '@/assets/dropdown-menu.less'
import './styles/admin.css'

installTDesignIconOfflineGuard()
initTheme()
initFont()

const app = createApp(App)

app.config.errorHandler = (err, instance, info) => {
  console.error('[睿乐 Admin] Unhandled Vue error:', err, '\nComponent:', instance, '\nInfo:', info)
}

app.use(TDesign)
app.use(createPinia())
app.use(router)
app.use(i18n)

router.isReady().finally(() => {
  app.mount('#app')
  installAutofillGuard()
})
