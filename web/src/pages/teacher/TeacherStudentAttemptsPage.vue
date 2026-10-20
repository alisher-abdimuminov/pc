<script setup lang="ts">
import { useRoute } from 'vue-router'
import AttemptsView from '@/components/AttemptsView.vue'
import BackLink from '@/components/BackLink.vue'
import { uz } from '@/locales/uz'

const route = useRoute()
const sid = route.params.id as string
const back = typeof route.query.group === 'string' ? `/teacher/groups/${route.query.group}` : '/teacher'
</script>

<template>
  <div class="space-y-4">
    <BackLink :to="back">{{ uz.teacher.groupsTitle }}</BackLink>
    <h1 class="text-xl font-semibold">{{ uz.teacher.reviewTitle }}</h1>
    <AttemptsView as="teacher" :url="`/api/teacher/students/${sid}/attempts`" :cache-key="`teacher-${sid}`" />
  </div>
</template>
