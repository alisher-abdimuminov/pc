<script setup lang="ts">
import { Building2, EllipsisVertical, Pencil, Plus, RefreshCw, Trash2 } from '@lucide/vue'
import { toTypedSchema } from '@vee-validate/zod'
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { computed, ref } from 'vue'
import { z } from 'zod'
import { faculties, type Faculty, type SyncResult } from '@/api/admin'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import FormField from '@/components/FormField.vue'
import ListFooter from '@/components/ListFooter.vue'
import ListHeader from '@/components/ListHeader.vue'
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
import { useCan } from '@/lib/permissions'
import { uz } from '@/locales/uz'

const qc = useQueryClient()
const can = useCan()
const { search, page, pageSize, query } = useListQuery('faculties', faculties.list)
const rows = computed(() => query.data.value?.items ?? [])
const { selected, allSelected, toggleOne, toggleAll } = useRowSelection(rows)

// ---- HEMIS sync ----
const syncResult = ref<SyncResult | null>(null)
const syncError = ref('')
const sync = useMutation({
  mutationFn: () => faculties.sync(),
  onMutate: () => {
    syncResult.value = null
    syncError.value = ''
  },
  onSuccess: async (r) => {
    syncResult.value = r
    await qc.invalidateQueries({ queryKey: ['faculties'] })
    await qc.invalidateQueries({ queryKey: ['faculty-options'] })
  },
  onError: (e) => (syncError.value = errorMessage(e)),
})

const formOpen = ref(false)
const editing = ref<Faculty | null>(null)
const formError = ref('')
const { defineField, handleSubmit, errors, setErrors, resetForm, isSubmitting } = useForm({
  validationSchema: toTypedSchema(z.object({ name: z.string().trim().min(1, uz.common.required).max(200) })),
})
const [name, nameAttrs] = defineField('name')

function openForm(f: Faculty | null) {
  editing.value = f
  formError.value = ''
  resetForm({ values: { name: f?.name ?? '' } })
  formOpen.value = true
}

const onSubmit = handleSubmit(async (v) => {
  formError.value = ''
  try {
    if (editing.value) await faculties.update(editing.value.id, v)
    else await faculties.create(v)
    await qc.invalidateQueries({ queryKey: ['faculties'] })
    formOpen.value = false
  } catch (e) {
    applyApiError(e, setErrors, formError)
  }
})

const toDelete = ref<Faculty | null>(null)
const deleteError = ref('')
const del = useMutation({
  mutationFn: (id: string) => faculties.remove(id),
  onSuccess: async () => {
    toDelete.value = null
    await qc.invalidateQueries({ queryKey: ['faculties'] })
  },
  onError: (e) => (deleteError.value = errorMessage(e)),
})
</script>

<template>
  <div>
    <ListHeader v-model:search="search" :title="uz.faculties.title" :total="query.data.value?.total">
      <template #actions>
        <Button v-if="can('faculties:create')" variant="outline" :disabled="sync.isPending.value" data-testid="sync-button" @click="sync.mutate()">
          <RefreshCw class="size-4" :class="sync.isPending.value && 'animate-spin'" aria-hidden="true" />
          {{ sync.isPending.value ? uz.faculties.syncing : uz.faculties.sync }}
        </Button>
        <Button v-if="can('faculties:create')" @click="openForm(null)"><Plus class="size-4" aria-hidden="true" />{{ uz.faculties.new }}</Button>
      </template>
    </ListHeader>

    <p v-if="syncError" class="text-destructive mb-4 rounded-md border border-rose-300 p-3 text-sm" role="alert" data-testid="sync-error">{{ syncError }}</p>
    <div v-if="syncResult" class="mb-4 rounded-md border border-emerald-300 bg-emerald-50 p-3 text-sm text-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-200" role="status" data-testid="sync-result">
      <p class="font-medium">{{ uz.faculties.syncDone }}</p>
      <p>
        {{ uz.faculties.syncTotal }}: {{ syncResult.total }} · {{ uz.faculties.syncCreated }}: {{ syncResult.created }} ·
        {{ uz.faculties.syncUpdated }}: {{ syncResult.updated }} · {{ uz.faculties.syncUnchanged }}: {{ syncResult.unchanged }}
      </p>
    </div>

    <div class="overflow-x-auto rounded-xl border">
      <Table>
        <TableHeader class="bg-muted/50">
          <TableRow>
            <TableHead class="w-10"><Checkbox :model-value="allSelected" :aria-label="uz.users.selectAll" data-testid="select-all" @update:model-value="toggleAll" /></TableHead>
            <TableHead>{{ uz.faculties.name }}</TableHead>
            <TableHead>{{ uz.users.addDate }}</TableHead>
            <TableHead class="text-right">{{ uz.common.actions }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="f in rows" :key="f.id" :data-state="selected.has(f.id) ? 'selected' : undefined" data-testid="faculty-row">
            <TableCell><Checkbox :model-value="selected.has(f.id)" :aria-label="f.name" @update:model-value="toggleOne(f.id, $event)" /></TableCell>
            <TableCell>
              <div class="flex items-center gap-3">
                <div class="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-full"><Building2 class="size-4" aria-hidden="true" /></div>
                <div class="min-w-0">
                  <p class="truncate font-medium">{{ f.name }}</p>
                  <p class="text-muted-foreground text-sm">{{ f.hemis_id ? `${uz.users.hemis} ID: ${f.hemis_id}` : '—' }}</p>
                </div>
              </div>
            </TableCell>
            <TableCell class="text-muted-foreground whitespace-nowrap">{{ formatDate(f.created_at) }}</TableCell>
            <TableCell class="text-right">
              <DropdownMenu v-if="can('faculties:update') || can('faculties:delete')">
                <DropdownMenuTrigger as-child>
                  <Button variant="outline" size="icon" :aria-label="uz.common.actions" data-testid="row-menu"><EllipsisVertical class="size-4" aria-hidden="true" /></Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem v-if="can('faculties:update')" @select="openForm(f)"><Pencil class="size-4" aria-hidden="true" />{{ uz.common.edit }}</DropdownMenuItem>
                  <DropdownMenuItem v-if="can('faculties:delete')" variant="destructive" @select="(deleteError = ''), (toDelete = f)"><Trash2 class="size-4" aria-hidden="true" />{{ uz.common.delete }}</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
          <TableRow v-if="query.data.value && !rows.length">
            <TableCell colspan="4" class="text-muted-foreground py-8 text-center">{{ uz.common.empty }}</TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
    <ListFooter v-if="query.data.value" v-model:page="page" :page-size="pageSize" :total="query.data.value.total" :row-count="rows.length" />

    <Dialog v-model:open="formOpen">
      <DialogContent>
        <DialogHeader><DialogTitle>{{ editing ? uz.faculties.edit : uz.faculties.new }}</DialogTitle></DialogHeader>
        <form class="space-y-4" novalidate @submit.prevent="onSubmit">
          <FormField :label="uz.faculties.name" :error="errors.name" for="name">
            <Input id="name" v-model="name" v-bind="nameAttrs" />
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
