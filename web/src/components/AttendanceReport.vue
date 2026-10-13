<script setup lang="ts">
import { Check, ChevronLeft, ChevronRight, Contrast, Download, X } from '@lucide/vue'
import { useQuery, keepPreviousData } from '@tanstack/vue-query'
import { computed, ref, type Component } from 'vue'
import { fetchReport, type DayStatus, type ReportCell } from '@/api/reports'
import DatePicker from '@/components/DatePicker.vue'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/lib/form'
import { periodRange, shiftAnchor, todayTashkent, type PeriodMode } from '@/lib/period'
import { uz } from '@/locales/uz'

const props = defineProps<{
  /** JSON endpoint, e.g. /api/teacher/groups/ID/report */
  url: string
  /** XLSX endpoint with the same query parameters */
  exportUrl: string
  /** extra fixed query parameters (e.g. group_id) */
  query?: Record<string, string | undefined>
  studentLink?: (studentId: string) => string
  showGroup?: boolean
  /** identifies the query cache entry */
  cacheKey: string
}>()

const mode = ref<PeriodMode>('week')
const anchor = ref(todayTashkent())
const customFrom = ref(todayTashkent())
const customTo = ref(todayTashkent())
const range = computed(() => periodRange(mode.value, anchor.value, customFrom.value, customTo.value))

const dmy = (d: string) => d.split('-').reverse().join('.')
const rangeLabel = computed(() => `${dmy(range.value.from)} – ${dmy(range.value.to)}`)

const search = computed(() => {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(props.query ?? {})) if (v) p.set(k, v)
  p.set('from', range.value.from)
  p.set('to', range.value.to)
  return p.toString()
})

const { data, isPending, error } = useQuery({
  queryKey: ['report', props.cacheKey, search],
  queryFn: () => fetchReport(`${props.url}?${search.value}`),
  placeholderData: keepPreviousData,
})
const exportHref = computed(() => `${props.exportUrl}?${search.value}`)

const modeOptions = [
  { value: 'day', label: uz.report.day },
  { value: 'week', label: uz.report.week },
  { value: 'month', label: uz.report.month },
  { value: 'custom', label: uz.report.custom },
]

const SYMBOL: Record<Exclude<DayStatus, ''>, { icon: Component; cls: string; label: string }> = {
  full: { icon: Check, cls: 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300', label: uz.report.full },
  partial: { icon: Contrast, cls: 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-300', label: uz.report.partial },
  absent: { icon: X, cls: 'border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-900 dark:bg-rose-950 dark:text-rose-300', label: uz.report.absent },
}

function cellInfo(c: ReportCell | undefined) {
  if (!c || !c.status) return null
  const s = SYMBOL[c.status]
  const title = `${s.label}${c.shift ? `, ${c.shift}-${uz.report.shift}` : ''}${c.status === 'partial' ? ` (${c.steps}/3)` : ''}`
  return { ...s, title, shift: c.shift }
}

const dow = (d: string) => new Date(`${d}T00:00:00Z`).getUTCDay() // 0 = Sunday
const dayNumber = (d: string) => Number(d.slice(8, 10))
const today = todayTashkent()
/** Days nobody is expected on: Sundays and everything after today. */
const hatched = (d: string) => dow(d) === 0 || d > today
/** The month is named on the first column and on the 1st, so a range that crosses months stays readable. */
const monthHint = (d: string, i: number) => (i === 0 || dayNumber(d) === 1 ? uz.report.months[Number(d.slice(5, 7)) - 1] : '')
const initials = (name: string) => name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]!.toUpperCase()).join('')
const badgeText = (c: ReportCell) => (c.status === 'partial' ? `${c.steps}/3` : SYMBOL[c.status as Exclude<DayStatus, ''>].label)
</script>

