import { computed, ref, type Ref } from 'vue'

/** Checkbox selection over the rows of the current page. */
export function useRowSelection(rows: Ref<{ id: string }[]>) {
  const selected = ref(new Set<string>())
  const allSelected = computed<boolean | 'indeterminate'>(() =>
    !rows.value.length || !rows.value.some((r) => selected.value.has(r.id))
      ? false
      : rows.value.every((r) => selected.value.has(r.id)) ? true : 'indeterminate',
  )
  function toggleOne(id: string, on: boolean | 'indeterminate') {
    const next = new Set(selected.value)
    if (on === true) next.add(id)
    else next.delete(id)
    selected.value = next
  }
  function toggleAll(on: boolean | 'indeterminate') {
    selected.value = on === true ? new Set(rows.value.map((r) => r.id)) : new Set()
  }
  return { selected, allSelected, toggleOne, toggleAll }
}
