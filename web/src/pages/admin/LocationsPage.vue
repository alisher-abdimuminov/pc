<script setup lang="ts">
import { EllipsisVertical, ExternalLink, Map, MapPin, Pencil, Plus, Trash2 } from '@lucide/vue'
import { toTypedSchema } from '@vee-validate/zod'
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { computed, ref } from 'vue'
import { z } from 'zod'
import { locations, type Location } from '@/api/admin'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import FormField from '@/components/FormField.vue'
import ListFooter from '@/components/ListFooter.vue'
import ListHeader from '@/components/ListHeader.vue'
import LocationMap from '@/components/LocationMap.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useListQuery } from '@/composables/useListQuery'
import { useRowSelection } from '@/composables/useRowSelection'
import { applyApiError, errorMessage } from '@/lib/form'
import { formatDate, parsePoints } from '@/lib/format'
import { useCan } from '@/lib/permissions'
import { uz } from '@/locales/uz'

const qc = useQueryClient()
const can = useCan()
const { search, page, pageSize, query } = useListQuery('locations', locations.list)

const formOpen = ref(false)
const editing = ref<Location | null>(null)
const formError = ref('')
const points = ref<string[]>([])
const pointsError = ref('')
const preview = computed(() => parsePoints(points.value))

const { defineField, handleSubmit, errors, setErrors, resetForm, isSubmitting } = useForm({
  validationSchema: toTypedSchema(
    z.object({
      name: z.string().trim().min(1, uz.common.required).max(200),
      tolerance_meters: z.coerce.number({ message: uz.common.required }).int().min(0).max(1000),
      location_url: z.string().trim().max(500).optional(),
    }),
  ),
})
const [name, nameAttrs] = defineField('name')
const [toleranceRaw, toleranceAttrs] = defineField('tolerance_meters')
const tolerance = computed({
  get: () => toleranceRaw.value as string | number | undefined,
  set: (v) => (toleranceRaw.value = v),
})
const [url, urlAttrs] = defineField('location_url')

function openForm(l: Location | null) {
  editing.value = l
  formError.value = ''
  pointsError.value = ''
  points.value = l ? l.polygon.map(([a, b]) => `${a},${b}`) : ['', '', '', '']
  resetForm({ values: { name: l?.name ?? '', tolerance_meters: l?.tolerance_meters ?? 30, location_url: l?.location_url ?? '' } })
  formOpen.value = true
}

const addPoint = () => points.value.push('')
const removePoint = (i: number) => points.value.splice(i, 1)

const onSubmit = handleSubmit(async (v) => {
  formError.value = ''
  pointsError.value = ''
  const rows = points.value.map((p) => p.trim())
  const bad = rows.findIndex((r) => parsePoints([r]).length !== 1)
  if (rows.length < 3) pointsError.value = uz.locations.minPoints
  else if (bad >= 0) pointsError.value = `${bad + 1}-nuqta noto'g'ri. ${uz.locations.pointFormat}`
  if (pointsError.value) return
  try {
    const body = { name: v.name, points: rows, tolerance_meters: v.tolerance_meters, location_url: v.location_url ?? '' }
    if (editing.value) await locations.update(editing.value.id, body)
    else await locations.create(body)
    await qc.invalidateQueries({ queryKey: ['locations'] })
    formOpen.value = false
  } catch (e) {
    applyApiError(e, (f) => {
      const { points: p, ...rest } = f
      if (p) pointsError.value = p
      setErrors(rest)
    }, formError)
  }
})

const toDelete = ref<Location | null>(null)
const deleteError = ref('')
const del = useMutation({
  mutationFn: (id: string) => locations.remove(id),
  onSuccess: async () => {
    toDelete.value = null
    await qc.invalidateQueries({ queryKey: ['locations'] })
  },
  onError: (e) => (deleteError.value = errorMessage(e)),
})
const viewing = ref<Location | null>(null)
const rows = computed(() => query.data.value?.items ?? [])
const { selected, allSelected, toggleOne, toggleAll } = useRowSelection(rows)
</script>

