<script setup lang="ts">
import { parseDate, type DateValue } from '@internationalized/date'
import { CalendarIcon, XIcon } from '@lucide/vue'
import { computed, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'
import { uz } from '@/locales/uz'

defineOptions({ inheritAttrs: false })

/** Date picker with a "YYYY-MM-DD" string model ('' = no date). */
const props = defineProps<{ modelValue: string; id?: string; placeholder?: string; clearable?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const open = ref(false)

const value = computed<DateValue | undefined>(() => {
  try {
    return props.modelValue ? parseDate(props.modelValue) : undefined
  } catch {
    return undefined
  }
})

/** 2026-09-05 -> 05.09.2026 */
const label = computed(() => (props.modelValue ? props.modelValue.split('-').reverse().join('.') : ''))

function pick(v: DateValue | undefined) {
  if (v) emit('update:modelValue', v.toString())
  open.value = false
}
</script>

<template>
  <div :class="cn('inline-flex items-center gap-1', $attrs.class as string)">
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        :id="id"
        type="button"
        variant="outline"
        :class="cn('min-w-40 flex-1 justify-start font-normal', !label && 'text-muted-foreground')"
      >
        <CalendarIcon class="size-4" aria-hidden="true" />
        {{ label || placeholder || uz.common.pickDate }}
      </Button>
    </PopoverTrigger>
    <PopoverContent class="w-auto p-0" align="start">
      <Calendar :model-value="value" :week-starts-on="1" :weekday-labels="[...uz.student.weekdays]" initial-focus @update:model-value="pick">
        <template #calendar-heading="{ date }">
          <span class="text-sm font-medium">{{ uz.student.months[date.month - 1] }} {{ date.year }}</span>
        </template>
      </Calendar>
    </PopoverContent>
  </Popover>
  <Button v-if="clearable && modelValue" type="button" variant="ghost" size="icon" :aria-label="uz.common.clear" @click="emit('update:modelValue', '')">
    <XIcon class="size-4" aria-hidden="true" />
  </Button>
  </div>
</template>
