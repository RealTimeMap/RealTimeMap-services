<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { useServicesStore } from '@/stores/services'
import { useHealthStore } from '@/stores/health'
import { HEALTH_CHECK_INTERVAL, PROTOCOL_LABELS } from '@/constants'
import type { Protocol, HealthState } from '@/types'
import ServiceCard from '@/components/common/ServiceCard.vue'

const servicesStore = useServicesStore()
const healthStore = useHealthStore()

const search = ref('')
const protocolFilter = ref<Protocol | null>(null)
const statusFilter = ref<HealthState | 'all'>('all')

const protocols: Protocol[] = ['http', 'kafka', 'grpc', 'socketio']

const filteredServices = computed(() => {
  let result = servicesStore.services
  if (search.value) {
    const q = search.value.toLowerCase()
    result = result.filter(s => s.name.toLowerCase().includes(q) || s.description.toLowerCase().includes(q))
  }
  if (protocolFilter.value) {
    result = result.filter(s => s.protocols.includes(protocolFilter.value!))
  }
  if (statusFilter.value !== 'all') {
    result = result.filter(s => {
      const state = healthStore.statuses[s.id]?.state ?? 'unknown'
      return state === statusFilter.value
    })
  }
  return result
})

onMounted(async () => {
  await servicesStore.load()
  healthStore.checkAll()
})

useIntervalFn(() => healthStore.checkAll(), HEALTH_CHECK_INTERVAL)
</script>

<template>
  <div class="page">
    <div class="filters">
      <input v-model="search" type="text" placeholder="Поиск по сервисам..." class="search-input" />
      <div class="filter-row">
        <button
          v-for="proto in protocols"
          :key="proto"
          class="filter-chip"
          :class="{ active: protocolFilter === proto }"
          @click="protocolFilter = protocolFilter === proto ? null : proto"
        >
          {{ PROTOCOL_LABELS[proto] }}
        </button>
        <span class="filter-divider">|</span>
        <button
          v-for="opt in [
            { value: 'all' as const, label: 'Все' },
            { value: 'healthy' as const, label: 'Работающие' },
            { value: 'unhealthy' as const, label: 'Упавшие' },
          ]"
          :key="opt.value"
          class="filter-chip"
          :class="{ active: statusFilter === opt.value }"
          @click="statusFilter = opt.value"
        >
          {{ opt.label }}
        </button>
      </div>
    </div>

    <div v-if="servicesStore.loading" class="center-state">
      <div class="spinner" />
    </div>

    <div v-else-if="servicesStore.error" class="error-box">
      <p>{{ servicesStore.error }}</p>
    </div>

    <div v-else-if="filteredServices.length === 0" class="center-state">
      <p class="empty-text">Сервисы не найдены</p>
    </div>

    <div v-else class="services-grid">
      <ServiceCard v-for="service in filteredServices" :key="service.id" :service="service" />
    </div>
  </div>
</template>

<style scoped>
.page {
  max-width: 1280px;
  margin: 0 auto;
  padding: 24px 16px;
}
.filters {
  margin-bottom: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.search-input {
  width: 100%;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  padding: 10px 16px;
  font-size: 13px;
  color: var(--color-text);
  outline: none;
}
.search-input::placeholder {
  color: var(--color-text-muted);
}
.search-input:focus {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 40%, transparent);
}
.filter-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.filter-chip {
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border);
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: none;
  cursor: pointer;
  transition: all 0.15s;
}
.filter-chip:hover {
  border-color: var(--color-border-hover);
}
.filter-chip.active {
  border-color: var(--color-accent);
  background: color-mix(in srgb, var(--color-accent) 10%, transparent);
  color: var(--color-accent);
}
.filter-divider {
  color: var(--color-border);
  margin: 0 8px;
}
.center-state {
  display: flex;
  justify-content: center;
  padding: 80px 0;
}
.error-box {
  border-radius: var(--radius-md);
  border: 1px solid color-mix(in srgb, var(--color-error) 20%, transparent);
  background: color-mix(in srgb, var(--color-error) 5%, transparent);
  padding: 24px;
  text-align: center;
  color: var(--color-error);
  font-size: 13px;
}
.empty-text {
  color: var(--color-text-muted);
  font-size: 13px;
}
.services-grid {
  display: grid;
  grid-template-columns: repeat(1, 1fr);
  gap: 16px;
}
@media (min-width: 640px) {
  .services-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (min-width: 1024px) {
  .services-grid { grid-template-columns: repeat(3, 1fr); }
}
@media (min-width: 1280px) {
  .services-grid { grid-template-columns: repeat(4, 1fr); }
}
</style>
