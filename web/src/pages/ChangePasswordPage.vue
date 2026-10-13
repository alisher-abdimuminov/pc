<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import { useForm } from 'vee-validate'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { z } from 'zod'
import { changePassword } from '@/api/auth'
import { ApiError } from '@/api/client'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { uz } from '@/locales/uz'
import { useSessionStore } from '@/stores/session'

const schema = toTypedSchema(
  z
    .object({
      old_password: z.string().min(1, uz.common.required),
      new_password: z.string().min(8, uz.changePassword.tooShort),
      confirm_password: z.string().min(1, uz.common.required),
    })
    .refine((v) => v.new_password === v.confirm_password, {
      path: ['confirm_password'],
      message: uz.changePassword.mismatch,
    }),
)

const { defineField, handleSubmit, errors, setErrors, isSubmitting } = useForm({ validationSchema: schema })
const [oldPassword, oldAttrs] = defineField('old_password')
const [newPassword, newAttrs] = defineField('new_password')
const [confirmPassword, confirmAttrs] = defineField('confirm_password')
const formError = ref('')
const session = useSessionStore()
const router = useRouter()

const onSubmit = handleSubmit(async (values) => {
  formError.value = ''
  try {
    session.setUser(await changePassword(values.old_password, values.new_password))
    await router.push('/')
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.body.fields) setErrors(e.body.fields)
      else formError.value = e.body.message
    } else {
      formError.value = uz.common.unexpectedError
    }
  }
})
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-sm items-center p-4">
    <ThemeToggle class="absolute top-4 right-4" />
    <Card class="w-full">
      <CardHeader>
        <CardTitle>{{ uz.changePassword.title }}</CardTitle>
        <p v-if="session.user?.must_change_password" class="text-muted-foreground text-sm">
          {{ uz.changePassword.hint }}
        </p>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" novalidate @submit.prevent="onSubmit">
          <div class="space-y-1.5">
            <Label for="old_password">{{ uz.changePassword.oldPassword }}</Label>
            <Input id="old_password" v-model="oldPassword" v-bind="oldAttrs" type="password" autocomplete="current-password" />
            <p v-if="errors.old_password" class="text-destructive text-sm">{{ errors.old_password }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="new_password">{{ uz.changePassword.newPassword }}</Label>
            <Input id="new_password" v-model="newPassword" v-bind="newAttrs" type="password" autocomplete="new-password" />
            <p v-if="errors.new_password" class="text-destructive text-sm">{{ errors.new_password }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="confirm_password">{{ uz.changePassword.confirmPassword }}</Label>
            <Input id="confirm_password" v-model="confirmPassword" v-bind="confirmAttrs" type="password" autocomplete="new-password" />
            <p v-if="errors.confirm_password" class="text-destructive text-sm">{{ errors.confirm_password }}</p>
          </div>
          <p v-if="formError" class="text-destructive text-sm" role="alert">{{ formError }}</p>
          <Button type="submit" class="w-full" :disabled="isSubmitting">{{ uz.changePassword.submit }}</Button>
        </form>
      </CardContent>
    </Card>
  </main>
</template>
