<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    name: string
    size?: number
  }>(),
  { size: 34 },
)

const colors = [
  ['#dff58f', '#263418'],
  ['#ffd7b5', '#552e18'],
  ['#b9ddf5', '#18384e'],
  ['#efbec8', '#512631'],
  ['#b9e4d0', '#173f30'],
  ['#e1d3fa', '#35254f'],
]

const palette = computed(() => {
  const seed = Array.from(props.name).reduce((sum, letter) => sum + letter.charCodeAt(0), 0)
  return colors[seed % colors.length]
})

const initials = computed(() => props.name.slice(0, 1))
</script>

<template>
  <span
    class="agent-avatar"
    :style="{
      width: `${size}px`,
      height: `${size}px`,
      background: palette[0],
      color: palette[1],
      fontSize: `${Math.max(11, size * 0.35)}px`,
    }"
  >
    {{ initials }}
  </span>
</template>

<style scoped>
.agent-avatar {
  display: inline-grid;
  flex: 0 0 auto;
  place-items: center;
  border: 1px solid rgba(21, 34, 27, 0.1);
  border-radius: 6px;
  font-weight: 750;
  line-height: 1;
}
</style>
