import { VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import { SESSION_EXPIRED_EVENT } from './api/client'
import { router } from './router'
import { useSessionStore } from './stores/session'
import './style.css'

const app = createApp(App).use(createPinia()).use(VueQueryPlugin).use(router)
app.mount('#app')

// a 401 on a protected request means the session ended: drop it and go to the login page
window.addEventListener(SESSION_EXPIRED_EVENT, () => {
  const session = useSessionStore()
  if (!session.user) return
  session.setUser(null)
  void router.replace('/login')
})
