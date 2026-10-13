<script setup lang="ts">
import { BookOpen, CalendarDays, House, User } from '@lucide/vue'
import type { NavLink } from '@/components/AppSidebar.vue'
import RoleLayout from '@/components/RoleLayout.vue'
import { uz } from '@/locales/uz'

const links: NavLink[] = [
  { to: '/student', label: uz.nav.studentHome, icon: House, exact: true },
  { to: '/student/attendance', label: uz.nav.studentAttendance, icon: CalendarDays },
  { to: '/student/assignments', label: uz.nav.assignments, icon: BookOpen },
  { to: '/student/profile', label: uz.nav.profile, icon: User },
]
</script>

<template>
  <!-- phones: floating pill navigation; tablets and desktops: the sidebar -->
  <RoleLayout :title="uz.roles.student" :links="links" bottom-nav content-class="bg-muted/40 mx-auto min-h-svh w-full max-w-2xl space-y-4 p-4 pb-28 md:min-h-0 md:bg-transparent md:p-6 md:pb-6">
    <template #bottom>
      <nav class="pointer-events-none fixed inset-x-0 bottom-0 z-20 px-4 pb-[max(env(safe-area-inset-bottom),0.75rem)] md:hidden" aria-label="Asosiy menyu">
        <ul class="bg-background pointer-events-auto mx-auto flex max-w-md items-center justify-between gap-1 rounded-full border p-1.5 shadow-lg">
          <li v-for="l in links" :key="l.to">
            <RouterLink
              :to="l.to"
              :aria-label="l.label"
              :title="l.label"
              :exact-active-class="l.exact ? 'nav-active' : ''"
              :active-class="l.exact ? '' : 'nav-active'"
              class="text-muted-foreground flex h-11 min-w-11 items-center justify-center gap-2 rounded-full px-3 text-sm font-medium transition-colors [&.nav-active]:bg-muted [&.nav-active]:text-foreground [&.nav-active]:px-5"
            >
              <component :is="l.icon" class="size-5 shrink-0" aria-hidden="true" />
              <span class="hidden [.nav-active_&]:inline">{{ l.label }}</span>
            </RouterLink>
          </li>
        </ul>
      </nav>
    </template>
  </RoleLayout>
</template>
