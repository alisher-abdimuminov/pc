<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { permissionCatalog } from '@/api/admin'
import { Checkbox } from '@/components/ui/checkbox'
import { errorMessage } from '@/lib/form'
import { uz } from '@/locales/uz'

const model = defineModel<string[]>({ required: true })
defineProps<{ disabled?: boolean }>()

const { data, error } = useQuery({ queryKey: ['permission-catalog'], queryFn: permissionCatalog, staleTime: Infinity })

const ACTIONS = ['view', 'create', 'update', 'delete']
const sections = computed(() => {
  const by = new Map<string, Set<string>>()
  for (const p of data.value ?? []) {
    const [section, action] = p.key.split(':') as [string, string]
    if (!by.has(section)) by.set(section, new Set())
    by.get(section)!.add(action)
  }
  return [...by].map(([key, actions]) => ({ key, actions: ACTIONS.filter((a) => actions.has(a)) }))
})
const info = computed(() => new Map((data.value ?? []).map((p) => [p.key, p])))

const has = (key: string) => model.value.includes(key)

/** Ticking a permission also ticks what it depends on; unticking one also unticks what depends on it. */
function toggle(key: string, on: boolean) {
  const next = new Set(model.value)
  if (on) {
    next.add(key)
    info.value.get(key)?.requires.forEach((r) => next.add(r))
  } else {
    next.delete(key)
    for (const p of data.value ?? []) if (p.requires.includes(key)) next.delete(p.key)
  }
  model.value = [...next]
}

/** True when another ticked permission needs this one (shown as a hint, the cascade handles unticking). */
const neededByOther = (key: string) => (data.value ?? []).some((p) => has(p.key) && p.key !== key && p.requires.includes(key))
</script>

<template>
  <div class="space-y-2" data-testid="permission-picker">
    <p class="text-muted-foreground text-xs">{{ uz.permissions.hint }}</p>
    <p v-if="error" class="text-destructive text-sm" role="alert">{{ errorMessage(error) }}</p>
    <div class="overflow-hidden rounded-xl border">
      <table class="w-full text-sm">
        <thead class="bg-muted/50 text-muted-foreground text-left">
          <tr>
            <th class="px-3 py-2 font-medium">{{ uz.permissions.title }}</th>
            <th v-for="a in ACTIONS" :key="a" class="w-20 px-2 py-2 text-center font-medium">{{ uz.permissions.actions[a] }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in sections" :key="s.key" class="border-t" :data-section="s.key">
            <td class="px-3 py-2">{{ uz.permissions.sections[s.key] ?? s.key }}</td>
            <td v-for="a in ACTIONS" :key="a" class="px-2 py-2 text-center">
              <Checkbox
                v-if="s.actions.includes(a)"
                :model-value="has(`${s.key}:${a}`)"
                :disabled="disabled"
                :aria-label="`${uz.permissions.sections[s.key] ?? s.key}: ${uz.permissions.actions[a]}`"
                :data-testid="`perm-${s.key}:${a}`"
                :title="neededByOther(`${s.key}:${a}`) ? uz.permissions.requiredBy : undefined"
                @update:model-value="toggle(`${s.key}:${a}`, $event === true)"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
