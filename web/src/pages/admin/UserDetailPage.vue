<script setup lang="ts">
import BackLink from '@/components/BackLink.vue'
import { Check } from '@lucide/vue'
import { toTypedSchema } from '@vee-validate/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { z } from 'zod'
import { selectOptions, users, type Sensitive } from '@/api/admin'
import FormField from '@/components/FormField.vue'
import SelectField from '@/components/SelectField.vue'
import PasswordDialog from '@/components/PasswordDialog.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import PermissionPicker from '@/components/PermissionPicker.vue'
import { Input } from '@/components/ui/input'
import { useCan, useIsAdmin } from '@/lib/permissions'
import { ApiError } from '@/api/client'
import { applyApiError, errorMessage } from '@/lib/form'
import { formatDateTime } from '@/lib/format'
import { uz } from '@/locales/uz'

const id = useRoute().params.id as string
const qc = useQueryClient()
const { data: user, isPending } = useQuery({ queryKey: ['user', id], queryFn: () => users.get(id) })

const facultyOptions = useQuery({ queryKey: ['faculty-options'], queryFn: selectOptions.faculties })
const groupOptions = useQuery({ queryKey: ['group-options'], queryFn: selectOptions.groups })
const fOpts = computed(() => (facultyOptions.data.value ?? []).map((f) => ({ value: f.id, label: f.name })))
const gOpts = computed(() => (groupOptions.data.value ?? []).map((g) => ({ value: g.id, label: g.name })))
const isStudent = computed(() => user.value?.role === 'student')
const can = useCan()
const isAdmin = useIsAdmin()
// a little admin changes students only (staff accounts and account security stay with admins)
const canEdit = computed(() => can('users:update') && (isAdmin() || isStudent.value))

// ---- edit form ----
const formError = ref('')
const saved = ref(false)
const { defineField, handleSubmit, errors, setErrors, resetForm, isSubmitting } = useForm({
  validationSchema: toTypedSchema(
    z.object({
      full_name: z.string().trim().min(1, uz.common.required),
      phone: z.string().trim().optional(),
      group_id: z.string().nullable().optional(),
      faculty_id: z.string().nullable().optional(),
      hemis_login: z.string().trim().optional(),
    }),
  ),
})
const [fullName, fullNameAttrs] = defineField('full_name')
const [phone, phoneAttrs] = defineField('phone')
const [groupId] = defineField('group_id')
const [facultyId] = defineField('faculty_id')
const [hemisLogin, hemisLoginAttrs] = defineField('hemis_login')
const isTeacher = computed(() => user.value?.role === 'teacher')
const isLittle = computed(() => user.value?.role === 'little')

// ---- permissions of a little admin (admins only) ----
const perms = ref<string[]>([])
const permsSaved = ref(false)
const permsError = ref('')
watch(user, (u) => u && (perms.value = [...u.permissions]), { immediate: true })
const savePerms = useMutation({
  mutationFn: () => users.setPermissions(id, perms.value),
  onMutate: () => {
    permsSaved.value = false
    permsError.value = ''
  },
  onSuccess: async () => {
    permsSaved.value = true
    await refresh()
  },
  onError: (e) => (permsError.value = errorMessage(e)),
})
watch(
  user,
  (u) => u && resetForm({ values: { full_name: u.full_name, phone: u.phone, group_id: u.group_id ?? '', faculty_id: u.faculty_id ?? '', hemis_login: u.hemis_login } }),
  { immediate: true },
)

const refresh = () => Promise.all([qc.invalidateQueries({ queryKey: ['user', id] }), qc.invalidateQueries({ queryKey: ['users'] })])

const onSubmit = handleSubmit(async (v) => {
  formError.value = ''
  saved.value = false
  try {
    await users.update(id, { full_name: v.full_name, phone: v.phone ?? '', group_id: v.group_id || null, faculty_id: v.faculty_id || null, ...(isTeacher.value ? { hemis_login: v.hemis_login ?? '' } : {}) })
    await refresh()
    saved.value = true
  } catch (e) {
    applyApiError(e, setErrors, formError)
  }
})

// ---- activation ----
const actionError = ref('')
const toggle = useMutation({
  mutationFn: (active: boolean) => users.setActive(id, active),
  onSuccess: refresh,
  onError: (e) => (actionError.value = errorMessage(e)),
})

// ---- password regeneration ----
const pwOpen = ref(false)
const newPassword = ref('')
const regen = useMutation({
  mutationFn: () => users.regeneratePassword(id),
  onSuccess: (r) => {
    newPassword.value = r.password
    pwOpen.value = true
    return refresh()
  },
  onError: (e) => (actionError.value = errorMessage(e)),
})

