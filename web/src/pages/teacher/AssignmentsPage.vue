<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, reactive, ref } from 'vue'
import { ApiError } from '@/api/client'
import { teacher, type Assignment } from '@/api/teacher'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import DateTimePicker from '@/components/DateTimePicker.vue'
import FormField from '@/components/FormField.vue'
import ListToolbar from '@/components/ListToolbar.vue'
import SelectField from '@/components/SelectField.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useListQuery } from '@/composables/useListQuery'
import { errorMessage } from '@/lib/form'
import { formatDateTime } from '@/lib/format'
import { uz } from '@/locales/uz'

const qc = useQueryClient()
const groupFilter = ref('')
const extra = computed(() => ({ group_id: groupFilter.value || undefined }))
const { search, page, pageSize, query } = useListQuery('teacher-assignments', teacher.assignments, extra)
const groups = useQuery({ queryKey: ['teacher-groups'], queryFn: teacher.groups })
const gOpts = computed(() => (groups.data.value ?? []).map((g) => ({ value: g.id, label: g.name })))

// ---- form ----
const formOpen = ref(false)
const editing = ref<Assignment | null>(null)
const saving = ref(false)
const formError = ref('')
const fieldErrors = ref<Record<string, string>>({})
const form = reactive({ group_id: '', title: '', description: '', deadline: '', remove_file: false })
const file = ref<File | null>(null)
const fileInput = ref<HTMLInputElement>()

/** ISO instant -> "YYYY-MM-DDTHH:mm" in Tashkent time (the DateTimePicker value). */
function toLocalInput(iso: string): string {
  const parts = new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Tashkent', dateStyle: 'short', timeStyle: 'short' }).format(new Date(iso))
  return parts.replace(' ', 'T')
}

function openForm(a: Assignment | null) {
  editing.value = a
  formError.value = ''
  fieldErrors.value = {}
  file.value = null
  if (fileInput.value) fileInput.value.value = ''
  Object.assign(form, {
    group_id: a?.group_id ?? (gOpts.value.length === 1 ? gOpts.value[0]!.value : ''),
    title: a?.title ?? '',
    description: a?.description ?? '',
    deadline: a ? toLocalInput(a.deadline) : '',
    remove_file: false,
  })
  formOpen.value = true
}

function onFile(e: Event) {
  file.value = (e.target as HTMLInputElement).files?.[0] ?? null
}

async function onSubmit() {
  formError.value = ''
  const errs: Record<string, string> = {}
  if (!form.title.trim()) errs.title = uz.common.required
  if (!form.deadline) errs.deadline = uz.common.required
  if (!editing.value && !form.group_id) errs.group_id = uz.common.required
  if (file.value && file.value.size > 20 * 1024 * 1024) errs.file = uz.assignments.fileHint
  fieldErrors.value = errs
  if (Object.keys(errs).length) return

  const fd = new FormData()
  fd.set('group_id', form.group_id)
  fd.set('title', form.title.trim())
  fd.set('description', form.description)
  fd.set('deadline', form.deadline)
  if (form.remove_file) fd.set('remove_file', 'true')
  if (file.value) fd.set('file', file.value)
  saving.value = true
  try {
    await teacher.saveAssignment(editing.value?.id ?? null, fd)
    await qc.invalidateQueries({ queryKey: ['teacher-assignments'] })
    formOpen.value = false
  } catch (e) {
    if (e instanceof ApiError) {
      fieldErrors.value = e.body.fields ?? {}
      formError.value = e.body.message
    } else formError.value = uz.common.unexpectedError
  } finally {
    saving.value = false
  }
}

const toDelete = ref<Assignment | null>(null)
const deleteError = ref('')
const del = useMutation({
  mutationFn: (id: string) => teacher.deleteAssignment(id),
  onSuccess: async () => {
    toDelete.value = null
    await qc.invalidateQueries({ queryKey: ['teacher-assignments'] })
  },
  onError: (e) => (deleteError.value = errorMessage(e)),
})
const isOverdue = (a: Assignment) => new Date(a.deadline) < new Date()
</script>

