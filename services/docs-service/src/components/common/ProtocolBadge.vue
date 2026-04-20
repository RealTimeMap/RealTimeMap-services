<script setup lang="ts">
import type { Protocol } from '@/types'
import { PROTOCOL_LABELS, protocolBadgeStyle } from '@/constants'

const props = defineProps<{ protocol: Protocol }>()
</script>

<template>
  <span class="badge" :style="protocolBadgeStyle(props.protocol)">
    {{ PROTOCOL_LABELS[props.protocol] }}
  </span>
</template>

<style scoped>
.badge {
  display: inline-flex;
  align-items: center;
  border-radius: var(--radius-full);
  border: 1px solid;
  padding: 2px 10px;
  font-size: 12px;
  font-weight: 500;
  line-height: 1.4;
}
</style>
