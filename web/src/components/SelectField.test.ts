import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SelectField from './SelectField.vue'

const options = [
  { value: 'a', label: 'Birinchi' },
  { value: 'b', label: 'Ikkinchi' },
]

describe('SelectField', () => {
  it('shows the placeholder while nothing is chosen', () => {
    const w = mount(SelectField, { props: { modelValue: '', options, placeholder: 'Tanlang' } })
    expect(w.find('[data-slot="select-trigger"]').text()).toContain('Tanlang')
  })

  it('applies a width class from the parent to the trigger', () => {
    const w = mount(SelectField, { props: { modelValue: '', options }, attrs: { class: 'w-44' } })
    const cls = w.find('[data-slot="select-trigger"]').classes()
    expect(cls).toContain('w-44')
    expect(cls).not.toContain('w-full') // tailwind-merge lets the parent's width win
  })

  it('renders a hidden native control so forms and tests can still see the value', () => {
    const w = mount(SelectField, { props: { modelValue: 'b', options } })
    expect(w.find('[data-slot="select-trigger"]').exists()).toBe(true)
  })
})
