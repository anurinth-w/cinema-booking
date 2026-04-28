<template>
  <div>
    <h1 class="page-title">Now Showing</h1>
    <div v-if="!auth.isLoggedIn" class="login-prompt">
      <p>Please login to book tickets.</p>
    </div>
    <div v-if="loading" class="loading">Loading showtimes...</div>
    <div v-else class="showtime-grid">
      <div v-for="st in showtimes" :key="st.id" class="showtime-card">
        <div class="movie-title">{{ st.movie_title }}</div>
        <div class="meta">
          <span>🏛 {{ st.hall }}</span>
          <span>🕐 {{ formatTime(st.starts_at) }}</span>
        </div>
        <div class="seat-summary">
          {{ availableCount(st) }} / {{ st.seats.length }} seats available
        </div>
        <router-link :to="`/showtime/${st.id}`" class="btn-book">
          Select Seats
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const showtimes = ref([])
const loading = ref(false)

onMounted(async () => {
  if (!auth.isLoggedIn) return
  loading.value = true
  try {
    const res = await auth.apiFetch('/api/showtimes')
    showtimes.value = await res.json()
  } finally {
    loading.value = false
  }
})

function formatTime(iso) {
  return new Date(iso).toLocaleString('th-TH', { dateStyle: 'short', timeStyle: 'short' })
}

function availableCount(st) {
  return st.seats.filter(s => s.status === 'AVAILABLE').length
}
</script>

<style scoped>
.page-title { font-size: 1.75rem; font-weight: 700; margin-bottom: 1.5rem; color: #f0c040; }
.login-prompt { text-align: center; padding: 3rem; color: #a0a0b8; }
.loading { color: #a0a0b8; }
.showtime-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1.25rem; }
.showtime-card {
  background: #1a1a24; border: 1px solid #2a2a38; border-radius: 12px;
  padding: 1.5rem; display: flex; flex-direction: column; gap: .75rem;
}
.movie-title { font-size: 1.15rem; font-weight: 700; color: #f0f0ff; }
.meta { display: flex; gap: 1rem; color: #8080a0; font-size: .9rem; }
.seat-summary { color: #60c080; font-size: .9rem; }
.btn-book {
  margin-top: auto; padding: .6rem 1rem; background: #f0c040; color: #1a1a24;
  border-radius: 8px; font-weight: 600; text-align: center; transition: background .2s;
}
.btn-book:hover { background: #e0b030; }
</style>
