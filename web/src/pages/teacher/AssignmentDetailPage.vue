<script setup lang="ts">
import BackLink from '@/components/BackLink.vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { teacher, type RosterEntry } from '@/api/teacher'
import FormField from '@/components/FormField.vue'
import SelectField from '@/components/SelectField.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { applyApiError } from '@/lib/form'
import { formatDateTime } from '@/lib/format'
import { uz } from '@/locales/uz'

const id = useRoute().params.id as string
const qc = useQueryClient()
const { data, isPending } = useQuery({ queryKey: ['submissions', id], queryFn: () => teacher.submissions(id) })

const GRADE_OPTIONS = Array.from({ length: 9 }, (_, i) => {
  const g = 1 + i * 0.5
  return { value: String(g), label: g.toFixed(1) }
})

const grading = ref<RosterEntry | null>(null)
const form = reactive({ grade: '4', feedback: '' })
const fieldErrors = ref<Record<string, string>>({})
const formError = ref('')

function openGrade(e: RosterEntry) {
  grading.value = e
  form.grade = String(e.grade ?? 4)
  form.feedback = e.feedback
  fieldErrors.value = {}
  formError.value = ''
}

const save = useMutation({
  mutationFn: () => teacher.grade(grading.value!.submission_id!, Number(form.grade), form.feedback),
  onSuccess: async () => {
    grading.value = null
    await qc.invalidateQueries({ queryKey: ['submissions', id] })
  },
  onError: (e) => {
    const err = ref('')
    applyApiError(e, (f) => (fieldErrors.value = f), err)
    formError.value = err.value
  },
})
</script>

<template>
  <div class="space-y-4">
    <BackLink to="/teacher/assignments">{{ uz.assignments.title }}</BackLink>
    <p v-if="isPending" class="text-muted-foreground text-sm">{{ uz.common.loading }}</p>
    <template v-else-if="data">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 class="text-xl font-semibold">{{ data.assignment.title }}</h1>
          <p class="text-muted-foreground text-sm">
            {{ data.assignment.group_name }} · {{ uz.assignments.deadline }}: {{ formatDateTime(data.assignment.deadline) }}
          </p>
          <p v-if="data.assignment.description" class="pt-2 text-sm whitespace-pre-line">{{ data.assignment.description }}</p>
          <a v-if="data.assignment.has_file" :href="`/api/files/assignments/${data.assignment.id}`" class="text-sm underline">{{ uz.assignments.download }}</a>
        </div>
        <Button as="a" :href="`/api/teacher/assignments/${id}/grades.xlsx`" variant="outline" download>{{ uz.assignments.exportGrades }}</Button>
      </div>

      <h2 class="font-medium">{{ uz.assignments.submitted }} ({{ data.submitted.length }})</h2>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{{ uz.report.student }}</TableHead>
            <TableHead>{{ uz.assignments.submittedAt }}</TableHead>
            <TableHead>{{ uz.assignments.grade }}</TableHead>
            <TableHead class="w-56 text-right">{{ uz.common.actions }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="e in data.submitted" :key="e.student_id">
            <TableCell>{{ e.full_name }}</TableCell>
            <TableCell class="whitespace-nowrap">{{ formatDateTime(e.submitted_at) }}</TableCell>
            <TableCell>
              <Badge v-if="e.grade !== null">{{ e.grade.toFixed(1) }}</Badge>
              <span v-else class="text-muted-foreground text-xs">{{ uz.assignments.notGraded }}</span>
              <p v-if="e.feedback" class="text-muted-foreground max-w-xs truncate text-xs" :title="e.feedback">{{ e.feedback }}</p>
            </TableCell>
            <TableCell class="space-x-2 text-right">
              <Button v-if="e.has_file" as="a" :href="`/api/files/submissions/${e.submission_id}`" variant="outline" size="sm" download>{{ uz.assignments.download }}</Button>
              <Button size="sm" @click="openGrade(e)">{{ uz.assignments.saveGrade }}</Button>
            </TableCell>
          </TableRow>
          <TableRow v-if="!data.submitted.length">
            <TableCell colspan="4" class="text-muted-foreground text-center">{{ uz.common.empty }}</TableCell>
          </TableRow>
        </TableBody>
      </Table>

      <h2 class="font-medium">{{ uz.assignments.notSubmitted }} ({{ data.missing.length }})</h2>
      <ul class="text-sm">
        <li v-for="e in data.missing" :key="e.student_id" class="border-b py-1.5">{{ e.full_name }}</li>
        <li v-if="!data.missing.length" class="text-muted-foreground">{{ uz.common.empty }}</li>
      </ul>
    </template>

    <Dialog :open="!!grading" @update:open="(v: boolean) => !v && (grading = null)">
      <DialogContent>
        <DialogHeader><DialogTitle>{{ grading?.full_name }}</DialogTitle></DialogHeader>
        <form class="space-y-4" novalidate @submit.prevent="save.mutate()">
          <FormField :label="uz.assignments.grade" :error="fieldErrors.grade" for="grade">
            <SelectField id="grade" v-model="form.grade" :options="GRADE_OPTIONS" />
            <p class="text-muted-foreground text-xs">{{ uz.assignments.gradeHint }}</p>
          </FormField>
          <FormField :label="uz.assignments.feedback" :error="fieldErrors.feedback" for="fb">
            <textarea id="fb" v-model="form.feedback" rows="4" class="border-input dark:bg-input/30 w-full rounded-lg border bg-transparent px-2.5 py-1.5 text-sm" />
          </FormField>
          <p v-if="formError" class="text-destructive text-sm" role="alert">{{ formError }}</p>
          <DialogFooter>
            <Button type="button" variant="outline" @click="grading = null">{{ uz.common.cancel }}</Button>
            <Button type="submit" :disabled="save.isPending.value">{{ uz.assignments.saveGrade }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>
