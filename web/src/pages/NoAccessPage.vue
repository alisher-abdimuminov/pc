<script setup lang="ts">
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { uz } from '@/locales/uz'
import { useSessionStore } from '@/stores/session'

const session = useSessionStore()
const router = useRouter()

async function onLogout() {
  await session.logout()
  await router.push('/login')
}
</script>

<template>
  <main class="mx-auto flex min-h-svh max-w-md flex-col items-center justify-center gap-4 p-6 text-center">
    <h1 class="text-xl font-semibold">{{ uz.noAccess.title }}</h1>
    <p class="text-muted-foreground text-sm">{{ uz.noAccess.hint }}</p>
    <Button variant="outline" data-testid="no-access-logout" @click="onLogout">{{ uz.common.logout }}</Button>
  </main>
</template>
