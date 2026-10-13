<script setup lang="ts">
import { CalendarDays, EllipsisVertical, Pencil, Plus, Trash2 } from '@lucide/vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { refDebounced } from '@vueuse/core'
import { computed, reactive, ref } from 'vue'
import { groups, locations, schedules, users, type Schedule, type ScheduleInput } from '@/api/admin'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import FormField from '@/components/FormField.vue'
import ListFooter from '@/components/ListFooter.vue'
import ListHeader from '@/components/ListHeader.vue'
import DatePicker from '@/components/DatePicker.vue'
import SelectField from '@/components/SelectField.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useListQuery } from '@/composables/useListQuery'
import { useRowSelection } from '@/composables/useRowSelection'
import { applyApiError, errorMessage } from '@/lib/form'
import { todayTashkent } from '@/lib/period'
import { uz } from '@/locales/uz'

const qc = useQueryClient()
const { search, page, pageSize, query } = useListQuery('schedules', schedules.list)

const dayKeys = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'] as const
const today = todayTashkent()
const dmy = (d: string) => d.split('-').reverse().join('.')
/** Whether the schedule runs now, starts later or is over. */
const stateOf = (s: Schedule) => (s.end_date < today ? 'ended' : s.start_date > today ? 'upcoming' : 'active')
const STATE = {
  active: { label: uz.schedules.statusActive, cls: 'border-emerald-600 text-emerald-700 dark:text-emerald-400' },
  upcoming: { label: uz.schedules.statusUpcoming, cls: 'border-amber-500 text-amber-700 dark:text-amber-400' },
  ended: { label: uz.schedules.statusEnded, cls: 'border-muted-foreground text-muted-foreground' },
} as const
const rows = computed(() => query.data.value?.items ?? [])
const { selected, allSelected, toggleOne, toggleAll } = useRowSelection(rows)

const groupOptions = useQuery({ queryKey: ['group-options'], queryFn: () => groups.list({ page_size: 100 }) })
const locationOptions = useQuery({ queryKey: ['location-options'], queryFn: () => locations.list({ page_size: 100 }) })
const gOpts = computed(() => (groupOptions.data.value?.items ?? []).map((g) => ({ value: g.id, label: g.name })))
const lOpts = computed(() => (locationOptions.data.value?.items ?? []).map((l) => ({ value: l.id, label: l.name })))

// ---- form (plain reactive state; the backend is the source of truth for validation) ----
const formOpen = ref(false)
const editing = ref<Schedule | null>(null)
const formError = ref('')
const fieldErrors = ref<Record<string, string>>({})
const saving = ref(false)
const target = ref<'group' | 'user'>('group')
const form = reactive({
  name: '', start_date: '', end_date: '', group_id: '', user_id: '', location_id: '',
  monday: false, tuesday: false, wednesday: false, thursday: false, friday: false, saturday: false,
})

const studentSearch = ref('')
const studentTerm = refDebounced(studentSearch, 300)
const studentOptions = useQuery({
  queryKey: ['student-options', studentTerm],
  queryFn: () => users.list({ role: 'student', is_active: true, search: studentTerm.value, page_size: 20 }),
  enabled: computed(() => formOpen.value && target.value === 'user'),
})
const sOpts = computed(() => {
  const opts = (studentOptions.data.value?.items ?? []).map((s) => ({ value: s.id, label: `${s.full_name} (${s.group_name ?? '—'})` }))
  if (form.user_id && editing.value?.user_id === form.user_id && !opts.some((o) => o.value === form.user_id)) {
    opts.unshift({ value: form.user_id, label: editing.value.user_name ?? form.user_id })
  }
  return opts
})

function openForm(s: Schedule | null) {
  editing.value = s
  formError.value = ''
  fieldErrors.value = {}
  studentSearch.value = ''
  target.value = s?.user_id ? 'user' : 'group'
  Object.assign(form, {
    name: s?.name ?? '', start_date: s?.start_date ?? '', end_date: s?.end_date ?? '',
    group_id: s?.group_id ?? '', user_id: s?.user_id ?? '', location_id: s?.location_id ?? '',
    monday: s?.monday ?? false, tuesday: s?.tuesday ?? false, wednesday: s?.wednesday ?? false,
    thursday: s?.thursday ?? false, friday: s?.friday ?? false, saturday: s?.saturday ?? false,
  })
  formOpen.value = true
}