// ---- reference photo -> face embedding (done by the face-service) ----
const photoError = ref('')
const photoOk = ref(false)
const photoVersion = ref(0) // cache-buster for the <img>
const upload = useMutation({
  mutationFn: (file: File) => users.uploadPhoto(id, file),
  onSuccess: async () => {
    photoOk.value = true
    photoVersion.value++
    await refresh()
  },
  onError: (e) => {
    photoOk.value = false
    photoError.value = e instanceof ApiError ? (e.body.fields?.image ?? e.body.message) : errorMessage(e)
  },
})
function onPhoto(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  photoError.value = ''
  photoOk.value = false
  if (file) upload.mutate(file)
  input.value = ''
}

// ---- passport data: fetched (and audited) only on explicit request ----
const sensitive = ref<Sensitive | null>(null)
const sensitiveError = ref('')
const reveal = useMutation({
  mutationFn: () => users.sensitive(id),
  onSuccess: (s) => (sensitive.value = s),
  onError: (e) => (sensitiveError.value = errorMessage(e)),
})
</script>

<template>
  <div class="space-y-4">
    <BackLink to="/admin/users">{{ uz.users.title }}</BackLink>
    <p v-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
    <template v-else-if="user">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-xl font-semibold">{{ user.full_name || user.username }}</h1>
          <p class="text-muted-foreground text-sm">
            {{ uz.roles[user.role] }} · {{ user.username || '—' }} ·
            <template v-if="user.hemis_login"><Badge variant="outline" data-testid="hemis-badge">{{ uz.users.hemisBadge }}</Badge> ·</template>
            <Badge :variant="user.is_active ? 'default' : 'secondary'">{{ user.is_active ? uz.users.active : uz.users.inactive }}</Badge>
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <Button v-if="user.username && isAdmin()" variant="outline" :disabled="regen.isPending.value" @click="regen.mutate()">{{ uz.users.regenerate }}</Button>
          <label class="flex items-center gap-2 text-sm">
            <Switch
              :model-value="user.is_active"
              :disabled="toggle.isPending.value || !canEdit"
              :aria-label="user.is_active ? uz.users.deactivate : uz.users.activate"
              @update:model-value="toggle.mutate($event)"
            />
            {{ user.is_active ? uz.users.active : uz.users.inactive }}
          </label>
        </div>
      </div>
      <p v-if="actionError" class="text-destructive text-sm" role="alert">{{ actionError }}</p>

      <div class="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>{{ uz.users.edit }}</CardTitle></CardHeader>
          <CardContent>
            <form class="space-y-4" novalidate @submit.prevent="onSubmit">
              <fieldset :disabled="!canEdit" class="space-y-4">
              <FormField :label="uz.users.fullName" :error="errors.full_name" for="full_name">
                <Input id="full_name" v-model="fullName" v-bind="fullNameAttrs" />
              </FormField>
              <FormField :label="uz.users.phone" :error="errors.phone" for="phone">
                <Input id="phone" v-model="phone" v-bind="phoneAttrs" />
              </FormField>
              <FormField v-if="isTeacher" :label="uz.users.hemisLogin" :error="errors.hemis_login" for="hemis_login">
                <Input id="hemis_login" v-model="hemisLogin" v-bind="hemisLoginAttrs" autocapitalize="none" />
                <p class="text-muted-foreground text-xs">{{ uz.users.hemisLoginHint }}</p>
              </FormField>
              <template v-if="isStudent">
                <FormField :label="uz.users.group" :error="errors.group_id" for="group">
                  <SelectField id="group" v-model="groupId" :options="gOpts" :placeholder="uz.common.none" />
                </FormField>
                <FormField :label="uz.users.faculty" :error="errors.faculty_id" for="faculty">
                  <SelectField id="faculty" v-model="facultyId" :options="fOpts" :placeholder="uz.common.none" />
                </FormField>
              </template>
              </fieldset>
              <p v-if="formError" class="text-destructive text-sm" role="alert">{{ formError }}</p>
              <p v-if="saved" class="flex items-center gap-1 text-sm text-emerald-600" role="status"><Check class="size-4" aria-hidden="true" />{{ uz.common.save }}</p>
              <Button v-if="canEdit" type="submit" :disabled="isSubmitting">{{ uz.common.save }}</Button>
            </form>
          </CardContent>
        </Card>

        <div class="space-y-4">
          <Card v-if="isLittle">
            <CardHeader><CardTitle>{{ uz.permissions.title }}</CardTitle></CardHeader>
            <CardContent class="space-y-3">
              <PermissionPicker v-model="perms" :disabled="!isAdmin()" />
              <template v-if="isAdmin()">
                <p v-if="permsError" class="text-destructive text-sm" role="alert">{{ permsError }}</p>
                <p v-if="permsSaved" class="text-sm text-emerald-600" role="status" data-testid="permissions-saved">{{ uz.permissions.saved }}</p>
                <Button :disabled="savePerms.isPending.value" data-testid="save-permissions" @click="savePerms.mutate()">{{ uz.common.save }}</Button>
              </template>
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle>{{ uz.users.details }}</CardTitle></CardHeader>
            <CardContent>
              <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
                <dt class="text-muted-foreground">{{ uz.users.lastLogin }}</dt><dd>{{ formatDateTime(user.last_login_at) }}</dd>
                <template v-if="isStudent">
                  <dt class="text-muted-foreground">{{ uz.users.group }}</dt><dd>{{ user.group_name ?? '—' }}</dd>
                  <dt class="text-muted-foreground">{{ uz.users.faculty }}</dt><dd>{{ user.faculty_name ?? '—' }}</dd>
                  <dt class="text-muted-foreground">{{ uz.users.birthDate }}</dt><dd>{{ user.birth_date || '—' }}</dd>
                  <dt class="text-muted-foreground">{{ uz.users.gender }}</dt><dd>{{ user.gender || '—' }}</dd>
                  <dt class="text-muted-foreground">{{ uz.users.level }} / {{ uz.users.semester }}</dt><dd>{{ user.level || '—' }} / {{ user.semester || '—' }}</dd>
                  <dt class="text-muted-foreground">{{ uz.users.address }}</dt><dd>{{ user.address || '—' }}</dd>
                  <dt class="text-muted-foreground">{{ uz.users.faceEnrolled }}</dt>
                  <dd>{{ user.face_enrolled ? uz.users.enrolled : uz.users.notEnrolled }}</dd>
                </template>
              </dl>
            </CardContent>
          </Card>

          <Card v-if="isStudent && canEdit">
            <CardHeader><CardTitle>{{ uz.users.photoTitle }}</CardTitle></CardHeader>
            <CardContent class="space-y-3 text-sm">
              <img
                v-if="user.has_photo"
                :src="`/api/files/avatars/${user.id}?v=${photoVersion}`"
                :alt="user.full_name"
                class="size-28 rounded-lg border object-cover"
              />
              <p class="text-muted-foreground">{{ uz.users.photoHint }}</p>
              <label class="block">
                <span class="sr-only">{{ user.has_photo ? uz.users.photoReplace : uz.users.photoUpload }}</span>
                <input type="file" accept="image/jpeg,image/png,image/webp" class="block w-full text-sm file:mr-3 file:rounded-md file:border file:px-3 file:py-1.5" :disabled="upload.isPending.value" data-testid="photo-input" @change="onPhoto" />
              </label>
              <p v-if="upload.isPending.value" class="text-muted-foreground">{{ uz.users.photoUploading }}</p>
              <p v-else-if="photoOk" class="text-emerald-700 dark:text-emerald-400" role="status"><Check class="mr-1 inline size-4 align-text-bottom" aria-hidden="true" />{{ uz.users.photoDone }}</p>
              <p v-if="photoError" class="text-destructive" role="alert">{{ photoError }}</p>
            </CardContent>
          </Card>

          <Card v-if="isStudent && isAdmin()">
            <CardHeader><CardTitle>{{ uz.users.sensitiveTitle }}</CardTitle></CardHeader>
            <CardContent class="space-y-3 text-sm">
              <template v-if="sensitive">
                <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5">
                  <dt class="text-muted-foreground">{{ uz.users.pin }}</dt><dd class="font-mono">{{ sensitive.passport_pin || '—' }}</dd>
                  <dt class="text-muted-foreground">{{ uz.users.passportNumber }}</dt><dd class="font-mono">{{ sensitive.passport_number || '—' }}</dd>
                </dl>
                <Button variant="outline" size="sm" @click="sensitive = null">{{ uz.users.hide }}</Button>
              </template>
              <template v-else>
                <p class="text-muted-foreground">{{ uz.users.sensitiveHint }}</p>
                <Button variant="outline" size="sm" :disabled="reveal.isPending.value" @click="reveal.mutate()">{{ uz.users.show }}</Button>
                <p v-if="sensitiveError" class="text-destructive" role="alert">{{ sensitiveError }}</p>
              </template>
            </CardContent>
          </Card>
        </div>
      </div>
    </template>

    <PasswordDialog v-if="user" v-model:open="pwOpen" :username="user.username" :password="newPassword" />
  </div>
</template>
