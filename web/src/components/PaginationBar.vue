<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import {
  Pagination, PaginationContent, PaginationEllipsis, PaginationItem, PaginationNext, PaginationPrevious,
} from '@/components/ui/pagination'
import { uz } from '@/locales/uz'

const props = defineProps<{ page: number; pageSize: number; total: number }>()
const emit = defineEmits<{ 'update:page': [page: number] }>()
const multiple = computed(() => props.total > props.pageSize)
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-2 pt-3 text-sm">
    <span class="text-muted-foreground">{{ uz.common.total }}: {{ total }}</span>
    <Pagination
      v-if="multiple"
      v-slot="{ page: current }"
      :page="page"
      :total="total"
      :items-per-page="pageSize"
      :sibling-count="1"
      show-edges
      class="mx-0 w-auto"
      @update:page="emit('update:page', $event)"
    >
      <PaginationContent v-slot="{ items }">
        <PaginationPrevious :aria-label="uz.common.prev" />
        <template v-for="(item, i) in items" :key="i">
          <PaginationItem v-if="item.type === 'page'" :value="item.value" as-child>
            <Button size="icon" :variant="item.value === current ? 'default' : 'outline'" :aria-label="`${item.value}`" :aria-current="item.value === current ? 'page' : undefined">
              {{ item.value }}
            </Button>
          </PaginationItem>
          <PaginationEllipsis v-else :index="i" />
        </template>
        <PaginationNext :aria-label="uz.common.next" />
      </PaginationContent>
    </Pagination>
  </div>
</template>
