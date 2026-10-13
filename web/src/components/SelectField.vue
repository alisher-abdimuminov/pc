<script setup lang="ts">
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cn } from '@/lib/utils'

export interface Option {
  value: string
  label: string
}

defineOptions({ inheritAttrs: false })

/**
 * A shadcn Select with a plain string model. An empty string means "nothing chosen" and shows the
 * placeholder; when a placeholder is given it is also offered as the item that clears the choice.
 */
const props = defineProps<{ modelValue: string | null | undefined; options: Option[]; placeholder?: string; id?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const CLEAR = '__clear__' // reka-ui forbids an empty-string item value

function onChange(v: unknown) {
  emit('update:modelValue', v === CLEAR || v == null ? '' : String(v))
}
</script>

<template>
  <Select :model-value="props.modelValue || undefined" @update:model-value="onChange">
    <SelectTrigger :id="id" :class="cn('w-full', $attrs.class as string)">
      <SelectValue :placeholder="placeholder ?? ''" />
    </SelectTrigger>
    <SelectContent>
      <SelectItem v-if="placeholder !== undefined" :value="CLEAR" class="text-muted-foreground">{{ placeholder }}</SelectItem>
      <SelectItem v-for="o in options" :key="o.value" :value="o.value">{{ o.label }}</SelectItem>
    </SelectContent>
  </Select>
</template>
