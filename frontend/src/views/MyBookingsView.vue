<template>
  <div>
    <h2 class="page-title">My Bookings</h2>
    <div v-if="loading" class="loading">Loading...</div>
    <div v-else-if="bookings.length === 0" class="empty">No bookings yet.</div>
    <div v-else class="booking-list">
      <div v-for="b in bookings" :key="b.id" class="booking-card">
        <div class="movie">{{ b.movie_title }}</div>
        <div class="details">
          <span>🪑 Seat {{ b.seat_id }}</span>
          <span>💰 ฿{{ b.total_price }}</span>
          <span>📅 {{ formatDate(b.booked_at) }}</span>
          <span class="status confirmed">{{ b.status }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const bookings = ref([])
const loading = ref(true)

onMounted(async () => {
  const res = await auth.apiFetch('/api/bookings/mine')
  bookings.value = await res.json()
  loading.value = false
})

function formatDate(iso) {
  return new Date(iso).toLocaleString('th-TH', { dateStyle: 'medium', timeStyle: 'short' })
}
</script>

<style scoped>
.page-title { font-size: 1.5rem; font-weight: 700; color: #f0c040; margin-bottom: 1.5rem; }
.loading, .empty { color: #a0a0b8; }
.booking-list { display: flex; flex-direction: column; gap: 1rem; }
.booking-card {
  background: #1a1a24; border: 1px solid #2a2a38; border-radius: 10px;
  padding: 1.25rem; display: flex; flex-direction: column; gap: .75rem;
}
.movie { font-size: 1.1rem; font-weight: 700; color: #f0f0ff; }
.details { display: flex; gap: 1.5rem; color: #8080a0; font-size: .9rem; flex-wrap: wrap; }
.status.confirmed { color: #50c080; text-transform: capitalize; }
</style>
