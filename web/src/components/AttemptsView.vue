<script setup lang="ts">
import CheckMark from '@/components/CheckMark.vue'
import { useQuery } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { fetchAttempts } from '@/api/reports'
import DatePicker from '@/components/DatePicker.vue'
import ReviewSwitch from '@/components/ReviewSwitch.vue'
import { Badge } from '@/components/ui/badge'
import { errorMessage } from '@/lib/form'
import { formatDateTime } from '@/lib/format'
import { addDays, todayTashkent } from '@/lib/period'
import { uz } from '@/locales/uz'

const props = defineProps<{ url: string; cacheKey: string; as: 'admin' | 'teacher' }>()
const to = ref(todayTashkent())
const from = ref(addDays(todayTashkent(), -6))
const search = computed(() => `from=${from.value}&to=${to.value}`)
const { data, isPending, error } = useQuery({
  queryKey: ['attempts', props.cacheKey, search],
  queryFn: () => fetchAttempts(`${props.url}?${search.value}`),
})

const num = (v: number | null, d = 2) => (v === null ? '—' : v.toFixed(d))
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-end gap-3">
      <div class="text-muted-foreground flex flex-col gap-1 text-xs">{{ uz.report.from }}<DatePicker v-model="from" /></div>
      <div class="text-muted-foreground flex flex-col gap-1 text-xs">{{ uz.report.to }}<DatePicker v-model="to" /></div>
    </div>
    <p v-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
    <p v-else-if="error" class="text-destructive text-sm" role="alert">{{ errorMessage(error) }}</p>
    <template v-else-if="data">
      <h2 class="text-lg font-semibold">{{ data.student.full_name }} <span class="text-muted-foreground text-sm font-normal">{{ data.student.group_name }}</span></h2>
      <p v-if="!data.days.length" class="text-muted-foreground text-sm">{{ props.as === 'teacher' ? uz.teacher.noReviewAttempts : uz.teacher.noAttempts }}</p>
      <p v-else-if="props.as === 'teacher'" class="text-sm text-yellow-800 dark:text-yellow-300">{{ uz.teacher.review.hint }}</p>
      <section v-for="day in data.days" :key="day.date" class="space-y-2 rounded-lg border p-3">
        <h3 class="font-medium">{{ day.date }} <span class="text-muted-foreground text-sm font-normal">· {{ day.shift }}-{{ uz.report.shift }}</span></h3>
        <div v-for="step in day.steps" :key="step.step" class="space-y-1">
          <p class="text-sm">
            {{ step.step }}-{{ uz.teacher.step }}
            <Badge v-if="step.review === 'pending'" variant="secondary" class="bg-yellow-100 text-yellow-900 dark:bg-yellow-950 dark:text-yellow-300">{{ uz.teacher.review.pending }}</Badge>
            <Badge v-else :variant="step.success ? 'default' : 'secondary'">{{ step.success ? uz.teacher.ok : uz.teacher.failed }}</Badge>
          </p>
          <div class="overflow-x-auto">
            <table class="w-full text-xs">
              <thead>
                <tr class="text-muted-foreground text-left">
                  <th class="py-1 pr-3 font-normal">{{ uz.audit.time }}</th>
                  <th class="pr-3 font-normal">{{ uz.teacher.face }}</th>
                  <th class="pr-3 font-normal">{{ uz.teacher.location }}</th>
                  <th class="pr-3 font-normal">{{ uz.teacher.liveness }}</th>
                  <th class="pr-3 font-normal">{{ uz.teacher.distance }}</th>
                  <th class="pr-3 font-normal">{{ uz.teacher.spoof }}</th>
                  <th class="pr-3 font-normal">{{ uz.teacher.gpsAccuracy }}</th>
                  <th class="pr-3 font-normal">{{ uz.teacher.error }}</th>
                  <th class="pr-3 font-normal">{{ uz.teacher.selfie }}</th>
                  <th class="font-normal">{{ uz.teacher.result }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="a in step.attempts" :key="a.id" class="border-t">
                  <td class="py-1 pr-3 whitespace-nowrap">{{ formatDateTime(a.attempted_at) }}</td>
                  <td class="pr-3"><CheckMark :ok="a.face_verified" :label="`${uz.teacher.face}: ${a.face_verified ? uz.teacher.ok : uz.teacher.failed}`" /></td>
                  <td class="pr-3"><CheckMark :ok="a.location_verified" :label="`${uz.teacher.location}: ${a.location_verified ? uz.teacher.ok : uz.teacher.failed}`" /></td>
                  <td class="pr-3"><CheckMark :ok="a.liveness_verified" :label="`${uz.teacher.liveness}: ${a.liveness_verified ? uz.teacher.ok : uz.teacher.failed}`" /></td>
                  <td class="pr-3">{{ num(a.face_distance) }} / {{ num(a.face_threshold) }}</td>
                  <td class="pr-3">{{ num(a.antispoof_score) }}</td>
                  <td class="pr-3">{{ num(a.gps_accuracy, 0) }} m</td>
                  <td class="pr-3">{{ a.error || '—' }}</td>
                  <td class="pr-3">
                    <a v-if="a.has_selfie" :href="`/api/files/selfies/${a.id}`" target="_blank" rel="noopener" class="underline">{{ uz.teacher.selfie }}</a>
                    <span v-else class="text-muted-foreground">{{ uz.teacher.noSelfie }}</span>
                  </td>
                  <td>
                    <ReviewSwitch v-if="a.review" :attempt-id="a.id" :review="a.review" :as="props.as" />
                    <span v-else class="text-muted-foreground">—</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
