<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { teacher } from '@/api/teacher'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { uz } from '@/locales/uz'

const { data, isPending } = useQuery({ queryKey: ['teacher-groups'], queryFn: teacher.groups })
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-xl font-semibold">{{ uz.teacher.groupsTitle }}</h1>
    <p v-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
    <p v-else-if="!data?.length" class="text-muted-foreground text-sm">{{ uz.teacher.noGroups }}</p>
    <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <RouterLink v-for="g in data" :key="g.id" :to="`/teacher/groups/${g.id}`" class="block">
        <Card class="hover:bg-muted/40 h-full transition-colors">
          <CardHeader>
            <CardTitle>{{ g.name }}</CardTitle>
            <p class="text-muted-foreground text-xs">{{ g.faculty_name ?? '—' }} · {{ g.student_count }} {{ uz.teacher.students }}</p>
          </CardHeader>
          <CardContent class="space-y-1 text-sm">
            <template v-if="g.today.expected">
              <p class="text-2xl font-semibold">{{ g.today.attended_percent }}%</p>
              <p class="text-muted-foreground text-xs">{{ uz.teacher.today }}: {{ uz.teacher.attended }}</p>
              <p class="text-xs">
                <span class="text-emerald-700 dark:text-emerald-400">{{ uz.report.full }}: {{ g.today.full }}</span> ·
                <span class="text-amber-700 dark:text-amber-400">{{ uz.report.partial }}: {{ g.today.partial }}</span> ·
                <span class="text-rose-700 dark:text-rose-400">{{ uz.report.absent }}: {{ g.today.absent }}</span>
              </p>
            </template>
            <p v-else class="text-muted-foreground text-xs">{{ uz.teacher.noScheduleToday }}</p>
          </CardContent>
        </Card>
      </RouterLink>
    </div>
  </div>
</template>
