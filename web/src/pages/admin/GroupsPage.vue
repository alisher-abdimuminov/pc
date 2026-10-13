<script setup lang="ts">
import { EllipsisVertical, Eye, Pencil, Plus, RefreshCw, Trash2, UsersRound } from '@lucide/vue'
import { toTypedSchema } from '@vee-validate/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { computed, ref } from 'vue'
import { z } from 'zod'
import { faculties, groups, users, type Group, type SyncResult } from '@/api/admin'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import FormField from '@/components/FormField.vue'
import ListFooter from '@/components/ListFooter.vue'
import ListHeader from '@/components/ListHeader.vue'
import SelectField from '@/components/SelectField.vue'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useListQuery } from '@/composables/useListQuery'
import { useRowSelection } from '@/composables/useRowSelection'
import { applyApiError, errorMessage } from '@/lib/form'
import { formatDate } from '@/lib/format'
import { uz } from '@/locales/uz'

const qc = useQueryClient()
const facultyFilter = ref('')
const extra = computed(() => ({ faculty_id: facultyFilter.value || undefined }))
const { search, page, pageSize, query } = useListQuery('groups', groups.list, extra)

const rows = computed(() => query.data.value?.items ?? [])
const { selected, allSelected, toggleOne, toggleAll } = useRowSelection(rows)

const facultyOptions = useQuery({ queryKey: ['faculty-options'], queryFn: () => faculties.list({ page_size: 100 }) })
const teacherOptions = useQuery({
  queryKey: ['teacher-options'],
  queryFn: () => users.list({ role: 'teacher', is_active: true, page_size: 100 }),
})
const fOpts = computed(() => (facultyOptions.data.value?.items ?? []).map((f) => ({ value: f.id, label: f.name })))
const tOpts = computed(() => (teacherOptions.data.value?.items ?? []).map((t) => ({ value: t.id, label: t.full_name })))

// ---- HEMIS sync ----
const syncResult = ref<SyncResult | null>(null)
const syncError = ref('')
const sync = useMutation({
  mutationFn: () => groups.sync(),
  onMutate: () => {
    syncResult.value = null
    syncError.value = ''
  },
  onSuccess: async (r) => {
    syncResult.value = r
    await qc.invalidateQueries({ queryKey: ['groups'] })
    await qc.invalidateQueries({ queryKey: ['group-options'] })
  },
  onError: (e) => (syncError.value = errorMessage(e)),
})

const formOpen = ref(false)
const editing = ref<Group | null>(null)
const formError = ref('')
const { defineField, handleSubmit, errors, setErrors, resetForm, isSubmitting } = useForm({
  validationSchema: toTypedSchema(
    z.object({
      name: z.string().trim().min(1, uz.common.required).max(200),
      faculty_id: z.string().nullable().optional(),
      teacher_id: z.string().nullable().optional(),
    }),
  ),
})
const [name, nameAttrs] = defineField('name')
const [facultyId] = defineField('faculty_id')
const [teacherId] = defineField('teacher_id')

function openForm(g: Group | null) {
  editing.value = g
  formError.value = ''
  resetForm({ values: { name: g?.name ?? '', faculty_id: g?.faculty_id ?? '', teacher_id: g?.teacher_id ?? '' } })
  formOpen.value = true
}

const onSubmit = handleSubmit(async (v) => {
  formError.value = ''
  const body = { name: v.name, faculty_id: v.faculty_id || null, teacher_id: v.teacher_id || null }
  try {
    if (editing.value) await groups.update(editing.value.id, body)
    else await groups.create(body)
    await qc.invalidateQueries({ queryKey: ['groups'] })
    formOpen.value = false
  } catch (e) {
    applyApiError(e, setErrors, formError)
  }
})

const toDelete = ref<Group | null>(null)
const deleteError = ref('')
const del = useMutation({
  mutationFn: (id: string) => groups.remove(id),
  onSuccess: async () => {
    toDelete.value = null
    await qc.invalidateQueries({ queryKey: ['groups'] })
  },
  onError: (e) => (deleteError.value = errorMessage(e)),
})
</script>

