<script setup lang="ts">
import { Check, ChevronLeft, ChevronRight, Contrast, X } from '@lucide/vue'
import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import { computed, ref, type Component } from 'vue'
import { student } from '@/api/student'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/lib/form'
import { todayTashkent } from '@/lib/period'
import { uz } from '@/locales/uz'

const month = ref(todayTashkent().slice(0, 7))
const { data, isPending, error } = useQuery({
  queryKey: ['student-month', month],
  queryFn: () => student.month(month.value),
  placeholderData: keepPreviousData,
})

function shift(delta: number) {
  const [y, m] = month.value.split('-').map(Number)
  const d = new Date(Date.UTC(y!, m! - 1 + delta, 1))
  month.value = `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}`
}

const title = computed(() => {
  const [y, m] = month.value.split('-').map(Number)
  return `${uz.student.months[m! - 1]} ${y}`
})

/** Month grid, Monday first; leading blanks are null. */
const grid = computed(() => {
  const dates = data.value?.dates ?? []
  if (!dates.length) return []
  const lead = (new Date(dates[0] + 'T00:00:00Z').getUTCDay() + 6) % 7
  return [...Array<null>(lead).fill(null), ...dates]
})

const SYMBOL: Record<'full' | 'partial' | 'absent', { icon: Component; cls: string; label: string }> = {
  full: { icon: Check, cls: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300', label: uz.report.full },
  partial: { icon: Contrast, cls: 'bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-300', label: uz.report.partial },
  absent: { icon: X, cls: 'bg-rose-100 text-rose-800 dark:bg-rose-950 dark:text-rose-300', label: uz.report.absent },
} as const
function cell(d: string) {
  const c = data.value?.cells[d]
  if (!c || !c.status) return null
  const s = SYMBOL[c.status]
  return { ...s, title: `${s.label}${c.shift ? `, ${c.shift}-${uz.student.shift}` : ''}`, shift: c.shift }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <Button variant="outline" size="sm" :aria-label="uz.student.prevMonth" @click="shift(-1)"><ChevronLeft class="size-4" aria-hidden="true" /></Button>
      <h1 class="text-lg font-semibold" data-testid="month-title">{{ title }}</h1>
      <Button variant="outline" size="sm" :aria-label="uz.student.nextMonth" @click="shift(1)"><ChevronRight class="size-4" aria-hidden="true" /></Button>
    </div>

    <p v-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
    <p v-else-if="error" class="text-destructive text-sm" role="alert">{{ errorMessage(error) }}</p>
    <template v-else-if="data">
      <div class="grid grid-cols-7 gap-1 text-center text-xs" role="grid">
        <div v-for="w in uz.student.weekdays" :key="w" class="text-muted-foreground py-1" role="columnheader">{{ w }}</div>
        <template v-for="(d, i) in grid" :key="i">
          <div v-if="d === null" />
          <div v-else class="bg-muted/30 flex aspect-square flex-col items-center justify-center rounded-md" role="gridcell">
            <span class="text-muted-foreground text-[10px]">{{ Number(d.slice(8)) }}</span>
            <span v-if="cell(d)" class="inline-flex min-w-6 items-center justify-center rounded px-1 text-sm font-semibold" :class="cell(d)!.cls" :title="cell(d)!.title" :aria-label="`${d}: ${cell(d)!.title}`" data-testid="day-mark">
              <component :is="cell(d)!.icon" class="size-4" aria-hidden="true" />
            </span>
          </div>
        </template>
      </div>

      <dl class="grid grid-cols-4 gap-2 text-center text-sm">
        <div><dt class="text-muted-foreground text-xs">{{ uz.report.expected }}</dt><dd class="font-semibold">{{ data.totals.expected }}</dd></div>
        <div><dt class="text-xs text-emerald-700 dark:text-emerald-400">{{ uz.report.full }}</dt><dd class="font-semibold">{{ data.totals.full }}</dd></div>
        <div><dt class="text-xs text-amber-700 dark:text-amber-400">{{ uz.report.partial }}</dt><dd class="font-semibold">{{ data.totals.partial }}</dd></div>
        <div><dt class="text-xs text-rose-700 dark:text-rose-400">{{ uz.report.absent }}</dt><dd class="font-semibold">{{ data.totals.absent }}</dd></div>
      </dl>
      <ul class="text-muted-foreground flex flex-wrap gap-4 text-xs">
        <li class="flex items-center gap-1"><Check class="size-4" aria-hidden="true" /> {{ uz.report.full }}</li>
        <li class="flex items-center gap-1"><Contrast class="size-4" aria-hidden="true" /> {{ uz.report.partial }}</li>
        <li class="flex items-center gap-1"><X class="size-4" aria-hidden="true" /> {{ uz.report.absent }}</li>
      </ul>
    </template>
  </div>
</template>
