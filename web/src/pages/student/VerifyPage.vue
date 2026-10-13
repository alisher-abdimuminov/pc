<script setup lang="ts">
import BackLink from '@/components/BackLink.vue'
import { Check, X } from '@lucide/vue'
import { useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ApiError } from '@/api/client'
import { student, type AttemptResult } from '@/api/student'
import FaceCapture from '@/components/FaceCapture.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { errorMessage } from '@/lib/form'
import { accuracyLevel, canVerify, failureGuidance, geoErrorMessage, secureContextProblem } from '@/lib/verify'
import { uz } from '@/locales/uz'
import { useSessionStore } from '@/stores/session'

const qc = useQueryClient()
const router = useRouter()
const session = useSessionStore()
const { data: today, isPending } = useQuery({ queryKey: ['student-today'], queryFn: student.today })
const gate = computed(() => canVerify(today.value))
const gpsLimit = computed(() => today.value?.gps_max_accuracy_m || 100)

const insecure = secureContextProblem(window.isSecureContext)

// ---- location ----
interface Fix { latitude: number; longitude: number; accuracy: number }
const fix = ref<Fix | null>(null)
const geoError = ref('')
let watchId: number | null = null

function startLocation() {
  geoError.value = ''
  fix.value = null
  if (!('geolocation' in navigator)) {
    geoError.value = uz.verify.noGeoApi
    return
  }
  stopLocation()
  watchId = navigator.geolocation.watchPosition(
    (p) => {
      const next = { latitude: p.coords.latitude, longitude: p.coords.longitude, accuracy: p.coords.accuracy }
      // keep refining: prefer the most accurate reading seen so far
      if (!fix.value || next.accuracy <= fix.value.accuracy) fix.value = next
    },
    (e) => (geoError.value = geoErrorMessage(e.code)),
    { enableHighAccuracy: true, maximumAge: 0, timeout: 20000 },
  )
}
function stopLocation() {
  if (watchId !== null) navigator.geolocation.clearWatch(watchId)
  watchId = null
}

// ---- automatic capture ----
const sending = ref(false)
const result = ref<AttemptResult | null>(null)
const serverError = ref<{ code: string; message: string } | null>(null)
const captureKey = ref(0) // remounts the camera for the next try

/** FaceCapture holds its countdown while this is non-empty. */
const holdMessage = computed(() => {
  if (!fix.value) return geoError.value || uz.verify.guide.waitLocation
  if (level.value === 'poor') return uz.verify.guide.poorGps
  return ''
})
const cameraActive = computed(() => gate.value.ok && !insecure && !sending.value && !result.value && !serverError.value)

async function send(photo: Blob) {
  if (!fix.value) return
  sending.value = true
  serverError.value = null
  try {
    result.value = await student.attempt(photo, fix.value)
    if (result.value.blocked) {
      // too many face mismatches: the account is blocked and every session is gone, so sign out here too
      qc.clear()
      session.setUser(null)
      await router.replace({ path: '/login', query: { reason: 'inactive' } })
      return
    }
  } catch (e) {
    serverError.value =
      e instanceof ApiError
        ? { code: e.body.code, message: (uz.verify.errorCodes as Record<string, string>)[e.body.code] ?? e.body.message }
        : { code: 'unknown', message: errorMessage(e) }
  } finally {
    sending.value = false
    stopLocation()
    await qc.invalidateQueries({ queryKey: ['student-today'] })
    await qc.invalidateQueries({ queryKey: ['student-month'] })
  }
}

function again() {
  result.value = null
  serverError.value = null
  captureKey.value++
  startLocation() // always take a fresh position for the next try
}

onMounted(() => {
  if (!insecure) startLocation()
})
onBeforeUnmount(stopLocation)

const guidance = computed(() => failureGuidance(result.value?.failures ?? []))
const level = computed(() => (fix.value ? accuracyLevel(fix.value.accuracy, gpsLimit.value) : null))
</script>

