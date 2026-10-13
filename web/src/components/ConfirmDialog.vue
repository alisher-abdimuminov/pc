<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { uz } from '@/locales/uz'

defineProps<{ title: string; message?: string; confirmLabel?: string; loading?: boolean }>()
const open = defineModel<boolean>('open', { required: true })
defineEmits<{ confirm: [] }>()
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{{ title }}</DialogTitle>
        <DialogDescription>{{ message ?? uz.common.confirmDelete }}</DialogDescription>
      </DialogHeader>
      <DialogFooter>
        <Button variant="outline" @click="open = false">{{ uz.common.cancel }}</Button>
        <Button variant="destructive" :disabled="loading" @click="$emit('confirm')">
          {{ confirmLabel ?? uz.common.delete }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
