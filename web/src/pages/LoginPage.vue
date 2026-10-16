<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import { useForm } from 'vee-validate'
import { Eye, EyeOff, GraduationCap } from '@lucide/vue'
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { z } from 'zod'
import { ApiError } from '@/api/client'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { uz } from '@/locales/uz'
import { useSessionStore } from '@/stores/session'

const schema = toTypedSchema(
  z.object({
    username: z.string().trim().min(1, uz.common.required),
    password: z.string().min(1, uz.common.required),
  }),
)

const { defineField, handleSubmit, errors, setErrors, isSubmitting } = useForm({ validationSchema: schema })
const [username, usernameAttrs] = defineField('username')
const [password, passwordAttrs] = defineField('password')
const showPassword = ref(false)
const formError = ref('')
const session = useSessionStore()
const router = useRouter()
const route = useRoute()
const blocked = route.query.reason === 'inactive'

// ---- HEMIS sign-in for students and teachers: a plain full-page navigation, the server does the OAuth dance ----
const HEMIS_STUDENT_START = '/api/auth/hemis/student/start'
const HEMIS_TEACHER_START = '/api/auth/hemis/teacher/start'
const hemisLoading = ref(false)
const oauthCode = typeof route.query.error === 'string' ? route.query.error : ''
const oauthError = ref(oauthCode ? (uz.login.errors[oauthCode] ?? uz.common.unexpectedError) : '')
if (oauthCode) {
  // show the message once, then clean the URL
  const rest = { ...route.query }
  delete rest.error
  void router.replace({ query: rest })
}
const resetHemis = () => (hemisLoading.value = false)
onMounted(() => window.addEventListener('pageshow', resetHemis)) // back button restores the page from cache
onBeforeUnmount(() => window.removeEventListener('pageshow', resetHemis))

const onSubmit = handleSubmit(async (values) => {
  formError.value = ''
  try {
    await session.login(values.username, values.password)
    await router.push('/')
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.body.fields) setErrors(e.body.fields)
      formError.value = e.body.message
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
        <img src="/logo.png" alt="" class="mx-auto mb-2 size-20 object-contain" />
        <CardTitle>{{ uz.login.title }}</CardTitle>
        <p class="text-muted-foreground text-sm">{{ uz.login.hint }}</p>
      </CardHeader>
      <CardContent>
        <p v-if="blocked" class="mb-4 rounded-md bg-amber-100 p-3 text-sm text-amber-900 dark:bg-amber-950 dark:text-amber-200" role="alert" data-testid="inactive-warning">
          {{ uz.login.inactive }}
        </p>
        <form class="space-y-4" novalidate @submit.prevent="onSubmit">
          <div class="space-y-1.5">
            <Label for="username">{{ uz.login.username }}</Label>
            <Input id="username" v-model="username" v-bind="usernameAttrs" autocomplete="username" autocapitalize="none" />
            <p v-if="errors.username" class="text-destructive text-sm">{{ errors.username }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="password">{{ uz.login.password }}</Label>
            <div class="relative">
              <Input id="password" v-model="password" v-bind="passwordAttrs" :type="showPassword ? 'text' : 'password'" autocomplete="current-password" class="pr-10" />
              <button
                type="button"
                class="text-muted-foreground hover:text-foreground absolute inset-y-0 right-0 flex w-10 items-center justify-center"
                :aria-label="showPassword ? uz.login.hidePassword : uz.login.showPassword"
                :aria-pressed="showPassword"
                data-testid="toggle-password"
                @click="showPassword = !showPassword"
              >
                <EyeOff v-if="showPassword" class="size-4" aria-hidden="true" />
                <Eye v-else class="size-4" aria-hidden="true" />
              </button>
            </div>
            <p v-if="errors.password" class="text-destructive text-sm">{{ errors.password }}</p>
          </div>
          <p v-if="formError" class="text-destructive text-sm" role="alert">{{ formError }}</p>
          <Button type="submit" class="w-full" :disabled="isSubmitting">
            {{ isSubmitting ? uz.login.submitting : uz.login.submit }}
          </Button>
        </form>

        <div class="text-muted-foreground my-4 flex items-center gap-3 text-xs">
          <span class="bg-border h-px flex-1" /> {{ uz.login.or }} <span class="bg-border h-px flex-1" />
        </div>
        <p v-if="oauthError" class="text-destructive mb-3 text-sm" role="alert" data-testid="oauth-error">{{ oauthError }}</p>
        <Button
          as="a"
          :href="HEMIS_STUDENT_START"
          class="w-full"
          data-testid="hemis-student"
          :aria-disabled="hemisLoading"
          :class="hemisLoading && 'pointer-events-none opacity-60'"
          @click="hemisLoading = true"
        >
          <GraduationCap class="size-4" aria-hidden="true" />
          {{ hemisLoading ? uz.login.hemisRedirecting : uz.login.hemisStudent }}
        </Button>
        <Button
          as="a"
          :href="HEMIS_TEACHER_START"
          variant="outline"
          class="mt-2 w-full"
          data-testid="hemis-teacher"
          :aria-disabled="hemisLoading"
          :class="hemisLoading && 'pointer-events-none opacity-60'"
          @click="hemisLoading = true"
        >
          <GraduationCap class="size-4" aria-hidden="true" />
          {{ hemisLoading ? uz.login.hemisRedirecting : uz.login.hemisTeacher }}
        </Button>
      </CardContent>
    </Card>
  </main>
</template>