<template>
  <div class="space-y-4">
    <BackLink to="/student">{{ uz.verify.back }}</BackLink>
    <h1 class="text-xl font-semibold">{{ uz.verify.title }}</h1>

    <p v-if="insecure" class="rounded-md bg-amber-100 p-3 text-sm text-amber-900 dark:bg-amber-950 dark:text-amber-200" role="alert">{{ insecure }}</p>
    <p v-else-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>

    <!-- not allowed right now -->
    <Card v-else-if="!gate.ok && !result && !serverError">
      <CardContent class="space-y-3 pt-6 text-sm">
        <p>{{ uz.verify.notAvailable }}</p>
        <Button as-child variant="outline"><RouterLink to="/student">{{ uz.verify.back }}</RouterLink></Button>
      </CardContent>
    </Card>

    <!-- result -->
    <Card v-else-if="result" :class="result.success ? 'border-emerald-500' : 'border-rose-500'">
      <CardHeader>
        <CardTitle :class="result.success ? 'text-emerald-700 dark:text-emerald-400' : 'text-rose-700 dark:text-rose-400'" role="status">
          <span class="inline-flex items-center gap-2">
            <Check v-if="result.success" class="size-6" aria-hidden="true" />
            <X v-else class="size-6" aria-hidden="true" />
            {{ result.success ? uz.verify.resultOk : uz.verify.resultFail }}
          </span>
        </CardTitle>
        <p class="text-muted-foreground text-xs">{{ result.shift }}-{{ uz.student.shift }}, {{ result.step }}-{{ uz.student.step }}</p>
      </CardHeader>
      <CardContent class="space-y-3 text-sm">
        <ul v-if="!result.success" class="space-y-2">
          <li v-for="g in guidance" :key="g.code" class="rounded-md border p-2">
            <p class="flex items-center gap-1.5 font-medium"><X class="size-4 shrink-0 text-rose-600 dark:text-rose-400" aria-hidden="true" />{{ g.title }}</p>
            <p class="text-muted-foreground">{{ g.hint }}</p>
          </li>
        </ul>
        <p v-if="result.success">{{ uz.verify.stepsDone }}: {{ result.steps_done }}/3</p>
        <p v-else-if="result.attempts_left > 0">{{ uz.verify.attemptsLeft }}: {{ result.attempts_left }}</p>
        <p v-else class="text-destructive">{{ uz.verify.noAttemptsLeft }}</p>
        <div class="flex gap-2">
          <Button v-if="!result.success && result.attempts_left > 0" @click="again">{{ uz.verify.tryAgain }}</Button>
          <Button as-child :variant="result.success ? 'default' : 'outline'"><RouterLink to="/student">{{ uz.verify.back }}</RouterLink></Button>
        </div>
      </CardContent>
    </Card>

    <!-- rejected by the server before an attempt was recorded -->
    <Card v-else-if="serverError" class="border-amber-500">
      <CardContent class="space-y-3 pt-6 text-sm">
        <p role="alert">{{ serverError.message }}</p>
        <div class="flex gap-2">
          <Button v-if="serverError.code === 'face_unavailable'" @click="again">{{ uz.verify.tryAgain }}</Button>
          <Button as-child variant="outline"><RouterLink to="/student">{{ uz.verify.back }}</RouterLink></Button>
        </div>
      </CardContent>
    </Card>

    <!-- capture flow -->
    <template v-else>
      <p v-if="gate.ok" class="text-muted-foreground text-sm">{{ uz.verify.currentStep }}: {{ gate.step }}-{{ uz.verify.stepOf }}</p>

      <Card>
        <CardHeader><CardTitle class="text-base">{{ uz.verify.locationTitle }}</CardTitle></CardHeader>
        <CardContent class="space-y-2 text-sm">
          <p v-if="geoError" class="text-destructive" role="alert">{{ geoError }}</p>
          <p v-else-if="!fix" class="text-muted-foreground">{{ uz.verify.locationWaiting }}</p>
          <p v-else data-testid="accuracy" :class="level === 'good' ? 'text-emerald-700 dark:text-emerald-400' : 'text-amber-700 dark:text-amber-400'">
            {{ uz.verify.accuracy }}: ±{{ Math.round(fix.accuracy) }} m
            ({{ level === 'good' ? uz.verify.accuracyGood : uz.verify.accuracyPoor }})
          </p>
          <Button v-if="geoError || level === 'poor'" variant="outline" size="sm" @click="startLocation">{{ uz.verify.retryLocation }}</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle class="text-base">{{ uz.verify.cameraTitle }}</CardTitle></CardHeader>
        <CardContent class="space-y-3 text-sm">
          <p class="text-muted-foreground">{{ uz.verify.cameraHint }}</p>
          <FaceCapture :key="captureKey" :active="cameraActive" :blocked-message="holdMessage" :busy="sending" @capture="send" />
        </CardContent>
      </Card>
    </template>
  </div>
</template>