<template>
  <div>
    <ListToolbar v-model:search="search" :title="uz.assignments.title">
      <template #filters>
        <SelectField v-model="groupFilter" class="w-40" :options="gOpts" :placeholder="uz.assignments.group" />
      </template>
      <template #actions><Button @click="openForm(null)">{{ uz.assignments.new }}</Button></template>
    </ListToolbar>

    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{{ uz.assignments.name }}</TableHead>
          <TableHead>{{ uz.assignments.group }}</TableHead>
          <TableHead>{{ uz.assignments.deadline }}</TableHead>
          <TableHead class="text-right">{{ uz.assignments.submitted }}</TableHead>
          <TableHead class="w-56 text-right">{{ uz.common.actions }}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow v-for="a in query.data.value?.items" :key="a.id">
          <TableCell>
            <RouterLink :to="`/teacher/assignments/${a.id}`" class="font-medium hover:underline">{{ a.title }}</RouterLink>
          </TableCell>
          <TableCell>{{ a.group_name }}</TableCell>
          <TableCell class="whitespace-nowrap">
            {{ formatDateTime(a.deadline) }}
            <span v-if="isOverdue(a)" class="text-muted-foreground text-xs">({{ uz.assignments.overdue }})</span>
          </TableCell>
          <TableCell class="text-right">{{ a.submission_count }} / {{ a.student_count }}</TableCell>
          <TableCell class="space-x-2 text-right">
            <Button variant="outline" size="sm" @click="openForm(a)">{{ uz.common.edit }}</Button>
            <Button variant="outline" size="sm" @click="(deleteError = ''), (toDelete = a)">{{ uz.common.delete }}</Button>
          </TableCell>
        </TableRow>
        <TableRow v-if="query.data.value && !query.data.value.items.length">
          <TableCell colspan="5" class="text-muted-foreground text-center">{{ uz.common.empty }}</TableCell>
        </TableRow>
      </TableBody>
    </Table>
    <PaginationBar v-if="query.data.value" v-model:page="page" :page-size="pageSize" :total="query.data.value.total" />

    <Dialog v-model:open="formOpen">
      <DialogContent class="max-h-[90vh] overflow-y-auto sm:max-w-xl">
        <DialogHeader><DialogTitle>{{ editing ? uz.assignments.edit : uz.assignments.new }}</DialogTitle></DialogHeader>
        <form class="space-y-4" novalidate @submit.prevent="onSubmit">
          <FormField v-if="!editing" :label="uz.assignments.group" :error="fieldErrors.group_id" for="agroup">
            <SelectField id="agroup" v-model="form.group_id" :options="gOpts" :placeholder="uz.common.choose" />
          </FormField>
          <FormField :label="uz.assignments.name" :error="fieldErrors.title" for="atitle">
            <Input id="atitle" v-model="form.title" />
          </FormField>
          <FormField :label="uz.assignments.description" :error="fieldErrors.description" for="adesc">
            <textarea id="adesc" v-model="form.description" rows="4" class="border-input dark:bg-input/30 focus-visible:border-ring focus-visible:ring-ring/50 w-full rounded-lg border bg-transparent px-2.5 py-1.5 text-sm outline-none focus-visible:ring-3" />
          </FormField>
          <FormField :label="uz.assignments.deadline" :error="fieldErrors.deadline" for="adeadline">
            <DateTimePicker id="adeadline" v-model="form.deadline" />
          </FormField>
          <FormField :label="uz.assignments.file" :error="fieldErrors.file" for="afile">
            <input id="afile" ref="fileInput" type="file" class="text-sm" accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.zip,.rar,.jpg,.jpeg,.png" @change="onFile" />
            <p class="text-muted-foreground text-xs">{{ uz.assignments.fileHint }}</p>
            <label v-if="editing?.has_file" class="flex items-center gap-2 pt-1 text-sm">
              <Checkbox :model-value="form.remove_file" @update:model-value="form.remove_file = $event === true" /> {{ uz.assignments.removeFile }}
            </label>
          </FormField>
          <p v-if="formError" class="text-destructive text-sm" role="alert">{{ formError }}</p>
          <DialogFooter>
            <Button type="button" variant="outline" @click="formOpen = false">{{ uz.common.cancel }}</Button>
            <Button type="submit" :disabled="saving">{{ uz.common.save }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <ConfirmDialog :open="!!toDelete" :title="toDelete?.title ?? ''" :message="deleteError || undefined" :loading="del.isPending.value"
      @update:open="(v: boolean) => !v && (toDelete = null)" @confirm="toDelete && del.mutate(toDelete.id)" />
  </div>
</template>
