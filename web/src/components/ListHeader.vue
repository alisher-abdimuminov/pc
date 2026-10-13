<script setup lang="ts">
import { Search, SlidersHorizontal } from '@lucide/vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { uz } from '@/locales/uz'

defineProps<{ title: string; total?: number; filterCount?: number }>()
const search = defineModel<string>('search', { default: '' })
defineEmits<{ clearFilters: [] }>()

// a select list opens in its own portal: clicking an option must not close the filter popover
function keepOpenForSelect(e: Event) {
  const el = (e as CustomEvent<{ originalEvent: Event }>).detail?.originalEvent?.target as HTMLElement | undefined
  if (el?.closest('[role="listbox"], [data-slot="select-content"]')) e.preventDefault()
}
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-3 pb-5">
    <div class="flex items-center gap-3">
      <h1 class="text-xl font-semibold">{{ title }}</h1>
      <Badge v-if="total !== undefined" variant="secondary" class="rounded-full" data-testid="list-total">{{ total }}</Badge>
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <div class="relative">
        <Search class="text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2" aria-hidden="true" />
        <Input v-model="search" class="w-60 pl-9" :placeholder="uz.common.search" />
      </div>
      <Popover v-if="$slots.filters">
        <PopoverTrigger as-child>
          <Button variant="outline" size="icon" class="relative" :aria-label="uz.users.filters" data-testid="filter-button">
            <SlidersHorizontal class="size-4" aria-hidden="true" />
            <span v-if="filterCount" class="bg-primary text-primary-foreground absolute -top-1.5 -right-1.5 flex size-4 items-center justify-center rounded-full text-[10px]">{{ filterCount }}</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" class="w-64 space-y-2" @interact-outside="keepOpenForSelect">
          <p class="text-sm font-medium">{{ uz.users.filters }}</p>
          <slot name="filters" />
          <Button v-if="filterCount" variant="ghost" size="sm" class="w-full" @click="$emit('clearFilters')">{{ uz.users.clearFilters }}</Button>
        </PopoverContent>
      </Popover>
      <slot name="actions" />
    </div>
  </div>
</template>
