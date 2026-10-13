<script setup lang="ts">
import { Check, X } from '@lucide/vue'

/** A passed/failed mark: a Lucide check or x icon, with the meaning as screen-reader text. */
defineProps<{ ok: boolean; label: string }>()
</script>

<template>
  <span class="inline-flex items-center" :data-ok="ok">
    <Check v-if="ok" class="size-4 text-emerald-600 dark:text-emerald-400" aria-hidden="true" />
    <X v-else class="size-4 text-rose-600 dark:text-rose-400" aria-hidden="true" />
    <span class="sr-only">{{ label }}</span>
  </span>
</template>