<template>
  <div class="space-y-4">
    <div class="bg-card flex flex-wrap items-center gap-3 rounded-xl border p-3" data-testid="report-filters">
      <div role="radiogroup" :aria-label="uz.report.period" class="bg-muted inline-flex rounded-lg p-1">
        <button
          v-for="o in modeOptions"
          :key="o.value"
          type="button"
          role="radio"
          :aria-checked="mode === o.value"
          :data-testid="`mode-${o.value}`"
          class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
          :class="mode === o.value ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
          @click="mode = o.value as PeriodMode"
        >{{ o.label }}</button>
      </div>

      <div v-if="mode !== 'custom'" class="flex items-center gap-1" data-testid="period-nav">
        <Button variant="outline" size="icon" :aria-label="uz.report.prev" data-testid="period-prev" @click="anchor = shiftAnchor(mode, anchor, -1)"><ChevronLeft class="size-4" aria-hidden="true" /></Button>
        <DatePicker v-model="anchor" />
        <Button variant="outline" size="icon" :aria-label="uz.report.next" data-testid="period-next" @click="anchor = shiftAnchor(mode, anchor, 1)"><ChevronRight class="size-4" aria-hidden="true" /></Button>
        <span v-if="mode !== 'day'" class="text-muted-foreground ml-2 text-sm whitespace-nowrap" data-testid="range-label">{{ rangeLabel }}</span>
      </div>
      <div v-else class="flex items-center gap-2">
        <DatePicker v-model="customFrom" :aria-label="uz.report.from" />
        <span class="text-muted-foreground">–</span>
        <DatePicker v-model="customTo" :aria-label="uz.report.to" />
      </div>

      <slot name="filters" />

      <Button as="a" :href="exportHref" variant="outline" class="ml-auto" download><Download class="size-4" aria-hidden="true" />{{ uz.report.exportXlsx }}</Button>
    </div>

    <p v-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
    <p v-else-if="error" class="text-destructive text-sm" role="alert">{{ errorMessage(error) }}</p>
    <template v-else-if="data">
      <p v-if="!data.students.length" class="text-muted-foreground text-sm">{{ uz.report.noStudents }}</p>
      <div v-else class="overflow-x-auto rounded-xl border" data-testid="report-grid">
        <table class="w-full border-collapse text-sm">
          <thead>
            <tr class="bg-muted/50 text-left">
              <th class="bg-muted sticky left-0 z-10 min-w-56 border-r px-4 py-3 font-medium">{{ uz.report.student }}</th>
              <th v-for="(d, i) in data.dates" :key="d" class="min-w-28 border-r px-3 py-3 font-medium" :class="d === today && 'bg-muted'" :data-date="d">
                {{ uz.report.weekdays[dow(d)] }}
                <span v-if="monthHint(d, i)" class="text-muted-foreground ml-1 text-xs font-normal">{{ monthHint(d, i) }}</span>
              </th>
              <th class="min-w-20 border-r px-3 py-3 text-center text-xs font-medium whitespace-nowrap">{{ uz.report.expected }}</th>
              <th class="px-3 py-3 text-center text-xs font-medium" :title="uz.report.percentHint">{{ uz.report.percent }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in data.students" :key="s.id" class="border-t">
              <td class="bg-background sticky left-0 z-10 border-r px-4 py-3">
                <div class="flex items-center gap-3">
                  <Avatar class="size-10">
                    <AvatarImage :src="`/api/files/avatars/${s.id}`" :alt="s.full_name" />
                    <AvatarFallback>{{ initials(s.full_name) }}</AvatarFallback>
                  </Avatar>
                  <div class="min-w-0">
                    <RouterLink v-if="studentLink" :to="studentLink(s.id)" class="block truncate font-medium hover:underline">{{ s.full_name }}</RouterLink>
                    <p v-else class="truncate font-medium">{{ s.full_name }}</p>
                    <p class="text-muted-foreground truncate text-sm">{{ s.group_name ?? '—' }}</p>
                  </div>
                </div>
              </td>
              <td
                v-for="d in data.dates"
                :key="d"
                class="h-24 min-w-28 border-r align-top"
                :class="[hatched(d) ? 'report-hatched' : d === today && 'bg-muted/40']"
                :data-hatched="hatched(d) || undefined"
              >
                <div class="flex flex-col items-start gap-2 p-3">
                  <span class="text-sm">{{ dayNumber(d) }}</span>
                  <span
                    v-if="cellInfo(s.cells[d])"
                    class="inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs font-medium whitespace-nowrap"
                    :class="cellInfo(s.cells[d])!.cls"
                    :title="cellInfo(s.cells[d])!.title"
                    :aria-label="cellInfo(s.cells[d])!.title"
                  >
                    <component :is="cellInfo(s.cells[d])!.icon" class="size-3.5" aria-hidden="true" data-testid="cell-icon" />
                    {{ badgeText(s.cells[d]!) }}
                  </span>
                </div>
              </td>
              <td class="border-r px-3 py-3 text-center">{{ s.totals.expected }}</td>
              <td class="px-3 py-3 text-center font-semibold">{{ s.totals.expected ? `${s.totals.percent}%` : '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <ul class="text-muted-foreground flex flex-wrap gap-4 text-xs">
        <li class="flex items-center gap-1"><Check class="size-4" aria-hidden="true" /> {{ uz.report.full }}</li>
        <li class="flex items-center gap-1"><Contrast class="size-4" aria-hidden="true" /> {{ uz.report.partial }}</li>
        <li class="flex items-center gap-1"><X class="size-4" aria-hidden="true" /> {{ uz.report.absent }}</li>
        <li>{{ uz.report.percentHint }}</li>
      </ul>
    </template>
  </div>
</template>

<style scoped>
/* diagonal stripes for days nobody is expected on (Sundays, the future) */
.report-hatched {
  background-image: repeating-linear-gradient(135deg, transparent 0 7px, var(--border) 7px 8px);
}
</style>
