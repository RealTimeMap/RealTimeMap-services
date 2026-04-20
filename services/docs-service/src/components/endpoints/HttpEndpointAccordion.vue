<script setup lang="ts">
import { ref, computed } from 'vue'
import type { HttpEndpoint, SchemaField } from '@/types'
import { useSharedStore } from '@/stores/shared'
import HttpMethodBadge from '@/components/common/HttpMethodBadge.vue'
import ParameterTable from '@/components/common/ParameterTable.vue'
import SchemaViewer from '@/components/common/SchemaViewer.vue'
import CodeBlock from '@/components/common/CodeBlock.vue'
import HttpRequestTester from '@/components/testing/HttpRequestTester.vue'

const props = defineProps<{ endpoint: HttpEndpoint; serviceId: string }>()

const isOpen = ref(false)
const activeTab = ref<'overview' | 'schemas' | 'testing'>('overview')
const sharedStore = useSharedStore()

const allParameters = computed(() => {
  const params = [...(props.endpoint.parameters ?? [])]
  if (props.endpoint.pagination) {
    const pag = sharedStore.getPagination(props.endpoint.pagination)
    if (pag) {
      for (const qp of pag.queryParams) {
        if (!params.some(p => p.name === qp.name)) params.push(qp)
      }
    }
  }
  return params
})

const allResponses = computed(() => {
  const responses = [...props.endpoint.responses]
  if (props.endpoint.errors) {
    for (const errorId of props.endpoint.errors) {
      const err = sharedStore.getError(errorId)
      if (err && !responses.some(r => r.statusCode === err.statusCode)) {
        responses.push({ statusCode: err.statusCode, description: err.description, schema: err.schema, example: err.example })
      }
    }
  }
  responses.sort((a, b) => a.statusCode - b.statusCode)
  return responses
})

const paginationInfo = computed(() => {
  if (!props.endpoint.pagination) return null
  return sharedStore.getPagination(props.endpoint.pagination) ?? null
})

const activeResponseTab = ref(0)
const activeResponseItem = computed(() => allResponses.value[activeResponseTab.value])

function statusColor(code: number): string {
  if (code >= 200 && code < 300) return 'var(--color-success)'
  if (code >= 400 && code < 500) return 'var(--color-warning)'
  return 'var(--color-error)'
}

// Схемы для вкладки "Схемы"
const responseSchemas = computed(() => {
  const result: { statusCode: number; description: string; schemaName?: string; schemaDesc?: string; fields: SchemaField[] }[] = []
  for (const resp of allResponses.value) {
    let fields: SchemaField[] | undefined
    let schemaName: string | undefined
    let schemaDesc: string | undefined
    if (resp.schema?.length) {
      fields = resp.schema
    } else if (resp.schemaRef) {
      fields = sharedStore.resolveSchemaRef(resp.schemaRef)
      const s = sharedStore.getSchema(resp.schemaRef)
      if (s) {
        schemaName = s.name
        schemaDesc = s.description
      }
    }
    if (fields?.length) {
      result.push({ statusCode: resp.statusCode, description: resp.description, schemaName, schemaDesc, fields })
    }
  }
  return result
})
</script>

