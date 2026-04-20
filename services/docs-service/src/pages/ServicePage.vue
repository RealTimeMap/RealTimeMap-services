<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { ServiceMeta } from '@/types'
import { fetchServiceMeta } from '@/api/docs'
import { useHealthStore } from '@/stores/health'
import { PROTOCOL_LABELS, protocolBadgeStyle } from '@/constants'
import HealthIndicator from '@/components/common/HealthIndicator.vue'

const props = defineProps<{ serviceId: string }>()

const service = ref<ServiceMeta | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const healthStore = useHealthStore()

onMounted(async () => {
  try {
    service.value = await fetchServiceMeta(props.serviceId)
    healthStore.checkService(props.serviceId)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="page">
    <nav class="breadcrumbs">
      <router-link to="/" class="breadcrumb-link">Главная</router-link>
      <span class="breadcrumb-sep">/</span>
      <span class="breadcrumb-current">{{ serviceId }}</span>
    </nav>

    <div v-if="loading" class="center-state">
      <div class="spinner" />
    </div>

    <div v-else-if="error" class="error-box">
      <p>{{ error }}</p>
    </div>

    <template v-else-if="service">
      <div class="service-header">
        <h1 class="service-title">{{ service.name }}</h1>
        <HealthIndicator :status="healthStore.statuses[serviceId]" />
      </div>

      <p class="service-desc">{{ service.description }}</p>

      <div v-if="service.team || service.repository" class="service-meta">
        <span v-if="service.team">Команда: <strong>{{ service.team }}</strong></span>
        <a v-if="service.repository" :href="service.repository" target="_blank" rel="noopener">Репозиторий</a>
      </div>

      <h2 class="protocols-title">Документация по протоколам</h2>
      <div class="protocols-grid">
        <router-link
          v-for="protocol in service.protocols"
          :key="protocol"
          :to="{ name: 'protocol', params: { serviceId, protocol } }"
          class="protocol-card"
        >
          <span class="protocol-badge" :style="protocolBadgeStyle(protocol)">
            {{ PROTOCOL_LABELS[protocol] }}
          </span>
        </router-link>
      </div>
    </template>
  </div>
</template>

<style scoped>
.page {
  max-width: 1280px;
  margin: 0 auto;
  padding: 24px 16px;
}
.breadcrumbs {
  margin-bottom: 16px;
  font-size: 13px;
  color: var(--color-text-muted);
}
.breadcrumb-link {
  color: var(--color-text-muted);
  text-decoration: none;
}
.breadcrumb-link:hover {
  color: var(--color-text-secondary);
  text-decoration: none;
}
.breadcrumb-sep {
  margin: 0 6px;
}
.breadcrumb-current {
  color: var(--color-text);
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
.service-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}
.service-title {
  font-size: 24px;
  font-weight: 700;
  margin: 0;
}
.service-desc {
  color: var(--color-text-secondary);
  margin-bottom: 24px;
}
.service-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 32px;
  font-size: 13px;
  color: var(--color-text-secondary);
}
.service-meta strong {
  color: var(--color-text);
}
.protocols-title {
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 16px;
}
.protocols-grid {
  display: grid;
  grid-template-columns: repeat(1, 1fr);
  gap: 16px;
}
@media (min-width: 640px) {
  .protocols-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (min-width: 1024px) {
  .protocols-grid { grid-template-columns: repeat(4, 1fr); }
}
.protocol-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  padding: 24px;
  text-decoration: none;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.protocol-card:hover {
  border-color: var(--color-border-hover);
  box-shadow: var(--shadow-md);
  text-decoration: none;
}
.protocol-badge {
  border-radius: var(--radius-full);
  border: 1px solid;
  padding: 6px 16px;
  font-size: 14px;
  font-weight: 600;
}
</style>
