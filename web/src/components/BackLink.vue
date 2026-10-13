<script setup lang="ts">
import { ArrowLeft } from '@lucide/vue'

defineProps<{ to: string }>()
</script>

<template>
  <RouterLink :to="to" class="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-sm hover:underline">
    <ArrowLeft class="size-4" aria-hidden="true" />
    <slot />
  </RouterLink>
</template>
