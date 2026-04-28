<template>
  <div>
    <nav class="navbar">
      <router-link to="/" class="brand">🎬 Cinema</router-link>
      <div class="nav-links">
        <template v-if="auth.isLoggedIn">
          <router-link to="/my-bookings">My Bookings</router-link>
          <router-link v-if="auth.isAdmin" to="/admin">Admin</router-link>
          <span class="user-name">{{ auth.backendUser?.name }}</span>
          <button class="btn-sm" @click="handleLogout">Logout</button>
        </template>
        <template v-else>
          <button class="btn-sm btn-primary" @click="handleLogin">Login with Google</button>
        </template>
      </div>
    </nav>
    <main class="container">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { useAuthStore } from './stores/auth'
import { useRouter } from 'vue-router'

const auth = useAuthStore()
const router = useRouter()

async function handleLogin() {
  await auth.loginWithGoogle()
  router.push('/')
}

async function handleLogout() {
  await auth.logout()
  router.push('/')
}
</script>

<style>
.navbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 1rem 2rem; background: #1a1a24; border-bottom: 1px solid #2a2a38;
}
.brand { font-size: 1.25rem; font-weight: 700; color: #f0c040; }
.nav-links { display: flex; align-items: center; gap: 1.5rem; }
.nav-links a { color: #a0a0b8; transition: color .2s; }
.nav-links a:hover, .nav-links a.router-link-active { color: #f0c040; }
.user-name { color: #c0c0d8; font-size: .9rem; }
.container { max-width: 1100px; margin: 0 auto; padding: 2rem 1rem; }
.btn-sm { padding: .4rem .9rem; border-radius: 6px; border: 1px solid #3a3a50; background: #2a2a38; color: #e0e0f0; cursor: pointer; font-size: .9rem; }
.btn-sm:hover { background: #3a3a50; }
.btn-primary { background: #f0c040; color: #1a1a24; border-color: #f0c040; font-weight: 600; }
.btn-primary:hover { background: #e0b030; }
</style>
