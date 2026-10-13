<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppSidebar, { type NavLink } from '@/components/AppSidebar.vue'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { Separator } from '@/components/ui/separator'
import { SidebarInset, SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar'

const props = defineProps<{
  title: string
  links: NavLink[]
  /** phone-first role: on small screens the sidebar/top bar give way to a bottom navigation slot */
  bottomNav?: boolean
  contentClass?: string
}>()

const route = useRoute()
const current = computed(() => {
  const hits = props.links.filter((l) => (l.exact ? route.path === l.to : route.path === l.to || route.path.startsWith(`${l.to}/`)))
  return hits.sort((a, b) => b.to.length - a.to.length)[0]?.label ?? props.title
})
const hideOnPhone = computed(() => (props.bottomNav ? 'max-md:hidden' : ''))
</script>

<template>
  <SidebarProvider>
    <AppSidebar :title="title" :links="links" :class="hideOnPhone" />
    <SidebarInset class="min-w-0">
      <header class="bg-background/80 sticky top-0 z-10 flex h-14 shrink-0 items-center gap-2 border-b px-4 backdrop-blur" :class="hideOnPhone">
        <SidebarTrigger class="-ml-1" />
        <Separator orientation="vertical" class="h-5" />
        <h1 class="truncate text-sm font-medium" data-testid="page-label">{{ current }}</h1>
        <ThemeToggle class="ml-auto" />
      </header>
      <div :class="contentClass ?? 'mx-auto w-full max-w-7xl p-4 md:p-6'">
        <RouterView />
      </div>
      <slot name="bottom" />
    </SidebarInset>
  </SidebarProvider>
</template>
