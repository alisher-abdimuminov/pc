import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PaginationBar from './PaginationBar.vue'

describe('PaginationBar', () => {
  it('shows the total but no controls for a single page', () => {
    const w = mount(PaginationBar, { props: { page: 1, pageSize: 20, total: 5 } })
    expect(w.text()).toContain('5')
    expect(w.find('[data-slot="pagination"]').exists()).toBe(false)
  })

  it('renders page buttons with the current one marked and emits the chosen page', async () => {
    const w = mount(PaginationBar, { props: { page: 3, pageSize: 10, total: 100 } })
    const current = w.find('[aria-current="page"]')
    expect(current.exists()).toBe(true)
    expect(current.text()).toBe('3')
    expect(w.find('[data-slot="pagination-ellipsis"]').exists()).toBe(true) // 10 pages: collapsed with ellipsis
    await w.find('button[aria-label="4"]').trigger('click')
    expect(w.emitted('update:page')?.[0]).toEqual([4])
  })

  it('first page: previous is disabled, next moves forward', async () => {
    const w = mount(PaginationBar, { props: { page: 1, pageSize: 10, total: 30 } })
    expect(w.find('[data-slot="pagination-previous"]').attributes('disabled')).toBeDefined()
    await w.find('[data-slot="pagination-next"]').trigger('click')
    expect(w.emitted('update:page')?.[0]).toEqual([2])
  })
})
