<script setup lang="ts">
import { LoaderCircle, MapPin, MapPinOff, RefreshCw } from '@lucide/vue'
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import { useGeoStatus } from '@/composables/useGeoStatus'
import { uz } from '@/locales/uz'

const props = defineProps<{ gpsLimit: number }>()
const { state, accuracy, refresh } = useGeoStatus()

const failed = computed(() => !['checking', 'ok'].includes(state.value))
const title = computed(() => (state.value === 'checking' ? uz.student.loc.checking : state.value === 'ok' ? uz.student.loc.ok : uz.student.loc.failed))
const hint = computed(() => {
  switch (state.value) {
    case 'checking': return uz.student.loc.checkingHint
    case 'ok': {
      const m = Math.round(accuracy.value ?? 0)
      return `${uz.student.loc.okHint}: ±${m} m${(accuracy.value ?? 0) > props.gpsLimit ? ` — ${uz.student.loc.poorHint}` : ''}`
    }
    default: return uz.student.loc[state.value]
  }
})
const poor = computed(() => state.value === 'ok' && (accuracy.value ?? 0) > props.gpsLimit)
</script>

<template>
  <section class="bg-card flex items-center gap-4 rounded-3xl border p-4 shadow-xs" aria-live="polite" data-testid="location-card">
    <div
      class="flex size-14 shrink-0 items-center justify-center rounded-2xl"
      :class="failed ? 'bg-rose-100 text-rose-600 dark:bg-rose-950 dark:text-rose-400' : poor ? 'bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-400' : state === 'ok' ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400' : 'bg-muted text-muted-foreground'"
    >
      <LoaderCircle v-if="state === 'checking'" class="size-7 animate-spin" aria-hidden="true" />
      <MapPinOff v-else-if="failed" class="size-7" aria-hidden="true" />
      <MapPin v-else class="size-7" aria-hidden="true" />
    </div>
    <div class="min-w-0 flex-1">
      <p class="font-semibold">{{ title }}</p>
      <p class="text-muted-foreground text-sm leading-snug">{{ hint }}</p>
    </div>
    <Button variant="secondary" size="icon" class="size-11 shrink-0 rounded-full" :aria-label="uz.student.loc.refresh" :disabled="state === 'checking'" @click="refresh">
      <RefreshCw class="size-5" aria-hidden="true" />
    </Button>
  </section>
</template>
