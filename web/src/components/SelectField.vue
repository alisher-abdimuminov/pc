<script setup lang="ts">
import { Check, ChevronDown, Search } from '@lucide/vue'
import {
  ComboboxAnchor,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxItemIndicator,
  ComboboxPortal,
  ComboboxRoot,
  ComboboxViewport,
} from 'reka-ui'
import { computed, ref, useTemplateRef } from 'vue'
import { cn } from '@/lib/utils'
import { uz } from '@/locales/uz'

export interface Option {
  value: string
  label: string
}

defineOptions({ inheritAttrs: false })

/**
 * A searchable select with a plain string model. An empty string means "nothing chosen" and shows the
 * placeholder; when a placeholder is given it is also offered as the item that clears the choice.
 * The list is filtered on the client by the label as you type.
 */
const props = defineProps<{ modelValue: string | null | undefined; options: Option[]; placeholder?: string; id?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const CLEAR = '__clear__' // an item value that stands for "nothing chosen"

const open = ref(false)
const trigger = useTemplateRef<HTMLButtonElement>('trigger')
const term = ref('')

const selected = computed(() => props.options.find((o) => o.value === props.modelValue))
const searching = computed(() => term.value.trim() !== '')
const shown = computed(() => {
  const q = term.value.trim().toLocaleLowerCase()
  return q ? props.options.filter((o) => o.label.toLocaleLowerCase().includes(q)) : props.options
})

function onChange(v: unknown) {
  emit('update:modelValue', v === CLEAR || v == null ? '' : String(v))
}
// the trigger toggles the list itself; a press on it must not also count as an outside click that closes it
function onOutside(e: Event) {
  const target = (e as CustomEvent<{ originalEvent?: Event }>).detail?.originalEvent?.target ?? e.target
  if (target instanceof Node && trigger.value?.contains(target)) e.preventDefault()
}
function onOpen(v: boolean) {
  open.value = v
  if (!v) term.value = ''
}
</script>

<template>
  <ComboboxRoot
    :model-value="props.modelValue || undefined"
    :open="open"
    ignore-filter
    @update:model-value="onChange"
    @update:open="onOpen"
  >
    <ComboboxAnchor as-child>
      <button
        :id="id"
        ref="trigger"
        type="button"
        role="combobox"
        aria-haspopup="listbox"
        :aria-expanded="open"
        data-slot="select-trigger"
        :data-placeholder="selected ? undefined : ''"
        :class="
          cn(
            'bg-input/50 border-transparent data-placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/30 aria-invalid:ring-destructive/20 aria-invalid:border-destructive flex h-9 w-full items-center justify-between gap-1.5 rounded-3xl border px-3 py-2 text-sm whitespace-nowrap transition-[color,box-shadow,background-color] outline-none focus-visible:ring-3 disabled:cursor-not-allowed disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0',
            $attrs.class as string,
          )
        "
        @click="onOpen(!open)"
        @keydown.down.prevent="onOpen(true)"
      >
        <span class="line-clamp-1 text-left">{{ selected?.label ?? placeholder ?? '' }}</span>
        <ChevronDown class="text-muted-foreground size-4" aria-hidden="true" />
      </button>
    </ComboboxAnchor>

    <ComboboxPortal>
      <ComboboxContent
        position="popper"
        :side-offset="4"
        align="start"
        data-slot="select-content"
        @pointer-down-outside="onOutside"
        class="bg-popover text-popover-foreground ring-foreground/5 dark:ring-foreground/10 z-50 w-(--reka-combobox-trigger-width) min-w-48 overflow-hidden rounded-3xl shadow-lg ring-1"
      >
        <div class="relative border-b p-2">
          <Search class="text-muted-foreground pointer-events-none absolute top-1/2 left-5 size-4 -translate-y-1/2" aria-hidden="true" />
          <ComboboxInput
            v-model="term"
            :display-value="() => ''"
            auto-focus
            :placeholder="uz.common.search"
            data-testid="select-search"
            class="bg-input/50 placeholder:text-muted-foreground h-8 w-full rounded-full border-0 pr-3 pl-8 text-sm outline-none"
          />
        </div>
        <ComboboxViewport class="max-h-60 overflow-y-auto p-1">
          <ComboboxItem
            v-if="placeholder !== undefined && !searching"
            :value="CLEAR"
            class="text-muted-foreground data-highlighted:bg-accent data-highlighted:text-accent-foreground relative flex cursor-default items-center rounded-2xl py-2 pr-8 pl-3 text-sm font-medium outline-hidden select-none"
          >{{ placeholder }}</ComboboxItem>
          <ComboboxItem
            v-for="o in shown"
            :key="o.value"
            :value="o.value"
            class="data-highlighted:bg-accent data-highlighted:text-accent-foreground relative flex cursor-default items-center rounded-2xl py-2 pr-8 pl-3 text-sm font-medium outline-hidden select-none"
          >
            {{ o.label }}
            <span class="absolute right-2 flex size-4 items-center justify-center">
              <ComboboxItemIndicator><Check class="size-4" aria-hidden="true" /></ComboboxItemIndicator>
            </span>
          </ComboboxItem>
          <ComboboxEmpty v-if="!shown.length" class="text-muted-foreground px-3 py-4 text-center text-sm" data-testid="select-empty">{{ uz.common.empty }}</ComboboxEmpty>
        </ComboboxViewport>
      </ComboboxContent>
    </ComboboxPortal>
  </ComboboxRoot>
</template>
