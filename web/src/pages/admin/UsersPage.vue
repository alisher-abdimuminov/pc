<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import { EllipsisVertical, KeyRound, Pencil, Plus, ShieldCheck, User } from '@lucide/vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { computed, ref } from 'vue'
import { z } from 'zod'
import { selectOptions, users } from '@/api/admin'
import FormField from '@/components/FormField.vue'
import ListFooter from '@/components/ListFooter.vue'
import ListHeader from '@/components/ListHeader.vue'
import SelectField from '@/components/SelectField.vue'
import PasswordDialog from '@/components/PasswordDialog.vue'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Checkbox } from '@/components/ui/checkbox'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import PermissionPicker from '@/components/PermissionPicker.vue'
import { Input } from '@/components/ui/input'
import { useCan, useIsAdmin } from '@/lib/permissions'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useListQuery } from '@/composables/useListQuery'
import { useRowSelection } from '@/composables/useRowSelection'
import { applyApiError, errorMessage } from '@/lib/form'
import { formatDate, formatDateTime } from '@/lib/format'
import { uz } from '@/locales/uz'

const qc = useQueryClient()
const can = useCan()
const isAdmin = useIsAdmin()
// a little admin may change students only; staff accounts and passwords stay with admins
const canManage = (u: { role: string }) => can('users:update') && (isAdmin() || u.role === 'student')
const role = ref('')
const facultyId = ref('')
const groupId = ref('')
const active = ref('')
const extra = computed(() => ({
  role: role.value || undefined,
  faculty_id: facultyId.value || undefined,
  group_id: groupId.value || undefined,
  is_active: active.value || undefined,
}))
const { search, page, pageSize, query } = useListQuery('users', users.list, extra)

const facultyOptions = useQuery({ queryKey: ['faculty-options'], queryFn: selectOptions.faculties })
const groupOptions = useQuery({ queryKey: ['group-options'], queryFn: selectOptions.groups })
const fOpts = computed(() => (facultyOptions.data.value ?? []).map((f) => ({ value: f.id, label: f.name })))
const gOpts = computed(() => (groupOptions.data.value ?? []).map((g) => ({ value: g.id, label: g.name })))
const roleOpts = [
  { value: 'admin', label: uz.roles.admin },
  { value: 'little', label: uz.roles.little },
  { value: 'teacher', label: uz.roles.teacher },
  { value: 'student', label: uz.roles.student },
]
const activeOpts = [
  { value: 'true', label: uz.users.active },
  { value: 'false', label: uz.users.inactive },
]

const filterCount = computed(() => [role, facultyId, groupId, active].filter((f) => f.value).length)
function clearFilters() {
  role.value = facultyId.value = groupId.value = active.value = ''
}
// ---- rows, selection, range ----
const rows = computed(() => query.data.value?.items ?? [])
const { selected, allSelected, toggleOne, toggleAll } = useRowSelection(rows)
const initials = (name: string) => name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]!.toUpperCase()).join('')
const subtitle = (u: { username: string; group_name: string | null }) => u.group_name || u.username || '—'

// ---- access switch and quick actions ----
const actionError = ref('')
const refresh = () => qc.invalidateQueries({ queryKey: ['users'] })
const toggle = useMutation({
  mutationFn: (v: { id: string; active: boolean }) => users.setActive(v.id, v.active),
  onMutate: () => (actionError.value = ''),
  onSuccess: refresh,
  onError: (e) => (actionError.value = errorMessage(e)),
})
const regen = useMutation({
  mutationFn: (id: string) => users.regeneratePassword(id),
  onMutate: () => (actionError.value = ''),
  onSuccess: async (r, id) => {
    const u = rows.value.find((x) => x.id === id)
    created.value = { username: u?.username ?? '', password: r.password }
    pwOpen.value = true
    await refresh()
  },
  onError: (e) => (actionError.value = errorMessage(e)),
})

// ---- create admin/teacher ----
const formOpen = ref(false)
const formError = ref('')
const { defineField, handleSubmit, errors, setErrors, resetForm, isSubmitting } = useForm({
  validationSchema: toTypedSchema(
    z.object({
      username: z.string().trim().min(1, uz.common.required),
      full_name: z.string().trim().min(1, uz.common.required),
      role: z.enum(['admin', 'little', 'teacher']),
      hemis_login: z.string().trim().optional(),
    }),
  ),
})
const [username, usernameAttrs] = defineField('username')
const [fullName, fullNameAttrs] = defineField('full_name')
const [newRole] = defineField('role')
const [hemisLogin, hemisLoginAttrs] = defineField('hemis_login')
const newPerms = ref<string[]>([])

