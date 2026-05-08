<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import type { Protocol, HttpDocs, SocketIODocs, KafkaDocs, GrpcDocs } from '@/types'
import { fetchProtocolDocs } from '@/api/docs'
import { PROTOCOL_LABELS } from '@/constants'
import HttpDocsView from '@/components/endpoints/HttpDocsView.vue'
import SocketIODocsView from '@/components/endpoints/SocketIODocsView.vue'

const props = defineProps<{ serviceId: string; protocol: string }>()

const docs = ref<HttpDocs | SocketIODocs | KafkaDocs | GrpcDocs | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const protocolKey = computed(() => props.protocol as Protocol)

onMounted(async () => {
  try {
    docs.value = await fetchProtocolDocs(props.serviceId, protocolKey.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки документации'
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
      <router-link :to="{ name: 'service', params: { serviceId } }" class="breadcrumb-link">{{ serviceId }}</router-link>
      <span class="breadcrumb-sep">/</span>
      <span class="breadcrumb-current">{{ PROTOCOL_LABELS[protocolKey] ?? protocol }}</span>
    </nav>

    <h1 class="page-title">{{ serviceId }} — {{ PROTOCOL_LABELS[protocolKey] }}</h1>

    <div v-if="loading" class="center-state">
      <div class="spinner" />
    </div>

    <div v-else-if="error" class="error-box">
      <p>{{ error }}</p>
    </div>

    <template v-else-if="docs">
      <HttpDocsView v-if="protocolKey === 'http'" :docs="(docs as HttpDocs)" :service-id="serviceId" />

      <SocketIODocsView v-else-if="protocolKey === 'socketio'" :docs="(docs as SocketIODocs)" :service-id="serviceId" />

      <div v-else class="fallback">
        <pre class="fallback-pre">{{ JSON.stringify(docs, null, 2) }}</pre>
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
.page-title {
  font-size: 24px;
  font-weight: 700;
  margin: 0 0 24px;
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
.fallback-pre {
  border-radius: var(--radius-md);
  background: var(--color-bg-tertiary);
  padding: 16px;
  overflow-x: auto;
  font-size: 12px;
}
</style>
