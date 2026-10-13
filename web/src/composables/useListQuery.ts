import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import { refDebounced } from '@vueuse/core'
import { computed, ref, watch, type Ref } from 'vue'
import type { ListQuery, Page } from '@/api/admin'

/** Search + pagination state with a vue-query list. `extra` holds additional filters. */
export function useListQuery<T>(
  key: string,
  fetcher: (q: ListQuery) => Promise<Page<T>>,
  extra: Ref<Record<string, string | undefined>> = ref({}),
  pageSize = 20,
) {
  const search = ref('')
  const page = ref(1)
  const debounced = refDebounced(search, 300)
  watch([debounced, extra], () => (page.value = 1), { deep: true })

  const params = computed<ListQuery>(() => ({
    page: page.value,
    page_size: pageSize,
    search: debounced.value,
    ...extra.value,
  }))
  const query = useQuery({
    queryKey: [key, params],
    queryFn: () => fetcher(params.value),
    placeholderData: keepPreviousData,
  })
  return { search, page, pageSize, query }
}