<template>
  <div :id="'endpoint-' + endpoint.id" class="accordion" :class="{ open: isOpen }">
    <button class="accordion-header" @click="isOpen = !isOpen">
      <HttpMethodBadge :method="endpoint.method" />
      <code class="endpoint-path">{{ endpoint.path }}</code>
      <span class="endpoint-summary truncate">{{ endpoint.summary }}</span>
      <span v-if="endpoint.pagination" class="pagination-badge">
        {{ paginationInfo?.name ?? endpoint.pagination }}
      </span>
      <svg class="chevron" :class="{ rotated: isOpen }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M6 9l6 6 6-6" />
      </svg>
    </button>

    <div v-if="isOpen" class="accordion-body">
      <p v-if="endpoint.description" class="endpoint-desc">{{ endpoint.description }}</p>

      <!-- Вкладки -->
      <div class="tab-bar">
        <button
          class="tab-btn"
          :class="{ active: activeTab === 'overview' }"
          @click="activeTab = 'overview'"
        >
          Обзор
        </button>
        <button
          class="tab-btn"
          :class="{ active: activeTab === 'schemas' }"
          @click="activeTab = 'schemas'"
        >
          Схемы
        </button>
        <button
          class="tab-btn"
          :class="{ active: activeTab === 'testing' }"
          @click="activeTab = 'testing'"
        >
          Тестирование
        </button>
      </div>

      <!-- Вкладка: Обзор -->
      <div v-if="activeTab === 'overview'" class="tab-content">
        <section v-if="allParameters.length">
          <h4 class="section-title">Параметры</h4>
          <ParameterTable :parameters="allParameters" />
        </section>

        <section v-if="endpoint.requestBody">
          <h4 class="section-title">Тело запроса</h4>
          <p v-if="endpoint.requestBody.description" class="body-desc">{{ endpoint.requestBody.description }}</p>
          <div v-if="endpoint.requestBody.example" class="example-block">
            <CodeBlock :code="JSON.stringify(endpoint.requestBody.example, null, 2)" language="json" />
          </div>
        </section>

        <section v-if="allResponses.length">
          <h4 class="section-title">Ответы</h4>
          <div class="response-tabs">
            <button
              v-for="(resp, i) in allResponses"
              :key="resp.statusCode"
              class="response-tab"
              :class="{ active: activeResponseTab === i }"
              :style="{
                color: statusColor(resp.statusCode),
                backgroundColor: activeResponseTab === i
                  ? `color-mix(in srgb, ${statusColor(resp.statusCode)} 10%, transparent)`
                  : 'transparent',
                borderColor: activeResponseTab === i
                  ? `color-mix(in srgb, ${statusColor(resp.statusCode)} 30%, transparent)`
                  : 'transparent',
              }"
              @click="activeResponseTab = i"
            >
              {{ resp.statusCode }}
            </button>
          </div>

          <div v-if="activeResponseItem" class="response-content">
            <p class="resp-desc">{{ activeResponseItem.description }}</p>
            <div v-if="activeResponseItem.example">
              <h5 class="section-title">Пример ответа</h5>
              <CodeBlock :code="JSON.stringify(activeResponseItem.example, null, 2)" language="json" />
            </div>
          </div>
        </section>
      </div>

      <!-- Вкладка: Схемы -->
      <div v-if="activeTab === 'schemas'" class="tab-content">
        <section v-if="paginationInfo" class="pagination-section">
          <div class="pagination-header">
            <span class="pagination-badge">{{ paginationInfo.name }}</span>
            <span class="pagination-desc">{{ paginationInfo.description }}</span>
          </div>
          <h5 class="section-title">Обёртка ответа</h5>
          <SchemaViewer :fields="paginationInfo.wrapperSchema" />
        </section>

        <section v-if="endpoint.requestBody">
          <h4 class="section-title">Схема тела запроса</h4>
          <SchemaViewer :fields="endpoint.requestBody.schema" />
        </section>

        <section v-if="responseSchemas.length">
          <h4 class="section-title">Схемы ответов</h4>
          <div v-for="rs in responseSchemas" :key="rs.statusCode" class="schema-block">
            <div class="schema-block-header">
              <span
                class="status-code-badge"
                :style="{ color: statusColor(rs.statusCode), borderColor: `color-mix(in srgb, ${statusColor(rs.statusCode)} 30%, transparent)` }"
              >
                {{ rs.statusCode }}
              </span>
              <span class="schema-block-desc">{{ rs.description }}</span>
              <span v-if="rs.schemaName" class="schema-ref-badge" :title="rs.schemaDesc">
                {{ rs.schemaName }}
              </span>
            </div>
            <SchemaViewer :fields="rs.fields" />
          </div>
        </section>

        <div v-if="!paginationInfo && !endpoint.requestBody && !responseSchemas.length" class="empty-state">
          Схемы не определены для этого endpoint
        </div>
      </div>

      <!-- Вкладка: Тестирование -->
      <div v-if="activeTab === 'testing'" class="tab-content">
        <HttpRequestTester :endpoint="endpoint" :service-id="serviceId" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.accordion {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  overflow: hidden;
  transition: background-color 0.15s;
}
.accordion:not(.open):hover {
  background: color-mix(in srgb, var(--color-bg-secondary) 50%, transparent);
}
.accordion.open {
  background: var(--color-bg-secondary);
}
.accordion-header {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  text-align: left;
  cursor: pointer;
  background: none;
  border: none;
  color: inherit;
  font: inherit;
}
.endpoint-path {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text);
}
.endpoint-summary {
  font-size: 13px;
  color: var(--color-text-secondary);
  flex: 1;
}
.pagination-badge {
  border-radius: var(--radius-full);
  background: color-mix(in srgb, #6366f1 10%, transparent);
  color: #6366f1;
  border: 1px solid color-mix(in srgb, #6366f1 20%, transparent);
  padding: 2px 8px;
  font-size: 10px;
  font-weight: 500;
  flex-shrink: 0;
}
.chevron {
  width: 16px;
  height: 16px;
  color: var(--color-text-muted);
  flex-shrink: 0;
  transition: transform 0.2s;
}
.chevron.rotated {
  transform: rotate(180deg);
}
.accordion-body {
  padding: 0 16px 16px;
  border-top: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.endpoint-desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  padding-top: 16px;
  margin: 0;
}

/* Панель вкладок */
.tab-bar {
  display: flex;
  gap: 0;
  border-bottom: 1px solid var(--color-border);
  margin-top: 12px;
}
.tab-btn {
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-muted);
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}
.tab-btn:hover {
  color: var(--color-text-secondary);
}
.tab-btn.active {
  color: var(--color-accent);
  border-bottom-color: var(--color-accent);
}

/* Содержимое вкладок */
.tab-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.section-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  margin: 0 0 8px;
}
.body-desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin: 0 0 8px;
}
.example-block {
  margin-top: 4px;
}

