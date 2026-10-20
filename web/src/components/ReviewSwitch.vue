<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { attempts as adminAttempts } from '@/api/admin'
import { teacher } from '@/api/teacher'
import { Switch } from '@/components/ui/switch'
import { errorMessage } from '@/lib/form'
import { uz } from '@/locales/uz'

/**
 * An attempt accepted without a face check (the face-service was busy) waits for a person. The switch
 * confirms it (on) or rejects it (off); until then it shows yellow.
 */
const props = defineProps<{ attemptId: string; review: 'pending' | 'approved' | 'rejected'; as: 'admin' | 'teacher' }>()

const qc = useQueryClient()
const error = ref('')
const approved = computed(() => props.review === 'approved')
const tone = computed(
  () =>
    ({
      pending: 'border-yellow-400 bg-yellow-100 text-yellow-900 dark:border-yellow-700 dark:bg-yellow-950 dark:text-yellow-300',
      approved: 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300',
      rejected: 'border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-900 dark:bg-rose-950 dark:text-rose-300',
    })[props.review],
)

const mutation = useMutation({
  mutationFn: (value: boolean) => (props.as === 'admin' ? adminAttempts.review(props.attemptId, value) : teacher.reviewAttempt(props.attemptId, value)),
  onSuccess: async () => {
    error.value = ''
    for (const key of ['admin-attempts', 'attempts', 'report', 'teacher-group']) await qc.invalidateQueries({ queryKey: [key] })
  },
  onError: (e) => (error.value = errorMessage(e)),
})
</script>

<template>
  <div class="inline-flex flex-col items-start gap-1" data-testid="review-switch">
    <div class="inline-flex items-center gap-2">
      <span class="rounded-md border px-2 py-0.5 text-xs font-medium whitespace-nowrap" :class="tone" data-testid="review-badge">{{ uz.teacher.review[review] }}</span>
      <Switch
        :model-value="approved"
        :disabled="mutation.isPending.value"
        :aria-label="uz.teacher.review.switchLabel"
        data-testid="review-toggle"
        @update:model-value="mutation.mutate($event)"
      />
    </div>
    <p v-if="error" class="text-destructive text-xs" role="alert">{{ error }}</p>
  </div>
</template>