<template>
  <div>
    <ListHeader v-model:search="search" :title="uz.locations.title" :total="query.data.value?.total">
      <template #actions><Button v-if="can('locations:create')" @click="openForm(null)"><Plus class="size-4" aria-hidden="true" />{{ uz.locations.new }}</Button></template>
    </ListHeader>

    <div class="overflow-x-auto rounded-xl border">
      <Table>
        <TableHeader class="bg-muted/50">
          <TableRow>
            <TableHead class="w-10"><Checkbox :model-value="allSelected" :aria-label="uz.users.selectAll" data-testid="select-all" @update:model-value="toggleAll" /></TableHead>
            <TableHead>{{ uz.locations.name }}</TableHead>
            <TableHead>{{ uz.locations.toleranceShort }}</TableHead>
            <TableHead>{{ uz.locations.url }}</TableHead>
            <TableHead>{{ uz.users.addDate }}</TableHead>
            <TableHead class="text-right">{{ uz.common.actions }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="l in rows" :key="l.id" :data-state="selected.has(l.id) ? 'selected' : undefined" data-testid="location-row">
            <TableCell><Checkbox :model-value="selected.has(l.id)" :aria-label="l.name" @update:model-value="toggleOne(l.id, $event)" /></TableCell>
            <TableCell>
              <div class="flex items-center gap-3">
                <div class="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-full"><MapPin class="size-4" aria-hidden="true" /></div>
                <div class="min-w-0">
                  <button class="block truncate font-medium hover:underline" data-testid="location-name" @click="viewing = l">{{ l.name }}</button>
                  <p class="text-muted-foreground text-sm">{{ l.polygon.length }} {{ uz.locations.pointsSuffix }}</p>
                </div>
              </div>
            </TableCell>
            <TableCell><Badge variant="outline">{{ l.tolerance_meters }} m</Badge></TableCell>
            <TableCell>
              <a v-if="l.location_url" :href="l.location_url" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1 text-sm hover:underline">
                <ExternalLink class="size-3.5" aria-hidden="true" />{{ uz.locations.openUrl }}
              </a>
              <span v-else class="text-muted-foreground">—</span>
            </TableCell>
            <TableCell class="text-muted-foreground whitespace-nowrap">{{ formatDate(l.created_at) }}</TableCell>
            <TableCell class="text-right">
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="outline" size="icon" :aria-label="uz.common.actions" data-testid="row-menu"><EllipsisVertical class="size-4" aria-hidden="true" /></Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem @select="viewing = l"><Map class="size-4" aria-hidden="true" />{{ uz.locations.viewMap }}</DropdownMenuItem>
                  <DropdownMenuItem v-if="can('locations:update')" @select="openForm(l)"><Pencil class="size-4" aria-hidden="true" />{{ uz.common.edit }}</DropdownMenuItem>
                  <DropdownMenuItem v-if="can('locations:delete')" variant="destructive" @select="(deleteError = ''), (toDelete = l)"><Trash2 class="size-4" aria-hidden="true" />{{ uz.common.delete }}</DropdownMenuItem>
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

    <Dialog :open="!!viewing" @update:open="(v: boolean) => !v && (viewing = null)">
      <DialogContent class="sm:max-w-2xl">
        <DialogHeader><DialogTitle>{{ viewing?.name }}</DialogTitle></DialogHeader>
        <LocationMap v-if="viewing" :polygon="viewing.polygon" />
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="formOpen">
      <DialogContent class="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader><DialogTitle>{{ editing ? uz.locations.edit : uz.locations.new }}</DialogTitle></DialogHeader>
        <form class="space-y-4" novalidate @submit.prevent="onSubmit">
          <FormField :label="uz.locations.name" :error="errors.name" for="name">
            <Input id="name" v-model="name" v-bind="nameAttrs" />
          </FormField>

          <FormField :label="uz.locations.points" :error="pointsError">
            <div class="space-y-2">
              <div v-for="(_, i) in points" :key="i" class="flex gap-2">
                <Input v-model="points[i]" :placeholder="uz.locations.pointPlaceholder" :aria-label="`${i + 1}-nuqta`" />
                <Button type="button" variant="outline" size="sm" :disabled="points.length <= 3" @click="removePoint(i)">{{ uz.common.remove }}</Button>
              </div>
              <Button type="button" variant="outline" size="sm" @click="addPoint">{{ uz.locations.addPoint }}</Button>
            </div>
          </FormField>
          <LocationMap :polygon="preview" />

          <div class="grid gap-4 sm:grid-cols-2">
            <FormField :label="uz.locations.tolerance" :error="errors.tolerance_meters" for="tol">
              <Input id="tol" v-model="tolerance" v-bind="toleranceAttrs" type="number" min="0" max="1000" />
            </FormField>
            <FormField :label="uz.locations.url" :error="errors.location_url" for="url">
              <Input id="url" v-model="url" v-bind="urlAttrs" type="url" />
            </FormField>
          </div>
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
