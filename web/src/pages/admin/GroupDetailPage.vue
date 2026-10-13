<script setup lang="ts">
import BackLink from '@/components/BackLink.vue'
import { useQuery } from '@tanstack/vue-query'
import { useRoute } from 'vue-router'
import { groups } from '@/api/admin'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useCan } from '@/lib/permissions'
import { uz } from '@/locales/uz'

const can = useCan()
const route = useRoute()
const id = route.params.id as string
const { data, isPending } = useQuery({ queryKey: ['group', id], queryFn: () => groups.get(id) })
</script>

<template>
  <div class="space-y-4">
    <BackLink to="/admin/groups">{{ uz.groups.title }}</BackLink>
    <p v-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
    <template v-else-if="data">
      <div>
        <h1 class="text-xl font-semibold">{{ data.name }}</h1>
        <p class="text-muted-foreground text-sm">
          {{ uz.groups.faculty }}: {{ data.faculty_name ?? '—' }} · {{ uz.groups.teacher }}: {{ data.teacher_name ?? '—' }}
        </p>
      </div>
      <h2 class="font-medium">{{ uz.groups.studentList }} ({{ data.student_count }})</h2>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{{ uz.users.fullName }}</TableHead>
            <TableHead>{{ uz.users.username }}</TableHead>
            <TableHead>{{ uz.users.status }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="s in data.students" :key="s.id">
            <TableCell>
              <RouterLink v-if="can('users:view')" :to="`/admin/users/${s.id}`" class="hover:underline">{{ s.full_name }}</RouterLink>
              <template v-else>{{ s.full_name }}</template>
            </TableCell>
            <TableCell>{{ s.username || '—' }}</TableCell>
            <TableCell><Badge :variant="s.is_active ? 'default' : 'secondary'">{{ s.is_active ? uz.users.active : uz.users.inactive }}</Badge></TableCell>
          </TableRow>
          <TableRow v-if="!data.students.length">
            <TableCell colspan="3" class="text-muted-foreground text-center">{{ uz.common.empty }}</TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </template>
  </div>
</template>
