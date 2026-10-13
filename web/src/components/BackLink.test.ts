import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import BackLink from './BackLink.vue'

describe('BackLink', () => {
  it('renders a link with an arrow icon instead of an arrow character', () => {
    const w = mount(BackLink, {
      props: { to: '/admin/groups' },
      slots: { default: 'Guruhlar' },
      global: { stubs: { RouterLink: { template: '<a :href="to" v-bind="$attrs"><slot /></a>', props: ['to'] } } },
    })
    expect(w.attributes('href')).toBe('/admin/groups')
    expect(w.text()).toBe('Guruhlar')
    expect(w.find('svg').classes().join(' ')).toContain('lucide-arrow-left')
    expect(w.find('svg').attributes('aria-hidden')).toBe('true')
    expect(w.text()).not.toContain('←')
  })
})
