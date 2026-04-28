<template>
  <div>
    <h2 class="page-title">Admin Dashboard</h2>

    <!-- Filters -->
    <div class="filters">
      <input v-model="filters.movie" placeholder="Filter by movie..." class="filter-input" @input="fetchBookings" />
      <input v-model="filters.user" placeholder="Filter by user email..." class="filter-input" @input="fetchBookings" />
      <input v-model="filters.date" type="date" class="filter-input" @change="fetchBookings" />
      <button class="btn-clear" @click="clearFilters">Clear</button>
    </div>

    <!-- Bookings Table -->
    <h3 class="section-title">Bookings ({{ bookings.length }})</h3>
    <div v-if="loadingBookings" class="loading">Loading...</div>
    <table v-else class="data-table">
      <thead>
        <tr>
          <th>Movie</th>
          <th>Seat</th>
          <th>User</th>
          <th>Price</th>
          <th>Date</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="b in bookings" :key="b.id">
          <td>{{ b.movie_title }}</td>
          <td>{{ b.seat_id }}</td>
          <td>{{ b.user_email }}</td>
          <td>฿{{ b.total_price }}</td>
          <td>{{ formatDate(b.booked_at) }}</td>
          <td><span class="badge confirmed">{{ b.status }}</span></td>
        </tr>
        <tr v-if="bookings.length === 0"><td colspan="6" class="empty-row">No bookings found</td></tr>
      </tbody>
    </table>

    <!-- Audit Logs -->
    <h3 class="section-title" style="margin-top: 2.5rem">Audit Logs</h3>
    <div v-if="loadingLogs" class="loading">Loading...</div>
    <div v-else class="log-list">
      <div v-for="log in auditLogs" :key="log.id" class="log-entry">
        <span class="log-event" :class="log.event.replace('.', '-')">{{ log.event }}</span>
        <span class="log-time">{{ formatDate(log.created_at) }}</span>
        <span class="log-payload">{{ JSON.stringify(log.payload) }}</span>
      </div>
      <div v-if="auditLogs.length === 0" class="empty">No logs yet.</div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const bookings = ref([])
const auditLogs = ref([])
const loadingBookings = ref(true)
const loadingLogs = ref(true)
const filters = ref({ movie: '', user: '', date: '' })

onMounted(() => {
  fetchBookings()
  fetchLogs()
})

async function fetchBookings() {
  loadingBookings.value = true
  const params = new URLSearchParams()
  if (filters.value.movie) params.set('movie', filters.value.movie)
  if (filters.value.user) params.set('user', filters.value.user)
  if (filters.value.date) params.set('date', filters.value.date)
  const res = await auth.apiFetch(`/api/admin/bookings?${params}`)
  bookings.value = await res.json()
  loadingBookings.value = false
}

async function fetchLogs() {
  loadingLogs.value = true
  const res = await auth.apiFetch('/api/admin/audit-logs')
  auditLogs.value = await res.json()
  loadingLogs.value = false
}

function clearFilters() {
  filters.value = { movie: '', user: '', date: '' }
  fetchBookings()
}

function formatDate(iso) {
  return new Date(iso).toLocaleString('th-TH', { dateStyle: 'short', timeStyle: 'medium' })
}
</script>

<style scoped>
.page-title { font-size: 1.5rem; font-weight: 700; color: #f0c040; margin-bottom: 1.5rem; }
.section-title { font-size: 1.1rem; font-weight: 600; color: #c0c0e0; margin-bottom: 1rem; }
.filters { display: flex; gap: .75rem; margin-bottom: 1.5rem; flex-wrap: wrap; }
.filter-input {
  padding: .5rem .85rem; background: #1a1a24; border: 1px solid #2a2a38;
  border-radius: 8px; color: #e0e0f0; font-size: .9rem; flex: 1; min-width: 160px;
}
.filter-input:focus { outline: none; border-color: #4040a0; }
.btn-clear {
  padding: .5rem 1rem; background: transparent; border: 1px solid #3a3a50;
  border-radius: 8px; color: #a0a0b8; cursor: pointer; font-size: .9rem;
}
.btn-clear:hover { background: #2a2a38; }
.loading, .empty { color: #a0a0b8; font-size: .9rem; }
.data-table { width: 100%; border-collapse: collapse; font-size: .9rem; }
.data-table th { text-align: left; padding: .6rem 1rem; color: #6060a0; border-bottom: 1px solid #2a2a38; }
.data-table td { padding: .65rem 1rem; border-bottom: 1px solid #1e1e2e; color: #c0c0d8; }
.data-table tr:hover td { background: #1e1e2a; }
.empty-row { text-align: center; color: #6060a0; }
.badge { padding: .2rem .6rem; border-radius: 4px; font-size: .8rem; text-transform: capitalize; }
.badge.confirmed { background: #1a3a24; color: #50c080; }
.log-list { display: flex; flex-direction: column; gap: .5rem; max-height: 400px; overflow-y: auto; }
.log-entry {
  display: flex; gap: 1rem; align-items: baseline;
  background: #1a1a24; border: 1px solid #2a2a38; border-radius: 8px; padding: .6rem 1rem; font-size: .85rem;
}
.log-event { font-weight: 600; min-width: 160px; }
.log-event.booking-completed { color: #50c080; }
.log-event.booking-timeout { color: #f0a040; }
.log-event.seat-released { color: #8080d0; }
.log-time { color: #6060a0; min-width: 140px; }
.log-payload { color: #7070a0; font-family: monospace; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
