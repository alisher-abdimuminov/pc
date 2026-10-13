<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { student } from '@/api/student'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { Button } from '@/components/ui/button'
import { useSessionStore } from '@/stores/session'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { errorMessage } from '@/lib/form'
import { uz } from '@/locales/uz'

const session = useSessionStore()
const router = useRouter()
async function onLogout() {
  await session.logout()
  await router.push('/login')
}

const { data, isPending, error } = useQuery({ queryKey: ['student-profile'], queryFn: student.profile })
const initials = computed(() =>
  (data.value?.full_name ?? '')
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]!.toUpperCase())
    .join(''),
)
const rows = computed(() => {
  const p = data.value
  if (!p) return []
  return [
    [uz.student.group, p.group_name],
    [uz.student.faculty, p.faculty_name],
    [uz.student.birthDate, p.birth_date],
    [uz.student.gender, p.gender],
    [uz.student.phone, p.phone],
    [uz.student.level, p.level],
    [uz.student.semester, p.semester],
    [uz.student.address, [p.country, p.province, p.district, p.address].filter(Boolean).join(', ')],
  ] as [string, string | null][]
})
</script>

<template>
  <div class="space-y-4">
    <p v-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
    <p v-else-if="error" class="text-destructive text-sm" role="alert">{{ errorMessage(error) }}</p>
    <template v-else-if="data">
      <div class="flex items-center gap-4">
        <img v-if="data.photo_url" :src="data.photo_url" :alt="data.full_name" class="size-20 rounded-full border object-cover" />
        <div v-else class="bg-muted flex size-20 items-center justify-center rounded-full text-xl font-semibold" aria-hidden="true">{{ initials }}</div>
        <div>
          <h1 class="text-xl font-semibold">{{ data.full_name }}</h1>
          <p class="text-muted-foreground text-sm">{{ data.username }}</p>
        </div>
      </div>
      <Card>
        <CardHeader><CardTitle class="text-base">{{ uz.student.profileInfo }}</CardTitle></CardHeader>
        <CardContent>
          <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
            <template v-for="[label, value] in rows" :key="label">
              <dt class="text-muted-foreground">{{ label }}</dt>
              <dd>{{ value || '—' }}</dd>
            </template>
          </dl>
        </CardContent>
      </Card>
      <div class="bg-card flex items-center justify-between rounded-3xl border p-4">
        <span class="text-sm font-medium">{{ uz.theme.label }}</span>
        <ThemeToggle />
      </div>
      <Button variant="outline" class="w-full" @click="onLogout">{{ uz.student.logout }}</Button>
    </template>
  </div>
</template>