function clientValidate(): Record<string, string> {
  const e: Record<string, string> = {}
  if (!form.start_date) e.start_date = uz.common.required
  if (!form.end_date) e.end_date = uz.common.required
  if (form.start_date && form.end_date && form.end_date < form.start_date) e.end_date = uz.schedules.endBeforeStart
  if (target.value === 'group' ? !form.group_id : !form.user_id) e.target = uz.schedules.pickTarget
  if (!dayKeys.some((k) => form[k])) e.weekdays = uz.schedules.pickWeekday
  if (!form.location_id) e.location_id = uz.common.required
  return e
}

async function onSubmit() {
  formError.value = ''
  fieldErrors.value = clientValidate()
  if (Object.keys(fieldErrors.value).length) return
  const body: ScheduleInput = {
    name: form.name.trim(), start_date: form.start_date, end_date: form.end_date,
    group_id: target.value === 'group' ? form.group_id : null,
    user_id: target.value === 'user' ? form.user_id : null,
    location_id: form.location_id,
    monday: form.monday, tuesday: form.tuesday, wednesday: form.wednesday,
    thursday: form.thursday, friday: form.friday, saturday: form.saturday,
  }
  saving.value = true
  try {
    if (editing.value) await schedules.update(editing.value.id, body)
    else await schedules.create(body)
    await qc.invalidateQueries({ queryKey: ['schedules'] })
    formOpen.value = false
  } catch (e) {
    const err = ref('')
    applyApiError(e, (f) => (fieldErrors.value = { ...f, ...(f.group_id || f.user_id ? { target: f.group_id ?? f.user_id ?? '' } : {}) }), err)
    formError.value = err.value
  } finally {
    saving.value = false
  }
}

const toDelete = ref<Schedule | null>(null)
const deleteError = ref('')
const del = useMutation({
  mutationFn: (id: string) => schedules.remove(id),
  onSuccess: async () => {
    toDelete.value = null
    await qc.invalidateQueries({ queryKey: ['schedules'] })
  },
  onError: (e) => (deleteError.value = errorMessage(e)),
})
</script>

