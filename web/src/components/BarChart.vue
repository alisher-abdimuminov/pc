<script setup lang="ts">
import { computed } from 'vue'
import type { DayStats } from '@/api/admin'
import { uz } from '@/locales/uz'

const props = defineProps<{ days: DayStats[] }>()

const W = 560
const H = 200
const PAD = { top: 10, right: 8, bottom: 28, left: 28 }
const max = computed(() => Math.max(1, ...props.days.map((d) => d.expected)))
const slot = computed(() => (W - PAD.left - PAD.right) / Math.max(1, props.days.length))
const y = (v: number) => PAD.top + (H - PAD.top - PAD.bottom) * (1 - v / max.value)
const segs = computed(() =>
  props.days.map((d, i) => {
    const x = PAD.left + i * slot.value + slot.value * 0.2
    const w = slot.value * 0.6
    const base = H - PAD.bottom
    const hFull = base - y(d.full)
    const hPartial = base - y(d.partial)
    const hAbsent = base - y(d.absent)
    return {
      d,
      x,
      w,
      full: { y: base - hFull, h: hFull },
      partial: { y: base - hFull - hPartial, h: hPartial },
      absent: { y: base - hFull - hPartial - hAbsent, h: hAbsent },
      label: d.date.slice(5),
    }
  }),
)
</script>

<template>
  <div>
    <svg :viewBox="`0 0 ${W} ${H}`" class="w-full" role="img" :aria-label="uz.stats.last7">
      <line :x1="PAD.left" :x2="W - PAD.right" :y1="H - PAD.bottom" :y2="H - PAD.bottom" stroke="currentColor" opacity="0.2" />
      <text :x="PAD.left - 4" :y="y(max) + 4" text-anchor="end" class="fill-current text-[10px] opacity-60">{{ max }}</text>
      <text :x="PAD.left - 4" :y="H - PAD.bottom" text-anchor="end" class="fill-current text-[10px] opacity-60">0</text>
      <g v-for="s in segs" :key="s.d.date">
        <rect :x="s.x" :y="s.full.y" :width="s.w" :height="s.full.h" class="fill-emerald-500"><title>{{ uz.stats.full }}: {{ s.d.full }}</title></rect>
        <rect :x="s.x" :y="s.partial.y" :width="s.w" :height="s.partial.h" class="fill-amber-400"><title>{{ uz.stats.partial }}: {{ s.d.partial }}</title></rect>
        <rect :x="s.x" :y="s.absent.y" :width="s.w" :height="s.absent.h" class="fill-rose-500"><title>{{ uz.stats.absent }}: {{ s.d.absent }}</title></rect>
        <text :x="s.x + s.w / 2" :y="H - 10" text-anchor="middle" class="fill-current text-[10px] opacity-70">{{ s.label }}</text>
      </g>
    </svg>
    <ul class="flex flex-wrap gap-4 pt-2 text-xs">
      <li class="flex items-center gap-1.5"><span class="inline-block size-2.5 rounded-sm bg-emerald-500" />{{ uz.stats.full }}</li>
      <li class="flex items-center gap-1.5"><span class="inline-block size-2.5 rounded-sm bg-amber-400" />{{ uz.stats.partial }}</li>
      <li class="flex items-center gap-1.5"><span class="inline-block size-2.5 rounded-sm bg-rose-500" />{{ uz.stats.absent }}</li>
    </ul>
  </div>
</template>
