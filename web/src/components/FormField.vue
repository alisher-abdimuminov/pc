<script setup lang="ts">
import { Label } from '@/components/ui/label'

defineProps<{ label: string; error?: string; for?: string }>()
</script>

<template>
  <div class="space-y-1.5">
    <Label :for="$props.for">{{ label }}</Label>
    <slot />
    <p v-if="error" class="text-destructive text-sm">{{ error }}</p>
  </div>
</template>
