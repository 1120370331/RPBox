import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { setUnauthorizedHandler } from '@shared/api/request'
import router from './router'
import i18n from './i18n'
import App from './App.vue'
import 'remixicon/fonts/remixicon.css'
import './styles/mobile.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.use(i18n)
app.mount('#app')

// 注入 401 处理：跳转登录页
setUnauthorizedHandler(() => {
  router.replace({ name: 'login' })
})
