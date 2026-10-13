import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DatePicker from './DatePicker.vue'

describe('DatePicker', () => {
  it('shows the date as dd.mm.yyyy', () => {
    const w = mount(DatePicker, { props: { modelValue: '2026-09-05' } })
    expect(w.text()).toContain('05.09.2026')
  })

  it('shows the placeholder when empty and tolerates garbage values', () => {
    expect(mount(DatePicker, { props: { modelValue: '', placeholder: 'Dan' } }).text()).toContain('Dan')
    expect(() => mount(DatePicker, { props: { modelValue: 'not-a-date' } })).not.toThrow()
  })

  it('can be cleared when clearable', async () => {
    const w = mount(DatePicker, { props: { modelValue: '2026-09-05', clearable: true } })
    await w.find('button[aria-label="Tozalash"]').trigger('click')
    expect(w.emitted('update:modelValue')?.[0]).toEqual([''])
    expect(mount(DatePicker, { props: { modelValue: '2026-09-05' } }).find('button[aria-label="Tozalash"]').exists()).toBe(false)
  })
})
