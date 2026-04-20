<script setup lang="ts">
import { ref, computed } from 'vue'
import type { HttpResponse, SchemaField } from '@/types'
import { useSharedStore } from '@/stores/shared'
import SchemaViewer from './SchemaViewer.vue'
import CodeBlock from './CodeBlock.vue'

const props = defineProps<{ responses: HttpResponse[] }>()

const sharedStore = useSharedStore()

const activeTab = ref(0)
const activeResponse = computed(() => props.responses[activeTab.value])

const activeSchema = computed<SchemaField[] | undefined>(() => {
  const resp = activeResponse.value
  if (!resp) return undefined
  if (resp.schema?.length) return resp.schema
  if (resp.schemaRef) return sharedStore.resolveSchemaRef(resp.schemaRef)
  return undefined
})

const activeSchemaName = computed(() => {
  const resp = activeResponse.value
  if (!resp?.schemaRef) return null
  return sharedStore.getSchema(resp.schemaRef)
})

function statusColor(code: number): string {
  if (code >= 200 && code < 300) return 'var(--color-success)'
  if (code >= 400 && code < 500) return 'var(--color-warning)'
  return 'var(--color-error)'
}
</script>

<template>
  <div>
    <div class="tabs">
      <button
        v-for="(resp, i) in responses"
        :key="resp.statusCode"
        class="tab"
        :class="{ active: activeTab === i }"
        :style="{
          color: statusColor(resp.statusCode),
          backgroundColor: activeTab === i
            ? `color-mix(in srgb, ${statusColor(resp.statusCode)} 10%, transparent)`
            : 'transparent',
          borderColor: activeTab === i
            ? `color-mix(in srgb, ${statusColor(resp.statusCode)} 20%, transparent)`
            : 'transparent',
        }"
        @click="activeTab = i"
      >
        {{ resp.statusCode }}
      </button>
    </div>

    <div v-if="activeResponse" class="tab-content">
      <p class="resp-desc">{{ activeResponse.description }}</p>

      <div v-if="activeSchema?.length" class="resp-section">
        <div class="section-header">
          <h5 class="section-title">Схема</h5>
          <span v-if="activeSchemaName" class="schema-ref-badge" :title="activeSchemaName.description">
            {{ activeSchemaName.name }}
          </span>
        </div>
        <SchemaViewer :fields="activeSchema" />
      </div>

      <div v-if="activeResponse.example" class="resp-section">
        <h5 class="section-title">Пример</h5>
        <CodeBlock :code="JSON.stringify(activeResponse.example, null, 2)" language="json" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 12px;
  border-bottom: 1px solid var(--color-border);
}
.tab {
  padding: 6px 12px;
  font-size: 12px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  font-weight: 700;
  border: 1px solid transparent;
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
  cursor: pointer;
  background: none;
  opacity: 0.6;
  transition: opacity 0.15s;
}
.tab:hover,
.tab.active {
  opacity: 1;
}
.tab.active {
  border-bottom-color: transparent;
  margin-bottom: -1px;
}
.resp-desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
}
.resp-section {
  margin-bottom: 12px;
}
.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.section-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  margin: 0;
}
.schema-ref-badge {
  border-radius: var(--radius-full);
  background: color-mix(in srgb, var(--color-info) 10%, transparent);
  color: var(--color-info);
  border: 1px solid color-mix(in srgb, var(--color-info) 20%, transparent);
  padding: 1px 8px;
  font-size: 10px;
  font-weight: 500;
}
</style>
