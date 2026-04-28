<template>
  <div>
    <div v-if="loading" class="loading">Loading...</div>
    <template v-else-if="showtime">
      <div class="header">
        <router-link to="/" class="back">← Back</router-link>
        <div>
          <h2 class="movie-title">{{ showtime.movie_title }}</h2>
          <div class="meta">{{ showtime.hall }} · {{ formatTime(showtime.starts_at) }}</div>
        </div>
        <div class="ws-status" :class="{ connected }">
          {{ connected ? '● Live' : '○ Reconnecting...' }}
        </div>
      </div>

      <!-- Screen indicator -->
      <div class="screen">SCREEN</div>

      <!-- Seat grid -->
      <div class="seat-grid">
        <div v-for="row in rows" :key="row" class="seat-row">
          <span class="row-label">{{ row }}</span>
          <button
            v-for="seat in seatsInRow(row)"
            :key="seat.id"
            class="seat"
            :class="seatClass(seat)"
            :disabled="seat.status !== 'AVAILABLE'"
            @click="selectSeat(seat)"
          >
            {{ seat.number }}
          </button>
        </div>
      </div>

      <!-- Legend -->
      <div class="legend">
        <span class="dot available"></span> Available
        <span class="dot locked"></span> Locked
        <span class="dot booked"></span> Booked
        <span class="dot selected"></span> Selected
      </div>

      <!-- Booking panel -->
      <div v-if="selectedSeat" class="booking-panel">
        <div class="booking-info">
          <strong>Seat {{ selectedSeat.id }}</strong>
          <span class="price">฿{{ selectedSeat.price }}</span>
        </div>

        <div v-if="countdown > 0" class="countdown">
          ⏱ {{ formatCountdown(countdown) }} remaining to confirm
        </div>

        <div class="booking-actions">
          <button v-if="!lockConfirmed" class="btn btn-primary" :disabled="locking" @click="lockSeat">
            {{ locking ? 'Locking...' : 'Reserve Seat' }}
          </button>
          <button v-if="lockConfirmed" class="btn btn-success" :disabled="confirming" @click="confirmBooking">
            {{ confirming ? 'Confirming...' : '✓ Confirm & Pay' }}
          </button>
          <button class="btn btn-ghost" @click="cancelSelection">Cancel</button>
        </div>

        <div v-if="error" class="error">{{ error }}</div>
        <div v-if="successMsg" class="success">{{ successMsg }}</div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useWebSocket } from '../composables/useWebSocket'

const route = useRoute()
const auth = useAuthStore()
const showtimeId = route.params.id

const showtime = ref(null)
const loading = ref(true)
const selectedSeat = ref(null)
const locking = ref(false)
const lockConfirmed = ref(false)
const confirming = ref(false)
const error = ref('')
const successMsg = ref('')
const countdown = ref(0)
let countdownTimer = null

// Fetch showtime
async function fetchShowtime() {
  const res = await auth.apiFetch(`/api/showtimes/${showtimeId}`)
  showtime.value = await res.json()
  loading.value = false
}
fetchShowtime()

// WebSocket real-time updates
const { connected } = useWebSocket(showtimeId, (msg) => {
  if (msg.type === 'seat_update' && msg.showtime_id === showtimeId) {
    if (!showtime.value) return
    const seat = showtime.value.seats.find(s => s.id === msg.seat_id)
    if (seat) {
      seat.status = msg.status
      // If our selected seat was released externally
      if (selectedSeat.value?.id === msg.seat_id && msg.status === 'AVAILABLE' && lockConfirmed.value) {
        resetSelection()
        error.value = 'Your lock expired. Please select the seat again.'
      }
    }
  }
})

const rows = computed(() => {
  if (!showtime.value) return []
  return [...new Set(showtime.value.seats.map(s => s.row))].sort()
})

function seatsInRow(row) {
  return showtime.value.seats.filter(s => s.row === row).sort((a, b) => a.number - b.number)
}

function seatClass(seat) {
  if (selectedSeat.value?.id === seat.id) return 'selected'
  return seat.status.toLowerCase()
}

function selectSeat(seat) {
  if (seat.status !== 'AVAILABLE') return
  resetSelection()
  selectedSeat.value = seat
}

function resetSelection() {
  selectedSeat.value = null
  lockConfirmed.value = false
  error.value = ''
  successMsg.value = ''
  countdown.value = 0
  clearInterval(countdownTimer)
}

function cancelSelection() {
  resetSelection()
}

async function lockSeat() {
  locking.value = true
  error.value = ''
  try {
    const res = await auth.apiFetch('/api/bookings/lock', {
      method: 'POST',
      body: JSON.stringify({ showtime_id: showtimeId, seat_id: selectedSeat.value.id })
    })
    if (!res.ok) {
      const data = await res.json()
      error.value = data.error || 'Failed to lock seat'
      return
    }
    lockConfirmed.value = true
    // Start countdown 5 minutes
    countdown.value = 300
    countdownTimer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        clearInterval(countdownTimer)
        lockConfirmed.value = false
        error.value = 'Lock expired. Please select the seat again.'
      }
    }, 1000)
  } finally {
    locking.value = false
  }
}

