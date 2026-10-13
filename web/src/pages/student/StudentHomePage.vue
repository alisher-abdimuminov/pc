<script setup lang="ts">
import { CalendarDays, Check, CircleAlert } from '@lucide/vue'
import { useQuery } from '@tanstack/vue-query'
import { useIntervalFn } from '@vueuse/core'
import { computed, ref } from 'vue'
import { student, type TodayStep } from '@/api/student'
import LocationStatusCard from '@/components/LocationStatusCard.vue'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { errorMessage } from '@/lib/form'
import { businessMinutes, clock, hourMarks, layoutSteps, markerTop, stepLook, timelineHeight, type StepLook } from '@/lib/timeline'
import { clockOffset } from '@/lib/timeleft'
import { canVerify } from '@/lib/verify'
import { uz } from '@/locales/uz'

const { data, isPending, error } = useQuery({
  queryKey: ['student-today'],
  queryFn: student.today,
  refetchInterval: 30000,
  refetchOnWindowFocus: true,
})
const profile = useQuery({ queryKey: ['student-profile'], queryFn: student.profile })

// "now" follows the server clock, not the device clock
const tick = ref(Date.now())
useIntervalFn(() => (tick.value = Date.now()), 15000)
const nowMin = computed(() => businessMinutes(tick.value + (data.value ? clockOffset(data.value.server_time) : 0)))

const steps = computed(() => data.value?.steps ?? [])
const rects = computed(() => layoutSteps(steps.value))
const height = computed(() => timelineHeight(steps.value))
const hours = computed(() => hourMarks(steps.value))
const marker = computed(() => markerTop(nowMin.value, steps.value))
const done = computed(() => steps.value.filter((s) => s.state === 'done').length)
const verify = computed(() => canVerify(data.value))
const shiftLine = computed(() =>
  [data.value?.shift ? `${data.value.shift}-${uz.student.shift}` : '', data.value?.schedule?.location_name ?? ''].filter(Boolean).join(' • '),
)

const initials = computed(() =>
  (profile.data.value?.full_name ?? '').split(/\s+/).filter(Boolean).slice(0, 2).map((p) => p[0]!.toUpperCase()).join(''),
)

const banner = computed(() => {
  switch (data.value?.state) {
    case 'no_schedule': return uz.student.noSchedule
    case 'before_start': return uz.student.beforeStart
    case 'finished': return uz.student.finished
    default: return ''
  }
})

const LOOK: Record<StepLook, { card: string; badge: string; label: string }> = {
  done: {
    card: 'border-emerald-400 bg-emerald-50 text-emerald-900 dark:border-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-100',
    badge: 'bg-emerald-200 text-emerald-900 dark:bg-emerald-900 dark:text-emerald-100',
    label: uz.student.done,
  },
  missed: {
    card: 'border-rose-400 border-dashed bg-rose-50 text-rose-800 dark:border-rose-700 dark:bg-rose-950/50 dark:text-rose-200',
    badge: 'bg-rose-200 text-rose-800 dark:bg-rose-900 dark:text-rose-100',
    label: uz.student.missed,
  },
  open: {
    card: 'border-lime-500 border-dashed bg-lime-50 text-lime-950 dark:border-lime-600 dark:bg-lime-950/40 dark:text-lime-100',
    badge: 'bg-lime-300 text-lime-950 dark:bg-lime-700 dark:text-lime-50',
    label: uz.student.open,
  },
  upcoming: {
    card: 'border-border border-dashed bg-muted/50 text-foreground',
    badge: 'bg-muted text-muted-foreground',
    label: uz.student.pending,
  },
}
const look = (s: TodayStep) => stepLook(s)
</script>

