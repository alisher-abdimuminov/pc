<script setup lang="ts">
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps<{ polygon: [number, number][]; toleranceMeters?: number }>()
const el = ref<HTMLDivElement>()
let map: L.Map | null = null
let layer: L.LayerGroup | null = null

function draw() {
  if (!map || !layer) return
  layer.clearLayers()
  const pts = props.polygon
  pts.forEach((p) => L.circleMarker(p, { radius: 4, color: '#2563eb' }).addTo(layer!))
  if (pts.length >= 3) {
    L.polygon(pts, { color: '#2563eb', weight: 2 }).addTo(layer)
  } else if (pts.length === 2) {
    L.polyline(pts, { color: '#2563eb' }).addTo(layer)
  }
  if (pts.length) map.fitBounds(L.latLngBounds(pts).pad(0.3), { maxZoom: 18 })
}

onMounted(() => {
  map = L.map(el.value!, { scrollWheelZoom: false }).setView([41.3, 69.24], 5)
  L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; OpenStreetMap',
  }).addTo(map)
  layer = L.layerGroup().addTo(map)
  draw()
})
watch(() => props.polygon, draw, { deep: true })
onBeforeUnmount(() => map?.remove())
</script>

<template>
  <div ref="el" class="h-64 w-full overflow-hidden rounded-lg border" role="img" aria-label="Xarita" />
</template>