async function confirmBooking() {
  confirming.value = true
  error.value = ''
  try {
    const res = await auth.apiFetch('/api/bookings/confirm', {
      method: 'POST',
      body: JSON.stringify({ showtime_id: showtimeId, seat_id: selectedSeat.value.id })
    })
    if (!res.ok) {
      const data = await res.json()
      error.value = data.error || 'Booking failed'
      return
    }
    clearInterval(countdownTimer)
    successMsg.value = `✅ Booking confirmed! Seat ${selectedSeat.value.id} is yours.`
    lockConfirmed.value = false
    countdown.value = 0
    selectedSeat.value = null
  } finally {
    confirming.value = false
  }
}

function formatTime(iso) {
  return new Date(iso).toLocaleString('th-TH', { dateStyle: 'short', timeStyle: 'short' })
}

function formatCountdown(s) {
  const m = Math.floor(s / 60).toString().padStart(2, '0')
  const sec = (s % 60).toString().padStart(2, '0')
  return `${m}:${sec}`
}

onUnmounted(() => clearInterval(countdownTimer))
</script>

<style scoped>
.loading { color: #a0a0b8; }
.header { display: flex; align-items: center; gap: 1.5rem; margin-bottom: 2rem; }
.back { color: #a0a0b8; }
.movie-title { font-size: 1.4rem; font-weight: 700; color: #f0f0ff; }
.meta { color: #8080a0; font-size: .9rem; margin-top: .25rem; }
.ws-status { margin-left: auto; font-size: .85rem; color: #f05050; }
.ws-status.connected { color: #50c080; }

.screen {
  background: linear-gradient(to bottom, #4040a0, #2a2a50);
  text-align: center; padding: .5rem; border-radius: 4px 4px 0 0;
  font-size: .75rem; letter-spacing: .2em; color: #a0a0d0; margin-bottom: 2rem;
}

.seat-grid { display: flex; flex-direction: column; gap: .6rem; align-items: center; }
.seat-row { display: flex; align-items: center; gap: .5rem; }
.row-label { width: 20px; text-align: right; color: #6060a0; font-size: .85rem; }

.seat {
  width: 36px; height: 36px; border-radius: 6px 6px 3px 3px;
  border: 1px solid #3a3a50; font-size: .8rem; cursor: pointer; transition: all .15s;
}
.seat.available { background: #1e2a3a; color: #80b0e0; border-color: #304060; }
.seat.available:hover { background: #2a3a50; border-color: #5080c0; }
.seat.locked { background: #3a2a10; color: #c08040; border-color: #604020; cursor: not-allowed; }
.seat.booked { background: #1a2a1a; color: #406040; border-color: #304030; cursor: not-allowed; }
.seat.selected { background: #f0c040; color: #1a1a24; border-color: #f0c040; font-weight: 700; }

.legend { display: flex; gap: 1.5rem; margin: 1.5rem auto; font-size: .85rem; color: #8080a0; justify-content: center; }
.dot { display: inline-block; width: 12px; height: 12px; border-radius: 3px; margin-right: .35rem; }
.dot.available { background: #1e2a3a; border: 1px solid #304060; }
.dot.locked { background: #3a2a10; }
.dot.booked { background: #1a2a1a; }
.dot.selected { background: #f0c040; }

.booking-panel {
  max-width: 420px; margin: 2rem auto 0;
  background: #1a1a24; border: 1px solid #2a2a38; border-radius: 12px; padding: 1.5rem;
  display: flex; flex-direction: column; gap: 1rem;
}
.booking-info { display: flex; justify-content: space-between; align-items: center; }
.price { font-size: 1.2rem; font-weight: 700; color: #f0c040; }
.countdown { color: #f0a040; font-size: .95rem; text-align: center; }
.booking-actions { display: flex; gap: .75rem; }
.btn { flex: 1; padding: .65rem; border-radius: 8px; border: none; font-size: .95rem; font-weight: 600; cursor: pointer; transition: background .2s; }
.btn-primary { background: #f0c040; color: #1a1a24; }
.btn-primary:hover:not(:disabled) { background: #e0b030; }
.btn-primary:disabled { opacity: .5; cursor: not-allowed; }
.btn-success { background: #40c080; color: #0a1a12; }
.btn-success:hover:not(:disabled) { background: #30b070; }
.btn-success:disabled { opacity: .5; cursor: not-allowed; }
.btn-ghost { background: transparent; border: 1px solid #3a3a50; color: #a0a0b8; }
.btn-ghost:hover { background: #2a2a38; }
.error { color: #f05050; font-size: .9rem; text-align: center; }
.success { color: #50c080; font-size: .95rem; text-align: center; }
</style>
