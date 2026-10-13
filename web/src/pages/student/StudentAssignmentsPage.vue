<script setup lang="ts">
import { Check } from '@lucide/vue'
import { useQuery, useQueryClient } from '@tanstack/vue-query'
import { useIntervalFn } from '@vueuse/core'
import { computed, ref } from 'vue'
import { student, type StudentAssignment } from '@/api/student'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { errorMessage } from '@/lib/form'
import { formatDateTime } from '@/lib/format'
import { clockOffset, formatTimeLeft } from '@/lib/timeleft'
import { uz } from '@/locales/uz'

const qc = useQueryClient()
const { data, isPending, error } = useQuery({ queryKey: ['student-assignments'], queryFn: student.assignments })

// Count down against the server clock, not the (possibly wrong) device clock.
const tick = ref(Date.now())
useIntervalFn(() => (tick.value = Date.now()), 1000)
const offset = computed(() => (data.value ? clockOffset(data.value.server_time) : 0))
const serverNow = computed(() => tick.value + offset.value)
const timeLeft = (a: StudentAssignment) => formatTimeLeft(new Date(a.deadline).getTime() - serverNow.value, uz.student.deadlinePassed)
const open = (a: StudentAssignment) => new Date(a.deadline).getTime() > serverNow.value
// the server decides; locally we only disable the button once the countdown reaches zero
const canSubmit = (a: StudentAssignment) => a.can_submit && open(a)

const busy = ref<string | null>(null)
const errors = ref<Record<string, string>>({})
const okId = ref<string | null>(null)

async function onFile(a: StudentAssignment, e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  errors.value = { ...errors.value, [a.id]: '' }
  okId.value = null
  if (file.size > 20 * 1024 * 1024) {
    errors.value = { ...errors.value, [a.id]: uz.assignments.fileHint }
    input.value = ''
    return
  }
  busy.value = a.id
  try {
    await student.submit(a.id, file)
    okId.value = a.id
  } catch (err) {
    errors.value = { ...errors.value, [a.id]: errorMessage(err) }
  } finally {
    busy.value = null
    input.value = ''
    await qc.invalidateQueries({ queryKey: ['student-assignments'] })
  }
}

function status(a: StudentAssignment) {
  if (a.submission?.graded_at) return { label: uz.student.graded, variant: 'default' as const }
  if (a.submission) return { label: uz.student.submitted, variant: 'secondary' as const }
  return { label: uz.student.notSubmitted, variant: 'outline' as const }
}
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-xl font-semibold">{{ uz.assignments.title }}</h1>
    <p v-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
    <p v-else-if="error" class="text-destructive text-sm" role="alert">{{ errorMessage(error) }}</p>
    <p v-else-if="!data?.items.length" class="text-muted-foreground text-sm">{{ uz.student.noAssignments }}</p>

    <Card v-for="a in data?.items" :key="a.id">
      <CardHeader class="gap-1">
        <div class="flex items-start justify-between gap-2">
          <CardTitle class="text-base">{{ a.title }}</CardTitle>
          <Badge :variant="status(a).variant">{{ status(a).label }}</Badge>
        </div>
        <p class="text-muted-foreground text-xs">
          {{ uz.assignments.deadline }}: {{ formatDateTime(a.deadline) }} ·
          <span :class="open(a) ? '' : 'text-destructive'" :data-testid="`left-${a.id}`">{{ uz.student.timeLeft }}: {{ timeLeft(a) }}</span>
        </p>
      </CardHeader>
      <CardContent class="space-y-3 text-sm">
        <p v-if="a.description" class="whitespace-pre-line">{{ a.description }}</p>
        <a v-if="a.has_file" :href="`/api/files/assignments/${a.id}`" class="inline-block underline" download>{{ uz.student.taskFile }}</a>

        <div v-if="a.submission" class="bg-muted/40 space-y-1 rounded-md p-2">
          <p class="text-xs">{{ uz.assignments.submittedAt }}: {{ formatDateTime(a.submission.submitted_at) }}</p>
          <a v-if="a.submission.has_file" :href="`/api/files/submissions/${a.submission.id}`" class="underline" download>{{ uz.student.myFile }}</a>
          <template v-if="a.submission.graded_at">
            <p class="font-medium">{{ uz.student.yourGrade }}: {{ a.submission.grade?.toFixed(1) }}</p>
            <p v-if="a.submission.feedback" class="whitespace-pre-line"><span class="text-muted-foreground">{{ uz.student.teacherFeedback }}:</span> {{ a.submission.feedback }}</p>
          </template>
        </div>

        <div class="space-y-1">
          <label class="block">
            <span class="sr-only">{{ a.submission ? uz.student.reupload : uz.student.upload }}</span>
            <input
              type="file"
              class="block w-full text-sm file:mr-3 file:rounded-md file:border file:px-3 file:py-1.5 disabled:opacity-50"
              :disabled="!canSubmit(a) || busy === a.id"
              accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.zip,.rar,.jpg,.jpeg,.png"
              :data-testid="`file-${a.id}`"
              @change="onFile(a, $event)"
            />
          </label>
          <p v-if="busy === a.id" class="text-muted-foreground text-xs">{{ uz.student.uploading }}</p>
          <p v-else-if="okId === a.id && !errors[a.id]" class="text-xs text-emerald-700 dark:text-emerald-400" role="status"><Check class="mr-1 inline size-4 align-text-bottom" aria-hidden="true" />{{ uz.student.uploaded }}</p>
          <p v-if="a.submission?.graded_at" class="text-muted-foreground text-xs">{{ uz.student.gradedLocked }}</p>
          <p v-else-if="!open(a)" class="text-destructive text-xs">{{ uz.student.deadlinePassed }}</p>
          <p v-else class="text-muted-foreground text-xs">{{ uz.assignments.fileHint }}</p>
          <p v-if="errors[a.id]" class="text-destructive text-sm" role="alert">{{ errors[a.id] }}</p>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
