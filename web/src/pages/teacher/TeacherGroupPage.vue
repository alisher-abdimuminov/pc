<script setup lang="ts">
import BackLink from '@/components/BackLink.vue'
import { useQuery } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { teacher } from '@/api/teacher'
import AttendanceReport from '@/components/AttendanceReport.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { uz } from '@/locales/uz'

const id = useRoute().params.id as string
const { data, isPending } = useQuery({ queryKey: ['teacher-group', id], queryFn: () => teacher.group(id), refetchInterval: 30000 })
const tab = ref<'today' | 'report'>('today')
const tabs = computed(() => [
  { key: 'today' as const, label: uz.teacher.todayStatus },
  { key: 'report' as const, label: uz.teacher.openReport },
])
</script>

<template>
  <div class="space-y-4">
    <BackLink to="/teacher">{{ uz.teacher.groupsTitle }}</BackLink>
    <p v-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
    <template v-else-if="data">
      <div>
        <h1 class="text-xl font-semibold">{{ data.name }}</h1>
        <p class="text-muted-foreground text-sm">{{ data.faculty_name ?? '—' }} · {{ data.student_count }} {{ uz.teacher.students }}</p>
      </div>
      <div class="flex gap-2" role="tablist">
        <Button v-for="t in tabs" :key="t.key" role="tab" :aria-selected="tab === t.key" :variant="tab === t.key ? 'default' : 'outline'" size="sm" @click="tab = t.key">
          {{ t.label }}
        </Button>
      </div>

      <Table v-if="tab === 'today'">
        <TableHeader>
          <TableRow>
            <TableHead>{{ uz.report.student }}</TableHead>
            <TableHead>{{ uz.teacher.todayStatus }}</TableHead>
            <TableHead>{{ uz.report.shift }}</TableHead>
            <TableHead>{{ uz.teacher.steps }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="s in data.students" :key="s.id">
            <TableCell>
              <RouterLink v-if="s.pending" :to="`/teacher/students/${s.id}?group=${id}`" class="font-medium hover:underline">{{ s.full_name }}</RouterLink>
              <span v-else class="font-medium">{{ s.full_name }}</span>
            </TableCell>
            <TableCell>
              <div class="flex flex-wrap items-center gap-2">
                <StatusBadge :status="s.status" />
                <RouterLink
                  v-if="s.pending"
                  :to="`/teacher/students/${s.id}?group=${id}`"
                  class="rounded-md border border-yellow-400 bg-yellow-100 px-2 py-0.5 text-xs font-medium text-yellow-900 dark:border-yellow-700 dark:bg-yellow-950 dark:text-yellow-300"
                  :title="uz.report.pendingHint"
                  data-testid="pending-badge"
                >{{ uz.report.pending }}: {{ s.pending }}</RouterLink>
              </div>
            </TableCell>
            <TableCell>{{ s.shift || '—' }}</TableCell>
            <TableCell>{{ s.status ? `${s.steps}/3` : '—' }}</TableCell>
          </TableRow>
          <TableRow v-if="!data.students.length">
            <TableCell colspan="4" class="text-muted-foreground text-center">{{ uz.report.noStudents }}</TableCell>
          </TableRow>
        </TableBody>
      </Table>

      <AttendanceReport
        v-else
        :url="`/api/teacher/groups/${id}/report`"
        :export-url="`/api/teacher/groups/${id}/report.xlsx`"
        :cache-key="`teacher-${id}`"
      />
    </template>
  </div>
</template>