/* Табы ответов (статус-коды) */
.response-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 12px;
}
.response-tab {
  padding: 4px 12px;
  font-size: 12px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  font-weight: 700;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  background: none;
  opacity: 0.6;
  transition: opacity 0.15s;
}
.response-tab:hover,
.response-tab.active {
  opacity: 1;
}
.response-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.resp-desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin: 0;
}

/* Вкладка Схемы */
.pagination-section {
  border-radius: var(--radius-md);
  border: 1px solid color-mix(in srgb, #6366f1 20%, transparent);
  background: color-mix(in srgb, #6366f1 5%, transparent);
  padding: 12px;
}
.pagination-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.pagination-desc {
  font-size: 12px;
  color: var(--color-text-muted);
}
.schema-block {
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  padding: 12px;
}
.schema-block + .schema-block {
  margin-top: 12px;
}
.schema-block-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.status-code-badge {
  font-size: 12px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  font-weight: 700;
  border: 1px solid;
  border-radius: var(--radius-sm);
  padding: 2px 8px;
}
.schema-block-desc {
  font-size: 12px;
  color: var(--color-text-secondary);
}
.schema-ref-badge {
  border-radius: var(--radius-full);
  background: color-mix(in srgb, var(--color-info) 10%, transparent);
  color: var(--color-info);
  border: 1px solid color-mix(in srgb, var(--color-info) 20%, transparent);
  padding: 1px 8px;
  font-size: 10px;
  font-weight: 500;
  margin-left: auto;
}
.empty-state {
  padding: 32px 0;
  text-align: center;
  color: var(--color-text-muted);
  font-size: 13px;
}
</style>
