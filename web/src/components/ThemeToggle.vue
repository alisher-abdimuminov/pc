<script setup lang="ts">
import { Moon, Sun } from '@lucide/vue'
import { useTheme } from '@/lib/theme'
import { uz } from '@/locales/uz'

const { theme, setTheme } = useTheme()
const options = [
  { value: 'light', label: uz.theme.light, icon: Sun },
  { value: 'dark', label: uz.theme.dark, icon: Moon },
] as const
</script>

<template>
  <div class="bg-background inline-flex items-center gap-1 rounded-full border p-1" role="radiogroup" :aria-label="uz.theme.label" data-testid="theme-toggle">
    <button
      v-for="o in options"
      :key="o.value"
      type="button"
      role="radio"
      :aria-checked="theme === o.value"
      :aria-label="o.label"
      :title="o.label"
      class="flex size-8 items-center justify-center rounded-full transition-colors outline-none focus-visible:ring-3 focus-visible:ring-ring/40"
      :class="theme === o.value ? 'bg-muted text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'"
      :data-theme="o.value"
      @click="setTheme(o.value)"
    >
      <component :is="o.icon" class="size-4" aria-hidden="true" />
    </button>
  </div>
</template>
