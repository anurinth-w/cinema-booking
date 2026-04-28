import { ref, onUnmounted } from 'vue'

export function useWebSocket(showtimeId, onMessage) {
  const connected = ref(false)
  let ws = null

  function connect() {
    const protocol = location.protocol === 'https:' ? 'wss' : 'ws'
    const host = import.meta.env.VITE_API_HOST || location.host
    ws = new WebSocket(`${protocol}://${host}/ws/${showtimeId}`)

    ws.onopen = () => {
      connected.value = true
      console.log('[WS] Connected to showtime', showtimeId)
    }

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        onMessage(msg)
      } catch (e) {
        console.error('[WS] Parse error', e)
      }
    }

    ws.onclose = () => {
      connected.value = false
      // Auto-reconnect after 3 seconds
      setTimeout(connect, 3000)
    }

    ws.onerror = (err) => {
      console.error('[WS] Error', err)
      ws.close()
    }
  }

  connect()

  onUnmounted(() => {
    if (ws) ws.close()
  })

  return { connected }
}
