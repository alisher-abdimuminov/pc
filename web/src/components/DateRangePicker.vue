<script setup lang="ts">
import { parseDate, type DateValue } from '@internationalized/date'
import { CalendarIcon, XIcon } from '@lucide/vue'
import { computed, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { RangeCalendar } from '@/components/ui/range-calendar'
import { cn } from '@/lib/utils'
import { uz } from '@/locales/uz'

defineOptions({ inheritAttrs: false })

/** Date range picker with "YYYY-MM-DD" string models for both ends ('' = open end). */
const props = defineProps<{ from: string; to: string; placeholder?: string; clearable?: boolean }>()
const emit = defineEmits<{ 'update:from': [value: string]; 'update:to': [value: string] }>()

const open = ref(false)

function parse(v: string): DateValue | undefined {
  try {
    return v ? parseDate(v) : undefined
  } catch {
    return undefined
  }
}
const value = computed(() => ({ start: parse(props.from), end: parse(props.to) }))

/** 2026-09-05 -> 05.09.2026 */
const fmt = (v: string) => v.split('-').reverse().join('.')
const label = computed(() => {
  if (!props.from && !props.to) return ''
  return `${props.from ? fmt(props.from) : '…'} – ${props.to ? fmt(props.to) : '…'}`
})

function pick(r: { start?: DateValue; end?: DateValue }) {
  emit('update:from', r.start?.toString() ?? '')
  emit('update:to', r.end?.toString() ?? '')
  if (r.start && r.end) open.value = false
}
function clear() {
  emit('update:from', '')
  emit('update:to', '')
}
</script>

<template>
  <div :class="cn('flex items-center gap-1', $attrs.class as string)">
    <Popover v-model:open="open">
      <PopoverTrigger as-child>
        <Button type="button" variant="outline" :class="cn('min-w-40 flex-1 justify-start font-normal', !label && 'text-muted-foreground')" data-testid="range-trigger">
          <CalendarIcon class="size-4" aria-hidden="true" />
          {{ label || placeholder || uz.common.pickDate }}
        </Button>
      </PopoverTrigger>
      <PopoverContent class="w-auto p-0" align="start">
        <RangeCalendar :model-value="value" :week-starts-on="1" :weekday-labels="[...uz.student.weekdays]" initial-focus @update:model-value="pick">
          <template #calendar-heading="{ date }">
            <span class="text-sm font-medium">{{ uz.student.months[date.month - 1] }} {{ date.year }}</span>
          </template>
        </RangeCalendar>
      </PopoverContent>
    </Popover>
    <Button v-if="clearable && (from || to)" type="button" variant="ghost" size="icon" :aria-label="uz.common.clear" @click="clear">
      <XIcon class="size-4" aria-hidden="true" />
    </Button>
  </div>
</template>
