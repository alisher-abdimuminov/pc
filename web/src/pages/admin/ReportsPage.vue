<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { groups } from '@/api/admin'
import AttendanceReport from '@/components/AttendanceReport.vue'
import SelectField from '@/components/SelectField.vue'
import { uz } from '@/locales/uz'

const groupId = ref('')
const options = useQuery({ queryKey: ['group-options'], queryFn: () => groups.list({ page_size: 100 }) })
const gOpts = computed(() => (options.data.value?.items ?? []).map((g) => ({ value: g.id, label: g.name })))
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-xl font-semibold">{{ uz.report.title }}</h1>
    <AttendanceReport
      url="/api/admin/reports/attendance"
      export-url="/api/admin/reports/attendance.xlsx"
      :query="{ group_id: groupId || undefined }"
      :cache-key="`admin-${groupId}`"
      :student-link="(sid: string) => `/admin/students/${sid}/attempts`"
      show-group
    >
      <template #filters>
        <SelectField v-model="groupId" class="w-52" :options="gOpts" :placeholder="uz.report.allStudents" />
      </template>
    </AttendanceReport>
  </div>
</template>
