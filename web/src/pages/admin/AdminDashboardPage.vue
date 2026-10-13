<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { getDashboard } from '@/api/admin'
import BarChart from '@/components/BarChart.vue'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { pct } from '@/lib/format'
import { uz } from '@/locales/uz'

const { data, isPending } = useQuery({ queryKey: ['dashboard'], queryFn: getDashboard, refetchInterval: 60000 })
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-xl font-semibold">{{ uz.nav.dashboard }}</h1>
    <p v-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
    <template v-else-if="data">
      <div class="grid grid-cols-2 gap-3 lg:grid-cols-3">
        <Card v-for="c in [
          { label: uz.stats.students, value: data.students },
          { label: uz.stats.teachers, value: data.teachers },
          { label: uz.stats.groups, value: data.groups },
        ]" :key="c.label">
          <CardHeader><CardTitle class="text-muted-foreground text-sm font-normal">{{ c.label }}</CardTitle></CardHeader>
          <CardContent class="text-3xl font-semibold" data-testid="stat-value">{{ c.value }}</CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader><CardTitle>{{ uz.stats.todayAttendance }}</CardTitle></CardHeader>
        <CardContent>
          <p v-if="data.today.expected === 0" class="text-muted-foreground text-sm">{{ uz.stats.noSchedule }}</p>
          <div v-else class="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <div><p class="text-muted-foreground text-xs">{{ uz.stats.expected }}</p><p class="text-2xl font-semibold">{{ data.today.expected }}</p></div>
            <div><p class="text-xs text-emerald-600">{{ uz.stats.full }}</p><p class="text-2xl font-semibold">{{ data.today.full }} <span class="text-muted-foreground text-sm font-normal">{{ pct(data.today.full, data.today.expected) }}</span></p></div>
            <div><p class="text-xs text-amber-600">{{ uz.stats.partial }}</p><p class="text-2xl font-semibold">{{ data.today.partial }}</p></div>
            <div><p class="text-xs text-rose-600">{{ uz.stats.absent }}</p><p class="text-2xl font-semibold">{{ data.today.absent }}</p></div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>{{ uz.stats.last7 }}</CardTitle></CardHeader>
        <CardContent><BarChart :days="data.last_7_days" /></CardContent>
      </Card>
    </template>
  </div>
</template>
