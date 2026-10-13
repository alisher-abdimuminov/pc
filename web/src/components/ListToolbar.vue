<script setup lang="ts">
import { Input } from '@/components/ui/input'
import { uz } from '@/locales/uz'

defineProps<{ title: string }>()
const search = defineModel<string>('search', { default: '' })
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-3 pb-4">
    <h1 class="text-xl font-semibold">{{ title }}</h1>
    <div class="flex flex-wrap items-center gap-2">
      <Input v-model="search" class="w-56" :placeholder="uz.common.search" />
      <slot name="filters" />
      <slot name="actions" />
    </div>
  </div>
</template>
