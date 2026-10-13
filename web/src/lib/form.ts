import type { Ref } from 'vue'
import { ApiError } from '@/api/client'
import { uz } from '@/locales/uz'

/** Maps a backend error onto form fields; returns a form-level message otherwise. */
export function applyApiError(
  e: unknown,
  setErrors: (fields: Record<string, string>) => void,
  formError: Ref<string>,
) {
  if (e instanceof ApiError) {
    if (e.body.fields && Object.keys(e.body.fields).length) setErrors(e.body.fields)
    formError.value = e.body.message
  } else {
    formError.value = uz.common.unexpectedError
  }
}

export function errorMessage(e: unknown): string {
  return e instanceof ApiError ? e.body.message : uz.common.unexpectedError
}
