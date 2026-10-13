<script setup lang="ts">
import { Building2, ChartColumn, ScanFace, CalendarRange, LayoutDashboard, MapPin, ScrollText, UserCog, UsersRound } from '@lucide/vue'
import { computed, type Component } from 'vue'
import type { NavLink } from '@/components/AppSidebar.vue'
import RoleLayout from '@/components/RoleLayout.vue'
import { ADMIN_SECTIONS, hasPermission } from '@/lib/permissions'
import { uz } from '@/locales/uz'
import { useSessionStore } from '@/stores/session'

const icons: Record<(typeof ADMIN_SECTIONS)[number]['key'], Component> = {
  dashboard: LayoutDashboard,
  faculties: Building2,
  groups: UsersRound,
  users: UserCog,
  locations: MapPin,
  schedules: CalendarRange,
  reports: ChartColumn,
  attempts: ScanFace,
  audit: ScrollText,
}

const session = useSessionStore()
// a little admin sees only the sections it was granted
const links = computed<NavLink[]>(() =>
  ADMIN_SECTIONS.filter((s) => hasPermission(session.user, s.permission)).map((s) => ({
    to: s.path,
    label: uz.nav[s.key],
    icon: icons[s.key],
    exact: s.key === 'dashboard',
  })),
)
</script>

<template>
  <RoleLayout :title="session.user ? uz.roles[session.user.role] : uz.roles.admin" :links="links" />
</template>
