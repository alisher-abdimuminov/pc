<script setup lang="ts">
import { Camera, CircleCheck, LoaderCircle, RotateCcw } from '@lucide/vue'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { evaluateFace, StableTimer } from '@/lib/faceGuard'
import { loadFaceLandmarker, type FaceDetector } from '@/lib/mediapipe'
import { cameraErrorMessage, MAX_CAPTURE_PX, scaleToFit } from '@/lib/verify'
import { uz } from '@/locales/uz'

/**
 * Opens the front camera, waits until exactly one face is centred, the right size and looking straight
 * for STABLE_MS, then takes the photo itself and emits it. No shutter button.
 */
const props = defineProps<{
  /** the camera runs only while this is true */
  active: boolean
  /** while set, the countdown is held and this text is shown (e.g. "waiting for GPS") */
  blockedMessage?: string
  /** the parent is sending the photo */
  busy?: boolean
}>()
const emit = defineEmits<{ capture: [photo: Blob] }>()

type UiState = 'idle' | 'warning' | 'ready' | 'error' | 'done'

const video = ref<HTMLVideoElement>()
const loading = ref(false)
const state = ref<UiState>('idle')
const message = ref<string>(uz.verify.guide.idle)
const progress = ref(0) // 0..1

let stream: MediaStream | null = null
let detector: FaceDetector | null = null
let frame: number | null = null
let lastVideoTime = -1
let captured = false
const timer = new StableTimer()

const stroke = computed(() => `${Math.round(progress.value * 100)} 100`)

function show(next: UiState, text: string) {
  state.value = next
  message.value = text
}

async function start() {
  stop()
  captured = false
  lastVideoTime = -1
  timer.reset()
  progress.value = 0
  show('idle', uz.verify.guide.loading)
  loading.value = true
  try {
    detector ??= await loadFaceLandmarker()
  } catch {
    loading.value = false
    show('error', uz.verify.guide.modelFailed)
    return
  }
  try {
    stream = await navigator.mediaDevices.getUserMedia({
      audio: false,
      video: { facingMode: 'user', width: { ideal: 720 }, height: { ideal: 1280 } },
    })
    const el = video.value
    if (!el) return
    el.srcObject = stream
    await el.play()
    show('idle', uz.verify.guide.idle)
    frame = requestAnimationFrame(tick)
  } catch (e) {
    show('error', cameraErrorMessage((e as DOMException).name))
  } finally {
    loading.value = false
  }
}

function stop() {
  if (frame !== null) cancelAnimationFrame(frame)
  frame = null
  stream?.getTracks().forEach((t) => t.stop())
  stream = null
  timer.reset()
}

function tick() {
  const el = video.value
  if (!el || !detector || captured) return
  frame = requestAnimationFrame(tick)
  if (el.readyState < 2 || el.currentTime === lastVideoTime) return
  lastVideoTime = el.currentTime

  const now = performance.now()
  const result = detector.detectForVideo(el, now)
  const verdict = evaluateFace(result.faceLandmarks, result.facialTransformationMatrixes?.[0]?.data)

  if (!verdict.ok) {
    timer.reset()
    progress.value = 0
    show('warning', uz.verify.guide[verdict.key])
    return
  }
  if (props.blockedMessage) {
    timer.reset()
    progress.value = 0
    show('warning', props.blockedMessage)
    return
  }
  show('ready', uz.verify.guide.hold)
  progress.value = timer.update(true, now)
  if (progress.value >= 1) void takePhoto(el)
}

async function takePhoto(el: HTMLVideoElement) {
  captured = true
  const { width, height } = scaleToFit(el.videoWidth, el.videoHeight, MAX_CAPTURE_PX)
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  canvas.getContext('2d')?.drawImage(el, 0, 0, width, height) // the preview is mirrored by CSS only; the file is not
  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.9))
  stop()
  if (!blob) {
    show('error', uz.common.unexpectedError)
    return
  }
  show('done', uz.verify.guide.checking)
  emit('capture', blob)
}

watch(
  () => props.active,
  (on) => (on ? void start() : stop()),
  { immediate: true },
)
onBeforeUnmount(stop)

defineExpose({ restart: start })
</script>

<template>
  <div class="space-y-3">
    <div class="relative mx-auto aspect-[3/4] w-full max-w-sm overflow-hidden rounded-2xl bg-black">
      <video ref="video" class="size-full -scale-x-100 object-cover" autoplay playsinline muted />

      <!-- oval guide with progress ring -->
      <div class="pointer-events-none absolute inset-0 flex items-center justify-center">
        <div class="relative h-[58%] w-[70%]">
          <div
            class="absolute inset-0 rounded-[45%] border-4 transition-colors"
            :class="{
              'border-white': state === 'idle',
              'border-yellow-400': state === 'warning',
              'border-red-500': state === 'error',
              'border-emerald-500': state === 'ready' || state === 'done',
            }"
          />
          <svg class="absolute inset-0 size-full -rotate-90" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
            <ellipse cx="50" cy="50" rx="46" ry="42" fill="none" pathLength="100" stroke="rgba(255,255,255,0.22)" stroke-width="3" stroke-dasharray="4 3" />
            <ellipse
              v-if="progress > 0 || state === 'done'"
              cx="50" cy="50" rx="46" ry="42" fill="none" pathLength="100"
              stroke="rgb(16 185 129)" stroke-width="4" stroke-linecap="round"
              :stroke-dasharray="state === 'done' ? '100 100' : stroke"
              data-testid="progress-ring"
            />
          </svg>
        </div>
      </div>

      <!-- status -->
      <div class="absolute inset-x-3 bottom-4 rounded-xl bg-black/60 p-3 text-center text-sm text-white backdrop-blur" role="status" aria-live="polite">
        <div class="flex items-center justify-center gap-2">
          <LoaderCircle v-if="loading || busy" class="size-4 animate-spin" aria-hidden="true" />
          <CircleCheck v-else-if="state === 'done'" class="size-4 text-emerald-400" aria-hidden="true" />
          <Camera v-else class="size-4" aria-hidden="true" />
          <span data-testid="face-message">{{ busy ? uz.verify.guide.checking : message }}</span>
        </div>
        <p v-if="state === 'ready' && !busy" class="mt-2 text-xs text-white/70">{{ uz.verify.guide.holdHint }}</p>
      </div>
    </div>

    <Button v-if="state === 'error'" variant="outline" class="w-full" @click="start">
      <RotateCcw class="mr-2 size-4" aria-hidden="true" />
      {{ uz.verify.guide.retry }}
    </Button>
  </div>
</template>
