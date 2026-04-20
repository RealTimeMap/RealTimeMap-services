<script setup lang="ts">
import type { HealthStatus } from '@/types'
import { HEALTH_LABELS, HEALTH_DOT_COLORS } from '@/constants'
import { computed } from 'vue'

const props = defineProps<{ status?: HealthStatus }>()

const state = computed(() => props.status?.state ?? 'unknown')

const tooltipText = computed(() => {
  if (!props.status) return 'Статус неизвестен'
  const parts: string[] = [HEALTH_LABELS[state.value]]
  if (props.status.latency != null) parts.push(`Latency: ${props.status.latency}ms`)
  if (props.status.httpStatus != null) parts.push(`HTTP ${props.status.httpStatus}`)
  if (props.status.timestamp) {
    parts.push(`Проверено: ${new Date(props.status.timestamp).toLocaleTimeString('ru-RU')}`)
  }
  return parts.join(' · ')
})
</script>

<template>
  <span class="health" :title="tooltipText">
    <span
      class="dot"
      :class="{ pulse: state === 'checking' }"
      :style="{ backgroundColor: HEALTH_DOT_COLORS[state] }"
    />
    <span class="label">{{ HEALTH_LABELS[state] }}</span>
  </span>
</template>

<style scoped>
.health {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.label {
  font-size: 12px;
  color: var(--color-text-secondary);
}
</style>
