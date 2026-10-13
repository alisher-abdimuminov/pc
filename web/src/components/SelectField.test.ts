import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import SelectField from './SelectField.vue'

const options = [
  { value: 'a', label: 'Birinchi' },
  { value: 'b', label: 'Ikkinchi' },
  { value: 'c', label: 'Uchinchi' },
]

const body = () => within(document.body)
function within(el: HTMLElement) {
  return {
    items: () => [...el.querySelectorAll<HTMLElement>('[role="option"]')],
    labels: () => [...el.querySelectorAll<HTMLElement>('[role="option"]')].map((o) => o.textContent?.trim()),
    search: () => el.querySelector<HTMLInputElement>('[data-testid="select-search"]'),
    empty: () => el.querySelector('[data-testid="select-empty"]'),
  }
}

const mounted: VueWrapper[] = []

async function open(props: Record<string, unknown>, attrs: Record<string, unknown> = {}) {
  const w = mount(SelectField, { props: { modelValue: '', options, ...props }, attrs, attachTo: document.body })
  mounted.push(w)
  await w.get('[data-slot="select-trigger"]').trigger('click')
  await flushPromises()
  return w
}

async function type(value: string) {
  const input = body().search()!
  input.value = value
  input.dispatchEvent(new Event('input', { bubbles: true }))
  await flushPromises()
}

afterEach(() => {
  mounted.splice(0).forEach((w) => w.unmount())
  document.body.innerHTML = ''
})

describe('SelectField', () => {
  it('shows the placeholder while nothing is chosen, and the label once chosen', () => {
    const empty = mount(SelectField, { props: { modelValue: '', options, placeholder: 'Tanlang' } })
    expect(empty.find('[data-slot="select-trigger"]').text()).toContain('Tanlang')
    const chosen = mount(SelectField, { props: { modelValue: 'b', options, placeholder: 'Tanlang' } })
    expect(chosen.find('[data-slot="select-trigger"]').text()).toContain('Ikkinchi')
  })

  it('applies a width class from the parent to the trigger', () => {
    const w = mount(SelectField, { props: { modelValue: '', options }, attrs: { class: 'w-44' } })
    const cls = w.find('[data-slot="select-trigger"]').classes()
    expect(cls).toContain('w-44')
    expect(cls).not.toContain('w-full') // tailwind-merge lets the parent's width win
  })

  it('opens with a search box and every option', async () => {
    await open({ placeholder: 'Hammasi' })
    expect(body().search()).not.toBeNull()
    expect(body().labels()).toEqual(['Hammasi', 'Birinchi', 'Ikkinchi', 'Uchinchi'])
  })

  it('filters the options by what is typed, ignoring case', async () => {
    await open({})
    await type('IKK')
    expect(body().labels()).toEqual(['Ikkinchi'])
    await type('chi') // matches anywhere in the label
    expect(body().labels()).toEqual(['Birinchi', 'Ikkinchi', 'Uchinchi'])
  })

  it('says so when nothing matches and hides the clear item while searching', async () => {
    await open({ placeholder: 'Hammasi' })
    await type('zzz')
    expect(body().items()).toHaveLength(0)
    expect(body().empty()?.textContent).toContain("Ma'lumot topilmadi")
    await type('bir')
    expect(body().labels()).toEqual(['Birinchi'])
    expect(body().empty()).toBeNull()
  })

  it('opens with an empty search even when a value is already chosen', async () => {
    await open({ modelValue: 'b', placeholder: 'Hammasi' })
    expect(body().search()!.value).toBe('')
    expect(body().labels()).toEqual(['Hammasi', 'Birinchi', 'Ikkinchi', 'Uchinchi'])
  })

  it('emits the value of the option picked from the filtered list', async () => {
    const w = await open({ modelValue: 'b', placeholder: 'Hammasi' })
    await type('uch')
    body().items()[0]!.click()
    await flushPromises()
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual(['c'])
  })

  it('emits an empty string for the clear item', async () => {
    const w = await open({ modelValue: 'b', placeholder: 'Hammasi' })
    body().items().find((o) => o.textContent?.trim() === 'Hammasi')!.click()
    await flushPromises()
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([''])
  })

  it('forgets the search when the list is reopened', async () => {
    const w = await open({})
    await type('ikk')
    await w.get('[data-slot="select-trigger"]').trigger('click') // closes
    await flushPromises()
    await w.get('[data-slot="select-trigger"]').trigger('click')
    await flushPromises()
    expect(body().labels()).toHaveLength(3)
  })
})
