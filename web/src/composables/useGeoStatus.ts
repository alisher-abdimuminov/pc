import { onMounted, ref } from 'vue'
import { secureContextProblem } from '@/lib/verify'

export type GeoState = 'checking' | 'ok' | 'denied' | 'unavailable' | 'timeout' | 'unsupported' | 'insecure'

/** Tracks whether the device can give a position right now (permission + fix), for the status card. */
export function useGeoStatus() {
  const state = ref<GeoState>('checking')
  const accuracy = ref<number | null>(null)

  function refresh() {
    accuracy.value = null
    if (secureContextProblem(window.isSecureContext)) {
      state.value = 'insecure'
      return
    }
    if (!('geolocation' in navigator)) {
      state.value = 'unsupported'
      return
    }
    state.value = 'checking'
    navigator.geolocation.getCurrentPosition(
      (p) => {
        accuracy.value = p.coords.accuracy
        state.value = 'ok'
      },
      (e) => {
        state.value = e.code === 1 ? 'denied' : e.code === 3 ? 'timeout' : 'unavailable'
      },
      { enableHighAccuracy: true, maximumAge: 0, timeout: 15000 },
    )
  }

  onMounted(refresh)
  return { state, accuracy, refresh }
}
