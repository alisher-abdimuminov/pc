<script setup lang="ts">
import { Ban, Check, EllipsisVertical, Eye, ListChecks, ShieldAlert, X } from '@lucide/vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { attempts, groups, users, type AdminAttempt } from '@/api/admin'
import DatePicker from '@/components/DatePicker.vue'
import ListFooter from '@/components/ListFooter.vue'
import ListHeader from '@/components/ListHeader.vue'
import SelectField from '@/components/SelectField.vue'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useListQuery } from '@/composables/useListQuery'
import { useRowSelection } from '@/composables/useRowSelection'
import { errorMessage } from '@/lib/form'
import { formatDateTime, formatTime } from '@/lib/format'
import { useCan } from '@/lib/permissions'
import { uz } from '@/locales/uz'

const groupId = ref('')
const result = ref('')
const kind = ref('')
const from = ref('')
const to = ref('')
const extra = computed(() => ({
  group_id: groupId.value || undefined,
  success: result.value || undefined,
  kind: kind.value || undefined,
  from: from.value || undefined,
  to: to.value || undefined,
}))
const { search, page, pageSize, query } = useListQuery('admin-attempts', attempts.list, extra)

const groupOptions = useQuery({ queryKey: ['group-options'], queryFn: () => groups.list({ page_size: 100 }) })
const gOpts = computed(() => (groupOptions.data.value?.items ?? []).map((g) => ({ value: g.id, label: g.name })))
const resultOpts = [
  { value: 'true', label: uz.attemptsPage.success },
  { value: 'false', label: uz.attemptsPage.failed },
]
const kindOpts = Object.entries(uz.attemptsPage.kinds).map(([value, label]) => ({ value, label }))

const rows = computed(() => query.data.value?.items ?? [])
const { selected: checked, allSelected, toggleOne, toggleAll } = useRowSelection(rows)
const filterCount = computed(() => [groupId, result, kind].filter((f) => f.value).length)
function clearFilters() {
  groupId.value = result.value = kind.value = ''
}
const initials = (name: string) => name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]!.toUpperCase()).join('')
const dmy = (d: string) => d.split('-').reverse().join('.')
const checksOf = (a: AdminAttempt) =>
  [['face', a.face_verified, uz.attemptsPage.face], ['loc', a.location_verified, uz.attemptsPage.location], ['live', a.liveness_verified, uz.attemptsPage.liveness]] as const
/** A click on the row opens the details, unless it hit a link, button or checkbox inside it. */
function onRowClick(e: MouseEvent, a: AdminAttempt) {
  if ((e.target as HTMLElement).closest('a, button, [role="checkbox"], [role="menuitem"]')) return
  select(a)
}

const num = (v: number | null, d = 2) => (v === null ? '—' : v.toFixed(d))
const selected = ref<AdminAttempt | null>(null)

const can = useCan()

// ---- block the student (deactivate: ends their sessions) ----
const qc = useQueryClient()
const confirmBlock = ref(false)
const blockError = ref('')
const blockedIds = ref(new Set<string>()) // blocked in this session, until the list refreshes
const isBlocked = (a: AdminAttempt) => !a.student_active || blockedIds.value.has(a.student_id)
const spoofed = (a: AdminAttempt) => !a.liveness_verified
const block = useMutation({
  mutationFn: (id: string) => users.setActive(id, false),
  onSuccess: async (_, id) => {
    blockedIds.value = new Set(blockedIds.value).add(id)
    confirmBlock.value = false
    await qc.invalidateQueries({ queryKey: ['admin-attempts'] })
    await qc.invalidateQueries({ queryKey: ['users'] })
  },
  onError: (e) => (blockError.value = errorMessage(e)),
})
function select(a: AdminAttempt | null) {
  selected.value = a
  confirmBlock.value = false
  blockError.value = ''
}
</script>

