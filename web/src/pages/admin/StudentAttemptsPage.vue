<script setup lang="ts">
import BackLink from '@/components/BackLink.vue'
import { useRoute } from 'vue-router'
import AttemptsView from '@/components/AttemptsView.vue'
import { uz } from '@/locales/uz'

const sid = useRoute().params.id as string
</script>

<template>
  <div class="space-y-4">
    <BackLink to="/admin/reports">{{ uz.report.title }}</BackLink>
    <AttemptsView as="admin" :url="`/api/admin/students/${sid}/attempts`" :cache-key="`admin-${sid}`" />
  </div>
</template>
