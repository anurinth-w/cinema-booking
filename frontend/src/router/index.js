import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  { path: '/', component: () => import('../views/HomeView.vue') },
  { path: '/showtime/:id', component: () => import('../views/SeatMapView.vue'), meta: { requiresAuth: true } },
  { path: '/my-bookings', component: () => import('../views/MyBookingsView.vue'), meta: { requiresAuth: true } },
  { path: '/admin', component: () => import('../views/AdminView.vue'), meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/:pathMatch(.*)*', redirect: '/' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  // Wait for Firebase to resolve
  if (auth.loading) {
    await new Promise(resolve => {
      const unwatch = setInterval(() => {
        if (!auth.loading) { clearInterval(unwatch); resolve() }
      }, 50)
    })
  }
  if (to.meta.requiresAuth && !auth.isLoggedIn) return '/'
  if (to.meta.requiresAdmin && !auth.isAdmin) return '/'
})

export default router
