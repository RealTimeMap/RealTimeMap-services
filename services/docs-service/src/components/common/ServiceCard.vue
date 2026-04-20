<script setup lang="ts">
import type { ServiceIndex } from '@/types'
import { useHealthStore } from '@/stores/health'
import ProtocolBadge from './ProtocolBadge.vue'
import HealthIndicator from './HealthIndicator.vue'

const props = defineProps<{ service: ServiceIndex }>()
const healthStore = useHealthStore()
</script>

<template>
  <router-link :to="{ name: 'service', params: { serviceId: props.service.id } }" class="card">
    <div class="card-header">
      <h3 class="card-title">{{ props.service.name }}</h3>
      <HealthIndicator :status="healthStore.statuses[props.service.id]" />
    </div>
    <p class="card-desc line-clamp-2">{{ props.service.description }}</p>
    <div class="card-protocols">
      <ProtocolBadge v-for="protocol in props.service.protocols" :key="protocol" :protocol="protocol" />
    </div>
  </router-link>
</template>

<style scoped>
.card {
  display: block;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  padding: 20px;
  text-decoration: none;
  color: inherit;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.card:hover {
  border-color: var(--color-border-hover);
  box-shadow: var(--shadow-md);
  text-decoration: none;
}
.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}
.card-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}
.card-desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
}
.card-protocols {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
