<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { audit, type AuditEntry } from '@/api/admin'
import DatePicker from '@/components/DatePicker.vue'
import ListToolbar from '@/components/ListToolbar.vue'
import SelectField from '@/components/SelectField.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useListQuery } from '@/composables/useListQuery'
import { formatDateTime } from '@/lib/format'
import { uz } from '@/locales/uz'

const action = ref('')
const entity = ref('')
const from = ref('')
const to = ref('')
const actorId = ref('')
const extra = computed(() => ({
  action: action.value || undefined,
  entity_type: entity.value || undefined,
  from: from.value || undefined,
  to: to.value || undefined,
  actor_id: actorId.value || undefined,
}))
const { search, page, pageSize, query } = useListQuery('audit', audit.list, extra)

const actionOpts = Object.entries(uz.audit.actions).map(([value, label]) => ({ value, label }))
const entityOpts = Object.entries(uz.audit.entities).map(([value, label]) => ({ value, label }))
const actionLabel = (a: string) => (uz.audit.actions as Record<string, string>)[a] ?? a
const entityLabel = (e: string) => (uz.audit.entities as Record<string, string>)[e] ?? e
const actorLabel = (e: AuditEntry) => e.actor_name || e.actor_username || uz.audit.system

const selectedId = ref<string | null>(null)
const detail = useQuery({
  queryKey: ['audit-entry', selectedId],
  queryFn: () => audit.get(selectedId.value!),
  enabled: computed(() => !!selectedId.value),
})

const fmt = (v: unknown) => (typeof v === 'string' ? v : JSON.stringify(v))
const diffRows = computed(() => {
  const ch = detail.data.value?.changes ?? {}
  const keys = [...new Set([...Object.keys(ch.before ?? {}), ...Object.keys(ch.after ?? {})])].sort()
  return keys.map((k) => ({
    key: k,
    before: ch.before && k in ch.before ? fmt(ch.before[k]) : null,
    after: ch.after && k in ch.after ? fmt(ch.after[k]) : null,
  }))
})
</script>

<template>
  <div>
    <ListToolbar v-model:search="search" :title="uz.audit.title" />
    <div class="flex flex-wrap items-end gap-2 pb-4">
      <SelectField v-model="action" class="w-48" :options="actionOpts" :placeholder="uz.audit.action" />
      <SelectField v-model="entity" class="w-44" :options="entityOpts" :placeholder="uz.audit.entity" />
      <div class="text-muted-foreground flex flex-col gap-1 text-xs">{{ uz.audit.from }}<DatePicker v-model="from" clearable :placeholder="uz.audit.from" /></div>
      <div class="text-muted-foreground flex flex-col gap-1 text-xs">{{ uz.audit.to }}<DatePicker v-model="to" clearable :placeholder="uz.audit.to" /></div>
      <Input v-model="actorId" class="w-72" :placeholder="`${uz.audit.actor} ID`" />
    </div>

    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{{ uz.audit.time }}</TableHead>
          <TableHead>{{ uz.audit.actor }}</TableHead>
          <TableHead>{{ uz.audit.action }}</TableHead>
          <TableHead>{{ uz.audit.entity }}</TableHead>
          <TableHead class="text-right">{{ uz.common.actions }}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow v-for="e in query.data.value?.items" :key="e.id">
          <TableCell class="whitespace-nowrap">{{ formatDateTime(e.created_at) }}</TableCell>
          <TableCell>{{ actorLabel(e) }}</TableCell>
          <TableCell><Badge variant="secondary">{{ actionLabel(e.action) }}</Badge></TableCell>
          <TableCell>{{ entityLabel(e.entity_type) }}</TableCell>
          <TableCell class="text-right"><Button variant="outline" size="sm" @click="selectedId = e.id">{{ uz.audit.details }}</Button></TableCell>
        </TableRow>
        <TableRow v-if="query.data.value && !query.data.value.items.length">
          <TableCell colspan="5" class="text-muted-foreground text-center">{{ uz.common.empty }}</TableCell>
        </TableRow>
      </TableBody>
    </Table>
    <PaginationBar v-if="query.data.value" v-model:page="page" :page-size="pageSize" :total="query.data.value.total" />

    <Dialog :open="!!selectedId" @update:open="(v: boolean) => !v && (selectedId = null)">
      <DialogContent class="max-h-[90vh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader><DialogTitle>{{ uz.audit.details }}</DialogTitle></DialogHeader>
        <p v-if="detail.isPending.value" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
        <template v-else-if="detail.data.value">
          <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
            <dt class="text-muted-foreground">{{ uz.audit.time }}</dt><dd>{{ formatDateTime(detail.data.value.created_at) }}</dd>
            <dt class="text-muted-foreground">{{ uz.audit.actor }}</dt><dd>{{ actorLabel(detail.data.value) }} ({{ detail.data.value.actor_role || '—' }})</dd>
            <dt class="text-muted-foreground">{{ uz.audit.action }}</dt><dd>{{ actionLabel(detail.data.value.action) }}</dd>
            <dt class="text-muted-foreground">{{ uz.audit.entity }}</dt><dd>{{ entityLabel(detail.data.value.entity_type) }}</dd>
            <dt class="text-muted-foreground">{{ uz.audit.entityId }}</dt><dd class="font-mono text-xs">{{ detail.data.value.entity_id ?? '—' }}</dd>
            <dt class="text-muted-foreground">{{ uz.audit.ip }}</dt><dd>{{ detail.data.value.ip || '—' }}</dd>
          </dl>
          <p v-if="!diffRows.length" class="text-muted-foreground text-sm">{{ uz.audit.noChanges }}</p>
          <Table v-else>
            <TableHeader>
              <TableRow>
                <TableHead>{{ uz.audit.field }}</TableHead>
                <TableHead>{{ uz.audit.before }}</TableHead>
                <TableHead>{{ uz.audit.after }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="r in diffRows" :key="r.key">
                <TableCell class="font-mono text-xs">{{ r.key }}</TableCell>
                <TableCell class="max-w-[16rem] break-words text-rose-700 dark:text-rose-400">{{ r.before ?? '' }}</TableCell>
                <TableCell class="max-w-[16rem] break-words text-emerald-700 dark:text-emerald-400">{{ r.after ?? '' }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </template>
      </DialogContent>
    </Dialog>
  </div>
</template>