<template>
  <div>
    <ListHeader v-model:search="search" :title="uz.groups.title" :total="query.data.value?.total" :filter-count="facultyFilter ? 1 : 0" @clear-filters="facultyFilter = ''">
      <template #filters>
        <SelectField v-model="facultyFilter" :options="fOpts" :placeholder="uz.groups.faculty" />
      </template>
      <template #actions>
        <Button variant="outline" :disabled="sync.isPending.value" data-testid="sync-button" @click="sync.mutate()">
          <RefreshCw class="size-4" :class="sync.isPending.value && 'animate-spin'" aria-hidden="true" />
          {{ sync.isPending.value ? uz.groups.syncing : uz.groups.sync }}
        </Button>
        <Button @click="openForm(null)"><Plus class="size-4" aria-hidden="true" />{{ uz.groups.new }}</Button>
      </template>
    </ListHeader>

    <p v-if="syncError" class="text-destructive mb-4 rounded-md border border-rose-300 p-3 text-sm" role="alert" data-testid="sync-error">{{ syncError }}</p>
    <div v-if="syncResult" class="mb-4 rounded-md border border-emerald-300 bg-emerald-50 p-3 text-sm text-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-200" role="status" data-testid="sync-result">
      <p class="font-medium">{{ uz.groups.syncDone }}</p>
      <p>
        {{ uz.groups.syncTotal }}: {{ syncResult.total }} · {{ uz.groups.syncCreated }}: {{ syncResult.created }} ·
        {{ uz.groups.syncUpdated }}: {{ syncResult.updated }} · {{ uz.groups.syncUnchanged }}: {{ syncResult.unchanged }}
      </p>
    </div>

    <div class="overflow-x-auto rounded-xl border">
      <Table>
        <TableHeader class="bg-muted/50">
          <TableRow>
            <TableHead class="w-10"><Checkbox :model-value="allSelected" :aria-label="uz.users.selectAll" data-testid="select-all" @update:model-value="toggleAll" /></TableHead>
            <TableHead>{{ uz.groups.name }}</TableHead>
            <TableHead>{{ uz.groups.faculty }}</TableHead>
            <TableHead>{{ uz.groups.teacher }}</TableHead>
            <TableHead>{{ uz.users.addDate }}</TableHead>
            <TableHead class="text-right">{{ uz.common.actions }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="g in rows" :key="g.id" :data-state="selected.has(g.id) ? 'selected' : undefined" data-testid="group-row">
            <TableCell><Checkbox :model-value="selected.has(g.id)" :aria-label="g.name" @update:model-value="toggleOne(g.id, $event)" /></TableCell>
            <TableCell>
              <div class="flex items-center gap-3">
                <div class="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-full"><UsersRound class="size-4" aria-hidden="true" /></div>
                <div class="min-w-0">
                  <RouterLink :to="`/admin/groups/${g.id}`" class="block truncate font-medium hover:underline">{{ g.name }}</RouterLink>
                  <p class="text-muted-foreground text-sm">{{ g.student_count }} {{ uz.groups.studentsSuffix }}</p>
                </div>
              </div>
            </TableCell>
            <TableCell>{{ g.faculty_name ?? '—' }}</TableCell>
            <TableCell>{{ g.teacher_name ?? '—' }}</TableCell>
            <TableCell class="text-muted-foreground whitespace-nowrap">{{ formatDate(g.created_at) }}</TableCell>
            <TableCell class="text-right">
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="outline" size="icon" :aria-label="uz.common.actions" data-testid="row-menu"><EllipsisVertical class="size-4" aria-hidden="true" /></Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem as-child>
                    <RouterLink :to="`/admin/groups/${g.id}`"><Eye class="size-4" aria-hidden="true" />{{ uz.common.view }}</RouterLink>
                  </DropdownMenuItem>
                  <DropdownMenuItem @select="openForm(g)"><Pencil class="size-4" aria-hidden="true" />{{ uz.common.edit }}</DropdownMenuItem>
                  <DropdownMenuItem variant="destructive" @select="(deleteError = ''), (toDelete = g)"><Trash2 class="size-4" aria-hidden="true" />{{ uz.common.delete }}</DropdownMenuItem>
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

    <Dialog v-model:open="formOpen">
      <DialogContent>
        <DialogHeader><DialogTitle>{{ editing ? uz.groups.edit : uz.groups.new }}</DialogTitle></DialogHeader>
        <form class="space-y-4" novalidate @submit.prevent="onSubmit">
          <FormField :label="uz.groups.name" :error="errors.name" for="name">
            <Input id="name" v-model="name" v-bind="nameAttrs" />
          </FormField>
          <FormField :label="uz.groups.faculty" :error="errors.faculty_id" for="faculty">
            <SelectField id="faculty" v-model="facultyId" :options="fOpts" :placeholder="uz.common.none" />
          </FormField>
          <FormField :label="uz.groups.teacher" :error="errors.teacher_id" for="teacher">
            <SelectField id="teacher" v-model="teacherId" :options="tOpts" :placeholder="uz.common.none" />
          </FormField>
          <p v-if="formError" class="text-destructive text-sm" role="alert">{{ formError }}</p>
          <DialogFooter>
            <Button type="button" variant="outline" @click="formOpen = false">{{ uz.common.cancel }}</Button>
            <Button type="submit" :disabled="isSubmitting">{{ uz.common.save }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <ConfirmDialog :open="!!toDelete" :title="toDelete?.name ?? ''" :message="deleteError || undefined" :loading="del.isPending.value"
      @update:open="(v: boolean) => !v && (toDelete = null)" @confirm="toDelete && del.mutate(toDelete.id)" />
  </div>
</template>
