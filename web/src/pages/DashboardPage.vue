<script setup lang="ts">
import { uz } from '@/locales/uz'
import { useSessionStore } from '@/stores/session'

const session = useSessionStore()
</script>

<template>
  <section class="space-y-1">
    <h1 class="text-xl font-semibold">{{ uz.dashboard.welcome }}, {{ session.user?.full_name }}</h1>
    <p class="text-muted-foreground text-sm">{{ uz.dashboard.empty }}</p>
  </section>
</template>