function openForm() {
  formError.value = ''
  resetForm({ values: { username: '', full_name: '', role: 'teacher', hemis_login: '' } })
  newPerms.value = []
  formOpen.value = true
}

const created = ref<{ username: string; password: string } | null>(null)
const pwOpen = ref(false)
const onSubmit = handleSubmit(async (v) => {
  formError.value = ''
  try {
    const r = await users.create({
      ...v,
      hemis_login: v.role === 'teacher' ? v.hemis_login || undefined : undefined,
      permissions: v.role === 'little' ? newPerms.value : undefined,
    })
    created.value = { username: r.user.username, password: r.password }
    formOpen.value = false
    pwOpen.value = true
    await qc.invalidateQueries({ queryKey: ['users'] })
  } catch (e) {
    applyApiError(e, setErrors, formError)
  }
})
</script>

<template>
  <div>
    <ListHeader v-model:search="search" :title="uz.users.title" :total="query.data.value?.total" :filter-count="filterCount" @clear-filters="clearFilters">
      <template #filters>
        <SelectField v-model="role" :options="roleOpts" :placeholder="uz.users.role" />
        <SelectField v-model="facultyId" :options="fOpts" :placeholder="uz.users.faculty" />
        <SelectField v-model="groupId" :options="gOpts" :placeholder="uz.users.group" />
        <SelectField v-model="active" :options="activeOpts" :placeholder="uz.users.status" />
      </template>
      <template #actions><Button v-if="isAdmin()" @click="openForm"><Plus class="size-4" aria-hidden="true" />{{ uz.users.new }}</Button></template>
    </ListHeader>

    <p v-if="actionError" class="text-destructive pb-3 text-sm" role="alert">{{ actionError }}</p>

    <div class="overflow-x-auto rounded-xl border">
      <Table>
        <TableHeader class="bg-muted/50">
          <TableRow>
            <TableHead class="w-10">
              <Checkbox :model-value="allSelected" :aria-label="uz.users.selectAll" data-testid="select-all" @update:model-value="toggleAll" />
            </TableHead>
            <TableHead>{{ uz.users.user }}</TableHead>
            <TableHead>{{ uz.users.role }}</TableHead>
            <TableHead>{{ uz.users.status }}</TableHead>
            <TableHead>{{ uz.users.hemis }}</TableHead>
            <TableHead>{{ uz.users.addDate }}</TableHead>
            <TableHead>{{ uz.users.lastActive }}</TableHead>
            <TableHead>{{ uz.users.access }}</TableHead>
            <TableHead class="text-right">{{ uz.common.actions }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="u in rows" :key="u.id" :data-state="selected.has(u.id) ? 'selected' : undefined" data-testid="user-row">
            <TableCell>
              <Checkbox :model-value="selected.has(u.id)" :aria-label="u.full_name || u.username" @update:model-value="toggleOne(u.id, $event)" />
            </TableCell>
            <TableCell>
              <div class="flex items-center gap-3">
                <Avatar class="size-9">
                  <AvatarImage v-if="u.has_photo" :src="`/api/files/avatars/${u.id}`" :alt="u.full_name" />
                  <AvatarFallback>{{ initials(u.full_name || u.username) }}</AvatarFallback>
                </Avatar>
                <div class="min-w-0">
                  <RouterLink :to="`/admin/users/${u.id}`" class="block truncate font-medium hover:underline">{{ u.full_name || u.username }}</RouterLink>
                  <p class="text-muted-foreground truncate text-sm">{{ subtitle(u) }}</p>
                </div>
              </div>
            </TableCell>
            <TableCell>{{ uz.roles[u.role] }}</TableCell>
            <TableCell>
              <Badge
                variant="outline"
                data-testid="status-badge"
                :class="u.is_active ? 'border-emerald-600 text-emerald-700 dark:text-emerald-400' : 'border-rose-500 text-rose-600 dark:text-rose-400'"
              >{{ u.is_active ? uz.users.active : uz.users.inactive }}</Badge>
            </TableCell>
            <TableCell>
              <template v-if="u.hemis_login">
                <Badge variant="outline" data-testid="hemis-badge">{{ uz.users.hemisBadge }}</Badge>
                <p v-if="u.hemis_synced_at" class="text-muted-foreground mt-0.5 text-xs" :title="uz.users.hemisSynced">{{ formatDateTime(u.hemis_synced_at) }}</p>
              </template>
              <span v-else class="text-muted-foreground">—</span>
            </TableCell>
            <TableCell class="text-muted-foreground whitespace-nowrap">{{ formatDate(u.created_at) }}</TableCell>
            <TableCell class="whitespace-nowrap">{{ formatDateTime(u.last_login_at) }}</TableCell>
            <TableCell>
              <Switch
                :model-value="u.is_active"
                :disabled="toggle.isPending.value || !canManage(u)"
                :aria-label="u.is_active ? uz.users.deactivate : uz.users.activate"
                data-testid="access-switch"
                @update:model-value="toggle.mutate({ id: u.id, active: $event })"
              />
            </TableCell>
            <TableCell class="text-right">
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="outline" size="icon" :aria-label="uz.common.actions" data-testid="row-menu"><EllipsisVertical class="size-4" aria-hidden="true" /></Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem as-child>
                    <RouterLink :to="`/admin/users/${u.id}`"><User class="size-4" aria-hidden="true" />{{ uz.users.viewProfile }}</RouterLink>
                  </DropdownMenuItem>
                  <DropdownMenuItem v-if="canManage(u)" as-child>
                    <RouterLink :to="`/admin/users/${u.id}`"><Pencil class="size-4" aria-hidden="true" />{{ uz.users.edit }}</RouterLink>
                  </DropdownMenuItem>
                  <DropdownMenuItem v-if="u.role === 'little' && isAdmin()" as-child>
                    <RouterLink :to="`/admin/users/${u.id}`" data-testid="edit-permissions"><ShieldCheck class="size-4" aria-hidden="true" />{{ uz.permissions.edit }}</RouterLink>
                  </DropdownMenuItem>
                  <DropdownMenuItem v-if="u.username && isAdmin()" @select="regen.mutate(u.id)"><KeyRound class="size-4" aria-hidden="true" />{{ uz.users.regenerate }}</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
          <TableRow v-if="query.data.value && !rows.length">
            <TableCell colspan="9" class="text-muted-foreground py-8 text-center">{{ uz.common.empty }}</TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <ListFooter v-if="query.data.value" v-model:page="page" :page-size="pageSize" :total="query.data.value.total" :row-count="rows.length" />

    <Dialog v-model:open="formOpen">
      <DialogContent class="max-h-[90vh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{{ uz.users.new }}</DialogTitle>
          <DialogDescription>{{ uz.users.createdHint }}</DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit.prevent="onSubmit">
          <FormField :label="uz.users.role" :error="errors.role" for="role">
            <SelectField id="role" v-model="newRole" :options="roleOpts.slice(0, 3)" />
          </FormField>
          <FormField :label="uz.users.username" :error="errors.username" for="username">
            <Input id="username" v-model="username" v-bind="usernameAttrs" autocapitalize="none" />
          </FormField>
          <FormField :label="uz.users.fullName" :error="errors.full_name" for="full_name">
            <Input id="full_name" v-model="fullName" v-bind="fullNameAttrs" />
          </FormField>
          <FormField v-if="newRole === 'teacher'" :label="uz.users.hemisLogin" :error="errors.hemis_login" for="hemis_login">
            <Input id="hemis_login" v-model="hemisLogin" v-bind="hemisLoginAttrs" autocapitalize="none" />
            <p class="text-muted-foreground text-xs">{{ uz.users.hemisLoginHint }}</p>
          </FormField>
          <div v-if="newRole === 'little'">
            <p class="mb-2 text-sm font-medium">{{ uz.permissions.title }}</p>
            <PermissionPicker v-model="newPerms" />
          </div>
          <p v-if="formError" class="text-destructive text-sm" role="alert">{{ formError }}</p>
          <DialogFooter>
            <Button type="button" variant="outline" @click="formOpen = false">{{ uz.common.cancel }}</Button>
            <Button type="submit" :disabled="isSubmitting">{{ uz.common.create }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <PasswordDialog v-if="created" v-model:open="pwOpen" :username="created.username" :password="created.password" />
  </div>
</template>
