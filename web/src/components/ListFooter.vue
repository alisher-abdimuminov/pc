<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import { uz } from '@/locales/uz'

const props = defineProps<{ pageSize: number; total: number; rowCount: number }>()
const page = defineModel<number>('page', { required: true })
const from = computed(() => (props.rowCount ? (page.value - 1) * props.pageSize + 1 : 0))
const to = computed(() => (page.value - 1) * props.pageSize + props.rowCount)
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-2 pt-4 text-sm">
    <span class="text-muted-foreground" data-testid="showing">
      {{ uz.users.showing }} <b class="text-foreground">{{ from }}–{{ to }}</b> / <b class="text-foreground">{{ total }}</b>
    </span>
    <div class="flex gap-2">
      <Button variant="outline" :disabled="page <= 1" @click="page--">{{ uz.common.prev }}</Button>
      <Button :disabled="to >= total" @click="page++">{{ uz.common.next }}</Button>
    </div>
  </div>
</template>
