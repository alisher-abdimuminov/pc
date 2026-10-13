<script setup lang="ts">
import { Badge } from '@/components/ui/badge'
import { uz } from '@/locales/uz'

defineProps<{ status: '' | 'full' | 'partial' | 'absent' }>()
const labels = { full: uz.report.full, partial: uz.report.partial, absent: uz.report.absent } as const
const cls = {
  full: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300',
  partial: 'bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-300',
  absent: 'bg-rose-100 text-rose-800 dark:bg-rose-950 dark:text-rose-300',
} as const
</script>

<template>
  <Badge v-if="status" variant="secondary" :class="cls[status]">{{ labels[status] }}</Badge>
  <span v-else class="text-muted-foreground text-xs">{{ uz.teacher.noScheduleToday }}</span>
</template>
