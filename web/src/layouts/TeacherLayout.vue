<script setup lang="ts">
import { BookOpen, UsersRound } from '@lucide/vue'
import type { NavLink } from '@/components/AppSidebar.vue'
import RoleLayout from '@/components/RoleLayout.vue'
import { uz } from '@/locales/uz'

const links: NavLink[] = [
  { to: '/teacher', label: uz.nav.myGroups, icon: UsersRound, exact: true },
  { to: '/teacher/assignments', label: uz.nav.assignments, icon: BookOpen },
]
</script>

<template>
  <RoleLayout :title="uz.roles.teacher" :links="links" />
</template>