<template>
  <div>
    <ListHeader v-model:search="search" :title="uz.attemptsPage.title" :total="query.data.value?.total" :filter-count="filterCount" @clear-filters="clearFilters">
      <template #filters>
        <SelectField v-model="groupId" :options="gOpts" :placeholder="uz.attemptsPage.group" />
        <SelectField v-model="result" :options="resultOpts" :placeholder="uz.attemptsPage.all" />
        <SelectField v-model="kind" :options="kindOpts" :placeholder="uz.attemptsPage.kind" />
      </template>
      <template #actions>
        <div class="flex items-center gap-1.5" data-testid="date-range">
          <DatePicker v-model="from" clearable :placeholder="uz.audit.from" />
          <span class="text-muted-foreground">–</span>
          <DatePicker v-model="to" clearable :placeholder="uz.audit.to" />
        </div>
      </template>
    </ListHeader>

    <div class="overflow-x-auto rounded-xl border">
      <Table>
        <TableHeader class="bg-muted/50">
          <TableRow>
            <TableHead class="w-10"><Checkbox :model-value="allSelected" :aria-label="uz.users.selectAll" data-testid="select-all" @update:model-value="toggleAll" /></TableHead>
            <TableHead>{{ uz.attemptsPage.student }}</TableHead>
            <TableHead>{{ uz.audit.time }}</TableHead>
            <TableHead>{{ uz.attemptsPage.step }}</TableHead>
            <TableHead>{{ uz.attemptsPage.checks }}</TableHead>
            <TableHead>{{ uz.attemptsPage.result }}</TableHead>
            <TableHead class="text-right">{{ uz.common.actions }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow
            v-for="a in rows"
            :key="a.id"
            class="cursor-pointer"
            :data-state="checked.has(a.id) ? 'selected' : undefined"
            data-testid="attempt-row"
            @click="onRowClick($event, a)"
          >
            <TableCell><Checkbox :model-value="checked.has(a.id)" :aria-label="a.student_name" @update:model-value="toggleOne(a.id, $event)" /></TableCell>
            <TableCell>
              <div class="flex items-center gap-3">
                <Avatar class="size-9">
                  <AvatarImage :src="`/api/files/avatars/${a.student_id}`" :alt="a.student_name" />
                  <AvatarFallback>{{ initials(a.student_name || a.student_username) }}</AvatarFallback>
                </Avatar>
                <div class="min-w-0">
                  <RouterLink v-if="can('users:view')" :to="`/admin/users/${a.student_id}`" class="block truncate font-medium hover:underline">{{ a.student_name || a.student_username }}</RouterLink>
                  <p v-else class="truncate font-medium">{{ a.student_name || a.student_username }}</p>
                  <p class="text-muted-foreground truncate text-sm">{{ a.group_name ?? '—' }}</p>
                </div>
              </div>
            </TableCell>
            <TableCell class="whitespace-nowrap">
              <p class="font-medium">{{ formatTime(a.attempted_at) }}</p>
              <p class="text-muted-foreground text-sm">{{ dmy(a.date) }}</p>
            </TableCell>
            <TableCell class="whitespace-nowrap">
              <p>{{ a.step }}-{{ uz.student.step }}</p>
              <p class="text-muted-foreground text-sm">{{ a.shift }}-{{ uz.student.shift }}</p>
            </TableCell>
            <TableCell>
              <div class="flex flex-wrap gap-1.5">
                <span
                  v-for="[key, ok, label] in checksOf(a)"
                  :key="key"
                  class="inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-xs font-medium"
                  :class="ok ? 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300' : 'border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-900 dark:bg-rose-950 dark:text-rose-300'"
                  :data-flag="`${key}:${ok}`"
                  :title="`${label}: ${ok ? uz.teacher.ok : uz.teacher.failed}`"
                >
                  <Check v-if="ok" class="size-3.5" :aria-label="`${label}: ${uz.teacher.ok}`" />
                  <X v-else class="size-3.5" :aria-label="`${label}: ${uz.teacher.failed}`" />
                  {{ label }}
                </span>
              </div>
            </TableCell>
            <TableCell>
              <div class="flex items-center gap-1.5">
                <Badge
                  variant="outline"
                  :class="a.success ? 'border-emerald-600 text-emerald-700 dark:text-emerald-400' : 'border-rose-500 text-rose-600 dark:text-rose-400'"
                  data-testid="result-badge"
                >{{ a.success ? uz.attemptsPage.success : uz.attemptsPage.failed }}</Badge>
                <ShieldAlert v-if="spoofed(a)" class="size-4 text-rose-600 dark:text-rose-400" :aria-label="uz.attemptsPage.spoofSuspected" data-testid="spoof-icon" />
              </div>
            </TableCell>
            <TableCell class="text-right">
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="outline" size="icon" :aria-label="uz.common.actions" data-testid="row-menu"><EllipsisVertical class="size-4" aria-hidden="true" /></Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem @select="select(a)"><Eye class="size-4" aria-hidden="true" />{{ uz.attemptsPage.details }}</DropdownMenuItem>
                  <DropdownMenuItem as-child>
                    <RouterLink :to="`/admin/students/${a.student_id}/attempts`"><ListChecks class="size-4" aria-hidden="true" />{{ uz.attemptsPage.studentAttempts }}</RouterLink>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
          <TableRow v-if="query.data.value && !rows.length">
            <TableCell colspan="7" class="text-muted-foreground py-8 text-center">{{ uz.common.empty }}</TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
    <ListFooter v-if="query.data.value" v-model:page="page" :page-size="pageSize" :total="query.data.value.total" :row-count="rows.length" />

    <Dialog :open="!!selected" @update:open="(v: boolean) => !v && select(null)">
      <DialogContent class="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader><DialogTitle>{{ selected?.student_name }}</DialogTitle></DialogHeader>
        <div v-if="selected" class="grid gap-4 sm:grid-cols-[1fr_12rem]">
          <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
            <dt class="text-muted-foreground">{{ uz.audit.time }}</dt><dd>{{ formatDateTime(selected.attempted_at) }}</dd>
            <dt class="text-muted-foreground">{{ uz.attemptsPage.date }}</dt><dd>{{ selected.date }}</dd>
            <dt class="text-muted-foreground">{{ uz.attemptsPage.step }}</dt><dd>{{ selected.shift }}-{{ uz.student.shift }}, {{ selected.step }}-{{ uz.student.step }}</dd>
            <dt class="text-muted-foreground">{{ uz.attemptsPage.result }}</dt><dd>{{ selected.success ? uz.attemptsPage.success : uz.attemptsPage.failed }}</dd>
            <dt class="text-muted-foreground">{{ uz.teacher.error }}</dt><dd>{{ selected.error || '—' }}</dd>
            <dt class="text-muted-foreground">{{ uz.teacher.distance }}</dt><dd>{{ num(selected.face_distance) }} / {{ num(selected.face_threshold) }}</dd>
            <dt class="text-muted-foreground">{{ uz.teacher.spoof }}</dt><dd>{{ num(selected.antispoof_score) }}</dd>
            <dt class="text-muted-foreground">{{ uz.teacher.gpsAccuracy }}</dt><dd>{{ num(selected.gps_accuracy, 0) }} m</dd>
            <dt class="text-muted-foreground">{{ uz.attemptsPage.coords }}</dt><dd>{{ selected.latitude.toFixed(6) }}, {{ selected.longitude.toFixed(6) }}</dd>
            <dt class="text-muted-foreground">{{ uz.attemptsPage.ip }}</dt><dd>{{ selected.ip || '—' }}</dd>
          </dl>
          <div>
            <img v-if="selected.has_selfie" :src="`/api/files/selfies/${selected.id}`" :alt="uz.attemptsPage.selfie" class="aspect-[3/4] w-full rounded-lg border object-cover" />
            <p v-else class="text-muted-foreground text-sm">{{ uz.attemptsPage.noSelfie }}</p>
          </div>
          <p v-if="spoofed(selected)" class="text-destructive flex items-center gap-1.5 text-sm sm:col-span-2" data-testid="spoof-warning">
            <ShieldAlert class="size-4 shrink-0" aria-hidden="true" />{{ uz.attemptsPage.spoofSuspected }}
          </p>
          <div v-if="can('users:update')" class="space-y-2 sm:col-span-2" data-testid="block-section">
            <p v-if="isBlocked(selected)" class="text-muted-foreground flex items-center gap-1.5 text-sm" data-testid="blocked-note">
              <Ban class="size-4" aria-hidden="true" />{{ uz.attemptsPage.blocked }}
            </p>
            <Button v-else-if="!confirmBlock" variant="destructive" data-testid="block-button" @click="confirmBlock = true">
              <Ban class="size-4" aria-hidden="true" />{{ uz.attemptsPage.block }}
            </Button>
            <div v-else class="space-y-2 rounded-md border border-rose-300 p-3 text-sm dark:border-rose-900" role="alertdialog" data-testid="block-confirm">
              <p class="font-medium">{{ uz.attemptsPage.blockTitle }}</p>
              <p class="text-muted-foreground">{{ uz.attemptsPage.blockHint }}</p>
              <div class="flex gap-2">
                <Button variant="destructive" size="sm" :disabled="block.isPending.value" data-testid="block-confirm-button" @click="block.mutate(selected.student_id)">{{ uz.attemptsPage.block }}</Button>
                <Button variant="outline" size="sm" @click="confirmBlock = false">{{ uz.common.cancel }}</Button>
              </div>
            </div>
            <p v-if="blockError" class="text-destructive text-sm" role="alert">{{ blockError }}</p>
          </div>
          <Button as-child variant="outline" class="sm:col-span-2">
            <RouterLink :to="`/admin/students/${selected.student_id}/attempts`">{{ uz.attemptsPage.studentAttempts }}</RouterLink>
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
