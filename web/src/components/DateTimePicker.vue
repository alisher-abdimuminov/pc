<script setup lang="ts">
import { computed } from 'vue'
import DatePicker from '@/components/DatePicker.vue'
import { Input } from '@/components/ui/input'

/** "YYYY-MM-DDTHH:mm" string model: a calendar for the day plus a time field. */
const props = defineProps<{ modelValue: string; id?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const date = computed(() => props.modelValue.slice(0, 10))
const time = computed(() => props.modelValue.slice(11, 16) || '09:00')

const set = (d: string, t: string) => emit('update:modelValue', d ? `${d}T${t}` : '')
</script>

<template>
  <div class="flex gap-2">
    <DatePicker :id="id" :model-value="date" class="flex-1" @update:model-value="set($event, time)" />
    <Input :model-value="time" type="time" class="w-28" aria-label="Soat" @update:model-value="set(date, String($event))" />
  </div>
</template>
