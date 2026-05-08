<script setup lang="ts">
import type { SocketIODirection } from '@/types'

const props = defineProps<{ direction: SocketIODirection; compact?: boolean }>()

const isClientToServer = props.direction === 'client-to-server'
</script>

<template>
  <span class="badge" :class="{ 'c-to-s': isClientToServer, 's-to-c': !isClientToServer, compact }">
    <template v-if="isClientToServer">
      <span class="role">Client</span>
      <svg class="arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M5 12h14M13 5l7 7-7 7" />
      </svg>
      <span class="role">Server</span>
    </template>
    <template v-else>
      <span class="role">Server</span>
      <svg class="arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M5 12h14M13 5l7 7-7 7" />
      </svg>
      <span class="role">Client</span>
    </template>
  </span>
</template>

<style scoped>
.badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border-radius: var(--radius-sm);
  border: 1px solid;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 600;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  line-height: 1.4;
  white-space: nowrap;
}
.badge.compact {
  padding: 1px 6px;
  font-size: 10px;
}
.badge.c-to-s {
  color: var(--color-socketio);
  background: color-mix(in srgb, var(--color-socketio) 10%, transparent);
  border-color: color-mix(in srgb, var(--color-socketio) 25%, transparent);
}
.badge.s-to-c {
  color: var(--color-info, #3b82f6);
  background: color-mix(in srgb, var(--color-info, #3b82f6) 10%, transparent);
  border-color: color-mix(in srgb, var(--color-info, #3b82f6) 25%, transparent);
}
.arrow {
  width: 12px;
  height: 12px;
}
.role {
  letter-spacing: 0.02em;
}
</style>
