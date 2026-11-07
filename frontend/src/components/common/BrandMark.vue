<script setup lang="ts">
withDefaults(
  defineProps<{
    compact?: boolean
    inverse?: boolean
  }>(),
  {
    compact: false,
    inverse: false,
  },
)
</script>

<template>
  <div class="brand" :class="{ compact, inverse }">
    <div class="brand-glyph" aria-hidden="true">拾</div>
    <div v-if="!compact" class="brand-copy">
      <strong>拾刊</strong>
      <span>AGENT CONSOLE</span>
    </div>
  </div>
</template>

<style scoped>
.brand {
  display: flex;
  align-items: center;
  gap: 11px;
  color: var(--ink);
}

.brand-glyph {
  display: grid;
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  place-items: center;
  border: 1px solid currentColor;
  border-radius: 6px;
  font-family: "Microsoft YaHei UI", sans-serif;
  font-size: 18px;
  font-weight: 700;
  line-height: 1;
}

.brand-copy {
  display: flex;
  flex-direction: column;
  gap: 1px;
  white-space: nowrap;
}

.brand-copy strong {
  font-family: "Microsoft YaHei UI", sans-serif;
  font-size: 17px;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.brand-copy span {
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 8px;
  font-weight: 700;
  letter-spacing: 0.2em;
  opacity: 0.54;
}

.brand.inverse {
  color: #f7f5ec;
}

.brand.compact .brand-glyph {
  width: 36px;
  height: 36px;
}
</style>
