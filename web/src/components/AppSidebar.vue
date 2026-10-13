<script setup lang="ts">
import { LogOut } from '@lucide/vue'
import { watch, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Sidebar, SidebarContent, SidebarFooter, SidebarGroup, SidebarGroupContent, SidebarHeader, SidebarMenu,
  SidebarMenuButton, SidebarMenuItem, SidebarRail, useSidebar,
} from '@/components/ui/sidebar'
import { uz } from '@/locales/uz'
import { useSessionStore } from '@/stores/session'

export interface NavLink {
  to: string
  label: string
  icon: Component
  /** match the path exactly (for section home pages) instead of by prefix */
  exact?: boolean
}

defineProps<{ title: string; links: NavLink[] }>()

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const { setOpenMobile } = useSidebar()

const isActive = (l: NavLink) => (l.exact ? route.path === l.to : route.path === l.to || route.path.startsWith(`${l.to}/`))

// the mobile drawer closes itself after navigating
watch(() => route.path, () => setOpenMobile(false))

async function onLogout() {
  await session.logout()
  await router.push('/login')
}
</script>

<template>
  <Sidebar collapsible="icon" v-bind="$attrs">
    <SidebarHeader>
      <div class="flex items-center gap-2 px-2 py-1.5">
        <div class="bg-primary text-primary-foreground flex size-8 shrink-0 items-center justify-center rounded-lg text-sm font-bold" aria-hidden="true">A</div>
        <div class="grid min-w-0 flex-1 leading-tight group-data-[collapsible=icon]:hidden">
          <span class="truncate text-sm font-semibold">{{ uz.appName }}</span>
          <span class="text-muted-foreground truncate text-xs">{{ title }}</span>
        </div>
      </div>
    </SidebarHeader>

    <SidebarContent>
      <SidebarGroup>
        <SidebarGroupContent>
          <SidebarMenu>
            <SidebarMenuItem v-for="l in links" :key="l.to">
              <SidebarMenuButton as-child :is-active="isActive(l)" :tooltip="l.label">
                <RouterLink :to="l.to" :aria-current="isActive(l) ? 'page' : undefined">
                  <component :is="l.icon" aria-hidden="true" />
                  <span>{{ l.label }}</span>
                </RouterLink>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>

    <SidebarFooter>
      <div class="px-2 py-1 group-data-[collapsible=icon]:hidden">
        <p class="truncate text-sm font-medium" data-testid="sidebar-user">{{ session.user?.full_name || session.user?.username }}</p>
        <p class="text-muted-foreground truncate text-xs">{{ session.user ? uz.roles[session.user.role] : '' }}</p>
      </div>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton :tooltip="uz.common.logout" data-testid="sidebar-logout" @click="onLogout">
            <LogOut aria-hidden="true" />
            <span>{{ uz.common.logout }}</span>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarFooter>
    <SidebarRail />
  </Sidebar>
</template>