<template>
  <!-- profile card -->
  <section class="bg-card flex items-center justify-between gap-4 rounded-3xl border p-5 shadow-xs">
    <div class="min-w-0">
      <h1 v-if="profile.data.value" class="truncate text-xl font-bold" data-testid="student-name">{{ profile.data.value.full_name }}</h1>
      <Skeleton v-else class="h-7 w-48" data-testid="name-skeleton" aria-hidden="true" />
      <p class="text-muted-foreground mt-1 line-clamp-2 text-sm">
        {{ [profile.data.value?.group_name, profile.data.value?.faculty_name].filter(Boolean).join(' • ') }}
      </p>
    </div>
    <img v-if="profile.data.value?.photo_url" :src="profile.data.value.photo_url" :alt="profile.data.value.full_name" class="size-20 shrink-0 rounded-full border-4 border-white object-cover shadow-md" />
    <div v-else class="bg-muted flex size-20 shrink-0 items-center justify-center rounded-full text-xl font-semibold" aria-hidden="true">{{ initials }}</div>
  </section>

  <LocationStatusCard :gps-limit="data?.gps_max_accuracy_m || 100" />

  <p v-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
  <p v-else-if="error" class="text-destructive text-sm" role="alert">{{ errorMessage(error) }}</p>

  <template v-else-if="data">
    <!-- section header -->
    <header class="flex items-center gap-4 px-1 pt-2">
      <div class="bg-primary text-primary-foreground flex size-14 shrink-0 items-center justify-center rounded-2xl">
        <CalendarDays class="size-7" aria-hidden="true" />
      </div>
      <div class="min-w-0 flex-1">
        <h2 class="text-2xl font-bold">{{ uz.student.todayAttendance }}</h2>
        <p class="text-muted-foreground truncate text-sm uppercase" data-testid="shift-line">
          {{ shiftLine }}
        </p>
      </div>
      <span v-if="steps.length" class="bg-muted shrink-0 rounded-full px-4 py-2 text-sm font-semibold" data-testid="progress">{{ done }} / {{ steps.length }}</span>
    </header>

    <p v-if="banner" class="bg-card text-muted-foreground rounded-3xl border p-5 text-sm" data-testid="banner">{{ banner }}</p>
    <p v-if="data.shift_locked" class="rounded-2xl bg-amber-100 p-3 text-sm text-amber-900 dark:bg-amber-950 dark:text-amber-200" role="status">{{ uz.student.shiftLocked }}</p>

    <!-- timeline -->
    <section v-if="steps.length" class="bg-card rounded-3xl border p-4 shadow-xs">
      <div class="relative" :style="{ height: `${height}px`, marginTop: '12px' }" data-testid="timeline">
        <!-- hour labels with dotted guide lines -->
        <div v-for="h in hours" :key="h.label" class="absolute inset-x-0" :style="{ top: `${h.top}px` }" aria-hidden="true">
          <span class="text-muted-foreground absolute -top-2.5 left-0 w-12 text-sm">{{ h.label }}</span>
          <span class="border-border absolute top-0 right-0 left-14 border-t border-dotted" />
        </div>
        <span class="border-border absolute top-0 bottom-0 left-14 border-l border-dotted" aria-hidden="true" />

        <!-- step cards -->
        <ol class="absolute inset-y-0 right-0 left-[4.5rem]">
          <li
            v-for="(s, i) in steps"
            :key="s.step"
            class="absolute inset-x-0 flex flex-col justify-between overflow-hidden rounded-3xl border-2 p-4"
            :class="LOOK[look(s)].card"
            :style="{ top: `${rects[i]!.top + 4}px`, height: `${rects[i]!.height - 8}px` }"
            :aria-current="s.current ? 'step' : undefined"
            :data-look="look(s)"
          >
            <div>
              <div class="flex items-start justify-between gap-2">
                <h3 class="text-lg font-bold">{{ s.step }}-{{ uz.student.step }}</h3>
                <span class="rounded-full px-3 py-1 text-xs font-semibold" :class="LOOK[look(s)].badge">{{ LOOK[look(s)].label }}</span>
              </div>
              <p class="text-sm opacity-80">{{ s.start }} — {{ s.end }}</p>
              <p v-if="s.attempts && look(s) !== 'done'" class="text-xs opacity-80">{{ uz.student.attempts }}: {{ s.attempts }}/{{ s.max_attempts }}</p>
            </div>
            <div class="flex items-end justify-between gap-2 text-sm font-medium">
              <template v-if="look(s) === 'missed'">
                <span>{{ uz.student.timeUp }}</span>
                <CircleAlert class="size-6" aria-hidden="true" />
              </template>
              <template v-else-if="look(s) === 'done'">
                <span class="inline-flex items-center gap-1.5"><Check class="size-4" aria-hidden="true" />{{ uz.student.confirmed }}</span>
              </template>
              <template v-else-if="look(s) === 'open'">
                <Button v-if="verify.ok && verify.step === s.step" as-child class="w-full" data-testid="verify-button">
                  <RouterLink to="/student/verify">{{ uz.verify.start }}</RouterLink>
                </Button>
                <span v-else>{{ uz.student.open }}</span>
              </template>
              <template v-else>
                <span class="opacity-80">{{ s.start }} {{ uz.student.opensAt }}</span>
              </template>
            </div>
          </li>
        </ol>

        <!-- "now" marker -->
        <div v-if="marker !== null" class="pointer-events-none absolute inset-x-0 z-10" :style="{ top: `${marker}px` }" data-testid="now-marker">
          <span class="absolute -top-3.5 left-0 rounded-full bg-rose-500 px-3 py-1 text-sm font-bold text-white">{{ clock(nowMin) }}</span>
          <span class="absolute top-0 right-0 left-12 border-t-2 border-rose-500" />
          <span class="absolute top-0 left-[3.4rem] size-3 -translate-x-1/2 -translate-y-1/2 rounded-full bg-rose-500" />
        </div>
      </div>
    </section>
  </template>
</template>
