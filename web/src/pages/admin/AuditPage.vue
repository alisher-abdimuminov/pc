<script setup lang="ts">
import { EllipsisVertical, Eye } from '@lucide/vue'
import { useQuery } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { audit, type AuditEntry } from '@/api/admin'
import DateRangePicker from '@/components/DateRangePicker.vue'
import ListFooter from '@/components/ListFooter.vue'
import ListHeader from '@/components/ListHeader.vue'
import SelectField from '@/components/SelectField.vue'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useListQuery } from '@/composables/useListQuery'
import { useRowSelection } from '@/composables/useRowSelection'
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

const filterCount = computed(() => [action, entity, actorId].filter((f) => f.value).length + (from.value || to.value ? 1 : 0))
function clearFilters() {
  action.value = entity.value = from.value = to.value = actorId.value = ''
}
const rows = computed(() => query.data.value?.items ?? [])
const { selected, allSelected, toggleOne, toggleAll } = useRowSelection(rows)
const initials = (name: string) => name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]!.toUpperCase()).join('')

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
    <ListHeader v-model:search="search" :title="uz.audit.title" :total="query.data.value?.total" :filter-count="filterCount" @clear-filters="clearFilters">
      <template #filters>
        <SelectField v-model="action" :options="actionOpts" :placeholder="uz.audit.action" />
        <SelectField v-model="entity" :options="entityOpts" :placeholder="uz.audit.entity" />
        <DateRangePicker v-model:from="from" v-model:to="to" clearable :placeholder="uz.audit.period" />
        <Input v-model="actorId" :placeholder="`${uz.audit.actor} ID`" />
      </template>
    </ListHeader>

    <div class="overflow-x-auto rounded-xl border">
      <Table>
        <TableHeader class="bg-muted/50">
          <TableRow>
            <TableHead class="w-10"><Checkbox :model-value="allSelected" :aria-label="uz.users.selectAll" data-testid="select-all" @update:model-value="toggleAll" /></TableHead>
            <TableHead>{{ uz.audit.actor }}</TableHead>
            <TableHead>{{ uz.audit.action }}</TableHead>
            <TableHead>{{ uz.audit.entity }}</TableHead>
            <TableHead>{{ uz.audit.time }}</TableHead>
            <TableHead class="text-right">{{ uz.common.actions }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="e in rows" :key="e.id" :data-state="selected.has(e.id) ? 'selected' : undefined" data-testid="audit-row">
            <TableCell><Checkbox :model-value="selected.has(e.id)" :aria-label="actorLabel(e)" @update:model-value="toggleOne(e.id, $event)" /></TableCell>
            <TableCell>
              <div class="flex items-center gap-3">
                <Avatar class="size-9"><AvatarFallback>{{ initials(actorLabel(e)) }}</AvatarFallback></Avatar>
                <div class="min-w-0">
                  <p class="truncate font-medium">{{ actorLabel(e) }}</p>
                  <p v-if="e.actor_username" class="text-muted-foreground truncate text-sm">{{ e.actor_username }}</p>
                </div>
              </div>
            </TableCell>
            <TableCell><Badge variant="secondary">{{ actionLabel(e.action) }}</Badge></TableCell>
            <TableCell>{{ entityLabel(e.entity_type) }}</TableCell>
            <TableCell class="whitespace-nowrap">{{ formatDateTime(e.created_at) }}</TableCell>
            <TableCell class="text-right">
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="outline" size="icon" :aria-label="uz.common.actions" data-testid="row-menu"><EllipsisVertical class="size-4" aria-hidden="true" /></Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem @select="selectedId = e.id"><Eye class="size-4" aria-hidden="true" />{{ uz.audit.details }}</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
          <TableRow v-if="query.data.value && !rows.length">
            <TableCell colspan="6" class="text-muted-foreground py-8 text-center">{{ uz.common.empty }}</TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <ListFooter v-if="query.data.value" v-model:page="page" :page-size="pageSize" :total="query.data.value.total" :row-count="rows.length" />

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
