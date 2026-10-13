<script setup lang="ts">
import { Building2, ChartColumn, ScanFace, CalendarRange, LayoutDashboard, MapPin, ScrollText, UserCog, UsersRound } from '@lucide/vue'
import type { NavLink } from '@/components/AppSidebar.vue'
import RoleLayout from '@/components/RoleLayout.vue'
import { uz } from '@/locales/uz'

const links: NavLink[] = [
  { to: '/admin', label: uz.nav.dashboard, icon: LayoutDashboard, exact: true },
  { to: '/admin/faculties', label: uz.nav.faculties, icon: Building2 },
  { to: '/admin/groups', label: uz.nav.groups, icon: UsersRound },
  { to: '/admin/users', label: uz.nav.users, icon: UserCog },
  { to: '/admin/locations', label: uz.nav.locations, icon: MapPin },
  { to: '/admin/schedules', label: uz.nav.schedules, icon: CalendarRange },
  { to: '/admin/reports', label: uz.nav.reports, icon: ChartColumn },
  { to: '/admin/attempts', label: uz.nav.attempts, icon: ScanFace },
  { to: '/admin/audit', label: uz.nav.audit, icon: ScrollText },
]
</script>

<template>
  <RoleLayout :title="uz.roles.admin" :links="links" />
</template>
