import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import PermissionPicker from './PermissionPicker.vue'

const api = vi.hoisted(() => ({ permissionCatalog: vi.fn() }))
vi.mock('@/api/admin', () => api)

const catalog = [
  { key: 'faculties:view', requires: [] },
  { key: 'faculties:create', requires: ['faculties:view'] },
  { key: 'groups:view', requires: ['faculties:view'] },
  { key: 'groups:create', requires: ['faculties:view', 'groups:view'] },
  { key: 'audit:view', requires: [] },
]

async function render(initial: string[] = []) {
  api.permissionCatalog.mockResolvedValue(catalog)
  const value = ref<string[]>(initial)
  const Host = defineComponent({
    components: { PermissionPicker },
    setup: () => ({ value }),
    template: '<PermissionPicker v-model="value" />',
  })
  const w = mount(Host, { global: { plugins: [[VueQueryPlugin, { queryClient: new QueryClient() }]] } })
  await flushPromises()
  return { w, value }
}
const box = (w: ReturnType<typeof mount>, key: string) => w.get(`[data-testid="perm-${key}"]`)
const sorted = (v: string[]) => [...v].sort()

beforeEach(() => vi.clearAllMocks())

describe('PermissionPicker', () => {
  it('shows one row per section with only the actions that exist', async () => {
    const { w } = await render()
    expect(w.findAll('[data-section]').map((r) => r.attributes('data-section'))).toEqual(['faculties', 'groups', 'audit'])
    expect(w.find('[data-testid="perm-audit:view"]').exists()).toBe(true)
    expect(w.find('[data-testid="perm-audit:create"]').exists()).toBe(false)
  })

  it('ticking an action also ticks what it depends on', async () => {
    const { w, value } = await render()
    await box(w, 'groups:create').trigger('click')
    expect(sorted(value.value)).toEqual(['faculties:view', 'groups:create', 'groups:view'])
  })

  it('unticking a permission also unticks what depends on it', async () => {
    const { w, value } = await render(['faculties:view', 'faculties:create', 'groups:view', 'audit:view'])
    await box(w, 'faculties:view').trigger('click')
    expect(sorted(value.value)).toEqual(['audit:view']) // groups:view and faculties:create needed it
  })

  it('reflects the current grants', async () => {
    const { w } = await render(['audit:view'])
    expect(box(w, 'audit:view').attributes('aria-checked')).toBe('true')
    expect(box(w, 'groups:view').attributes('aria-checked')).toBe('false')
  })
})