<template>
  <div>
    <ListHeader v-model:search="search" :title="uz.schedules.title" :total="query.data.value?.total">
      <template #actions><Button @click="openForm(null)"><Plus class="size-4" aria-hidden="true" />{{ uz.schedules.new }}</Button></template>
    </ListHeader>

    <div class="overflow-x-auto rounded-xl border">
      <Table>
        <TableHeader class="bg-muted/50">
          <TableRow>
            <TableHead class="w-10"><Checkbox :model-value="allSelected" :aria-label="uz.users.selectAll" data-testid="select-all" @update:model-value="toggleAll" /></TableHead>
            <TableHead>{{ uz.schedules.name }}</TableHead>
            <TableHead>{{ uz.schedules.target }}</TableHead>
            <TableHead>{{ uz.schedules.location }}</TableHead>
            <TableHead>{{ uz.schedules.weekdays }}</TableHead>
            <TableHead>{{ uz.schedules.status }}</TableHead>
            <TableHead class="text-right">{{ uz.common.actions }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="s in rows" :key="s.id" :data-state="selected.has(s.id) ? 'selected' : undefined" data-testid="schedule-row">
            <TableCell><Checkbox :model-value="selected.has(s.id)" :aria-label="s.name || s.start_date" @update:model-value="toggleOne(s.id, $event)" /></TableCell>
            <TableCell>
              <div class="flex items-center gap-3">
                <div class="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-full"><CalendarDays class="size-4" aria-hidden="true" /></div>
                <div class="min-w-0">
                  <p class="truncate font-medium">{{ s.name || '—' }}</p>
                  <p class="text-muted-foreground text-sm whitespace-nowrap" data-testid="period">{{ dmy(s.start_date) }} – {{ dmy(s.end_date) }}</p>
                </div>
              </div>
            </TableCell>
            <TableCell>
              <p class="font-medium">{{ s.group_name ?? s.user_name ?? '—' }}</p>
              <p class="text-muted-foreground text-sm">{{ s.group_name ? uz.schedules.group : uz.schedules.student }}</p>
            </TableCell>
            <TableCell>{{ s.location_name ?? '—' }}</TableCell>
            <TableCell>
              <div class="flex gap-1" data-testid="weekdays">
                <span
                  v-for="(label, i) in uz.schedules.days"
                  :key="label"
                  class="flex size-7 items-center justify-center rounded-md border text-[11px] font-medium"
                  :class="s[dayKeys[i]!] ? 'border-primary bg-primary text-primary-foreground' : 'text-muted-foreground/60'"
                  :data-on="s[dayKeys[i]!] || undefined"
                  :title="uz.schedules.daysLong[i]"
                >{{ label.slice(0, 2) }}</span>
              </div>
            </TableCell>
            <TableCell><Badge variant="outline" :class="STATE[stateOf(s)].cls" data-testid="state-badge">{{ STATE[stateOf(s)].label }}</Badge></TableCell>
            <TableCell class="text-right">
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="outline" size="icon" :aria-label="uz.common.actions" data-testid="row-menu"><EllipsisVertical class="size-4" aria-hidden="true" /></Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem @select="openForm(s)"><Pencil class="size-4" aria-hidden="true" />{{ uz.common.edit }}</DropdownMenuItem>
                  <DropdownMenuItem variant="destructive" @select="(deleteError = ''), (toDelete = s)"><Trash2 class="size-4" aria-hidden="true" />{{ uz.common.delete }}</DropdownMenuItem>
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

    <Dialog v-model:open="formOpen">
      <DialogContent class="max-h-[90vh] overflow-y-auto sm:max-w-xl">
        <DialogHeader><DialogTitle>{{ editing ? uz.schedules.edit : uz.schedules.new }}</DialogTitle></DialogHeader>
        <form class="space-y-4" novalidate @submit.prevent="onSubmit">
          <FormField :label="uz.schedules.name" :error="fieldErrors.name" for="sname">
            <Input id="sname" v-model="form.name" />
          </FormField>

          <FormField :label="uz.schedules.target" :error="fieldErrors.target">
            <div class="space-y-2">
              <RadioGroup v-model="target" class="flex gap-4 text-sm" :aria-label="uz.schedules.target">
                <label class="flex items-center gap-2"><RadioGroupItem value="group" id="target-group" /> {{ uz.schedules.group }}</label>
                <label class="flex items-center gap-2"><RadioGroupItem value="user" id="target-user" /> {{ uz.schedules.student }}</label>
              </RadioGroup>
              <SelectField v-if="target === 'group'" v-model="form.group_id" :options="gOpts" :placeholder="uz.common.choose" />
              <template v-else>
                <Input v-model="studentSearch" :placeholder="uz.schedules.filterStudent" />
                <SelectField v-model="form.user_id" :options="sOpts" :placeholder="uz.common.choose" />
              </template>
            </div>
          </FormField>

          <FormField :label="uz.schedules.location" :error="fieldErrors.location_id" for="loc">
            <SelectField id="loc" v-model="form.location_id" :options="lOpts" :placeholder="uz.common.choose" />
          </FormField>

          <div class="grid gap-4 sm:grid-cols-2">
            <FormField :label="uz.schedules.startDate" :error="fieldErrors.start_date" for="sd">
              <DatePicker id="sd" v-model="form.start_date" class="w-full" />
            </FormField>
            <FormField :label="uz.schedules.endDate" :error="fieldErrors.end_date" for="ed">
              <DatePicker id="ed" v-model="form.end_date" class="w-full" />
            </FormField>
          </div>

          <FormField :label="uz.schedules.weekdays" :error="fieldErrors.weekdays">
            <div class="flex flex-wrap gap-3 text-sm">
              <label v-for="(k, i) in dayKeys" :key="k" class="flex items-center gap-2">
                <Checkbox :model-value="form[k]" @update:model-value="form[k] = $event === true" /> {{ uz.schedules.daysLong[i] }}
              </label>
            </div>
          </FormField>

          <p v-if="formError" class="text-destructive text-sm" role="alert">{{ formError }}</p>
          <DialogFooter>
            <Button type="button" variant="outline" @click="formOpen = false">{{ uz.common.cancel }}</Button>
            <Button type="submit" :disabled="saving">{{ uz.common.save }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <ConfirmDialog :open="!!toDelete" :title="toDelete?.name || toDelete?.start_date || ''" :message="deleteError || undefined" :loading="del.isPending.value"
      @update:open="(v: boolean) => !v && (toDelete = null)" @confirm="toDelete && del.mutate(toDelete.id)" />
  </div>
</template>
