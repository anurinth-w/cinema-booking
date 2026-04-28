import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { auth, loginWithGoogle, logout } from '../firebase'
import { onAuthStateChanged } from 'firebase/auth'

export const useAuthStore = defineStore('auth', () => {
  const firebaseUser = ref(null)
  const backendUser = ref(null)
  const idToken = ref(null)
  const loading = ref(true)

  const isLoggedIn = computed(() => !!firebaseUser.value)
  const isAdmin = computed(() => backendUser.value?.role === 'admin')

  // Sync Firebase auth state
  onAuthStateChanged(auth, async (user) => {
    firebaseUser.value = user
    if (user) {
      idToken.value = await user.getIdToken()
      await fetchBackendUser()
    } else {
      backendUser.value = null
      idToken.value = null
    }
    loading.value = false
  })

  async function fetchBackendUser() {
    try {
      const res = await apiFetch('/api/me')
      backendUser.value = await res.json()
    } catch (e) {
      console.error('Failed to fetch backend user', e)
    }
  }

  async function refreshToken() {
    if (firebaseUser.value) {
      idToken.value = await firebaseUser.value.getIdToken(true)
    }
  }

  async function apiFetch(path, options = {}) {
    if (!idToken.value) await refreshToken()
    return fetch(path, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${idToken.value}`,
        ...(options.headers || {})
      }
    })
  }

  return {
    firebaseUser, backendUser, idToken, loading,
    isLoggedIn, isAdmin,
    loginWithGoogle, logout, apiFetch, refreshToken
  }
})
