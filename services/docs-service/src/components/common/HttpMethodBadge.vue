<script setup lang="ts">
import type { HttpMethod } from '@/types'
import { httpMethodBadgeStyle } from '@/constants'

const props = defineProps<{ method: HttpMethod }>()
</script>

<template>
  <span class="badge" :style="httpMethodBadgeStyle(props.method)">
    {{ props.method }}
  </span>
</template>

<style scoped>
.badge {
  display: inline-flex;
  align-items: center;
  border-radius: var(--radius-sm);
  border: 1px solid;
  padding: 2px 6px;
  font-size: 11px;
  font-weight: 700;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  line-height: 1.4;
}
</style>
