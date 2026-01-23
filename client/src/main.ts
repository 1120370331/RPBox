import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'
import 'remixicon/fonts/remixicon.css'
import './styles/variables.css'
import './styles/animations.css'
import './styles/content.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
