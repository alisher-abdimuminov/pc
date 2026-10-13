<script setup lang="ts">
import { ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { uz } from '@/locales/uz'

const props = defineProps<{ username: string; password: string }>()
const open = defineModel<boolean>('open', { required: true })
const copied = ref(false)

watch(open, () => (copied.value = false))

async function copy() {
  try {
    await navigator.clipboard.writeText(props.password)
    copied.value = true
  } catch {
    copied.value = false
  }
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{{ uz.users.passwordTitle }}</DialogTitle>
        <DialogDescription>{{ uz.users.passwordHint }}</DialogDescription>
      </DialogHeader>
      <dl class="space-y-2 text-sm">
        <div class="flex justify-between gap-4">
          <dt class="text-muted-foreground">{{ uz.users.username }}</dt>
          <dd class="font-mono">{{ username }}</dd>
        </div>
        <div class="flex items-center justify-between gap-4">
          <dt class="text-muted-foreground">{{ uz.login.password }}</dt>
          <dd class="font-mono text-base font-semibold select-all" data-testid="generated-password">{{ password }}</dd>
        </div>
      </dl>
      <DialogFooter>
        <Button variant="outline" @click="copy">{{ copied ? uz.common.copied : uz.common.copy }}</Button>
        <Button @click="open = false">{{ uz.common.close }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
