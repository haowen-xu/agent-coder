import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { createAppRouter } from './router'

const app = createApp(App)
const pinia = createPinia()
const router = createAppRouter(pinia)

app.use(pinia)
app.use(router)

router.isReady().finally(() => {
  app.mount('#app')
})
