<script setup lang="ts">
import { ref, computed } from 'vue'
import type { HttpEndpoint, Parameter } from '@/types'
import { useEnvironmentStore } from '@/stores/environment'
import { useAuthStore } from '@/stores/auth'
import { useSharedStore } from '@/stores/shared'
import { getServiceUrl } from '@/utils/env'
import { ENVIRONMENT_LABELS } from '@/constants'
import CodeBlock from '@/components/common/CodeBlock.vue'

const props = defineProps<{ endpoint: HttpEndpoint; serviceId: string }>()

const envStore = useEnvironmentStore()
const authStore = useAuthStore()
const sharedStore = useSharedStore()

const baseUrl = computed(() => getServiceUrl(props.serviceId, envStore.current))

const allParams = computed<Parameter[]>(() => {
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

const pathParams = ref<Record<string, string>>({})
const queryParams = ref<Record<string, string>>({})
const headerParams = ref<Record<string, string>>({})
const bodyText = ref(props.endpoint.requestBody?.example ? JSON.stringify(props.endpoint.requestBody.example, null, 2) : '')
const extraHeaders = ref('')

const sending = ref(false)
const responseStatus = ref<number | null>(null)
const responseBody = ref<string | null>(null)
const responseHeaders = ref<string | null>(null)
const responseTime = ref<number | null>(null)
const responseError = ref<string | null>(null)

function initParams(params: Parameter[]) {
  for (const p of params) {
    const target = p.location === 'path' ? pathParams : p.location === 'query' ? queryParams : headerParams
    if (!(p.name in target.value)) target.value[p.name] = p.example ?? ''
  }
}
initParams(allParams.value)

function buildUrl(): string {
  let path = props.endpoint.path
  if (!path.startsWith('/')) path = '/' + path
  for (const [key, val] of Object.entries(pathParams.value)) {
    path = path.replace(`:${key}`, encodeURIComponent(val))
    path = path.replace(`{${key}}`, encodeURIComponent(val))
  }
  const base = baseUrl.value.replace(/\/$/, '')
  const qp = new URLSearchParams()
  for (const [key, val] of Object.entries(queryParams.value)) {
    if (val) qp.set(key, val)
  }
  const qs = qp.toString()
  return `${base}${path}${qs ? '?' + qs : ''}`
}

function buildHeaders(): Record<string, string> {
  const h: Record<string, string> = {}
  for (const [key, val] of Object.entries(headerParams.value)) {
    if (val) h[key] = val
  }
  const token = authStore.getToken(envStore.current)
  if (token && !h['Authorization']) h['Authorization'] = `Bearer ${token}`
  if (props.endpoint.requestBody && props.endpoint.method !== 'GET') h['Content-Type'] = 'application/json'
  if (extraHeaders.value.trim()) {
    for (const line of extraHeaders.value.split('\n')) {
      const idx = line.indexOf(':')
      if (idx > 0) h[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
    }
  }
  return h
}

function toCurl(): string {
  const url = buildUrl()
  const headers = buildHeaders()
  let cmd = `curl -X ${props.endpoint.method} '${url}'`
  for (const [k, v] of Object.entries(headers)) cmd += ` \\\n  -H '${k}: ${v}'`
  if (bodyText.value && props.endpoint.method !== 'GET') cmd += ` \\\n  -d '${bodyText.value.replace(/\n/g, '')}'`
  return cmd
}

async function send() {
  sending.value = true
  responseStatus.value = null
  responseBody.value = null
  responseHeaders.value = null
  responseTime.value = null
  responseError.value = null

  const url = buildUrl()
  const headers = buildHeaders()
  const start = performance.now()

  try {
    const res = await fetch(url, {
      method: props.endpoint.method,
      headers,
      body: props.endpoint.method !== 'GET' && bodyText.value ? bodyText.value : undefined,
      signal: AbortSignal.timeout(15000),
    })
    responseTime.value = Math.round(performance.now() - start)
    responseStatus.value = res.status
    const headersArr: string[] = []
    res.headers.forEach((v, k) => headersArr.push(`${k}: ${v}`))
    responseHeaders.value = headersArr.join('\n')
    const text = await res.text()
    try { responseBody.value = JSON.stringify(JSON.parse(text), null, 2) }
    catch { responseBody.value = text }
  } catch (e) {
    responseTime.value = Math.round(performance.now() - start)
    responseError.value = e instanceof Error ? e.message : 'Ошибка запроса'
  } finally {
    sending.value = false
  }
}

const showCurl = ref(false)
</script>

<template>
  <div class="tester">
    <div class="env-indicator">
      <span class="env-dot" />
      {{ ENVIRONMENT_LABELS[envStore.current] ?? envStore.current }}
      <span class="env-arrow">→</span>
      <code class="env-url">{{ buildUrl() }}</code>
    </div>

    <!-- Path параметры -->
    <div v-if="allParams.filter(p => p.location === 'path').length" class="param-group">
      <label class="param-label">Path параметры</label>
      <div v-for="p in allParams.filter(p => p.location === 'path')" :key="p.name" class="param-row">
        <code class="param-name">{{ p.name }}</code>
        <input v-model="pathParams[p.name]" class="param-input" :placeholder="p.example ?? p.type" />
      </div>
    </div>

    <!-- Query параметры -->
    <div v-if="allParams.filter(p => p.location === 'query').length" class="param-group">
      <label class="param-label">Query параметры</label>
      <div v-for="p in allParams.filter(p => p.location === 'query')" :key="p.name" class="param-row">
        <code class="param-name">{{ p.name }}</code>
        <input v-model="queryParams[p.name]" class="param-input" :placeholder="p.example ?? p.type" />
      </div>
    </div>

    <!-- Request Body -->
    <div v-if="endpoint.requestBody && endpoint.method !== 'GET'" class="param-group">
      <label class="param-label">Тело запроса</label>
      <textarea v-model="bodyText" class="body-textarea" rows="6" />
    </div>

    <!-- Доп. заголовки -->
    <details class="extra-headers">
      <summary class="extra-summary">Дополнительные заголовки</summary>
      <textarea v-model="extraHeaders" class="body-textarea" rows="3" placeholder="Header-Name: value&#10;Another-Header: value" />
    </details>

    <!-- Кнопки -->
    <div class="actions">
      <button class="btn-send" :disabled="sending" @click="send">
        {{ sending ? 'Отправка...' : 'Отправить' }}
      </button>
      <button class="btn-curl" @click="showCurl = !showCurl">cURL</button>
    </div>

    <CodeBlock v-if="showCurl" :code="toCurl()" language="bash" />

    <!-- Результат -->
    <div v-if="responseStatus !== null || responseError" class="response-section">
      <div class="response-meta">
        <span v-if="responseStatus !== null" class="response-status" :style="{ color: responseStatus < 400 ? 'var(--color-success)' : 'var(--color-error)' }">
          {{ responseStatus }}
        </span>
        <span v-if="responseError" class="response-error">{{ responseError }}</span>
        <span v-if="responseTime !== null" class="response-time">{{ responseTime }}ms</span>
      </div>

      <details v-if="responseHeaders" class="response-headers-details">
        <summary class="extra-summary">Заголовки ответа</summary>
        <pre class="response-headers-pre">{{ responseHeaders }}</pre>
      </details>

      <CodeBlock v-if="responseBody" :code="responseBody" language="json" />
    </div>
  </div>
</template>

<style scoped>
.tester {
  display: flex;
  flex-direction: column;
  gap: 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-bg);
  padding: 16px;
}
.env-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--color-text-muted);
}
.env-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-accent);
}
.env-arrow {
  color: var(--color-text-muted);
}
.env-url {
  font-size: 11px;
}
.param-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.param-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--color-text-muted);
}
.param-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.param-name {
  font-size: 12px;
  width: 112px;
  flex-shrink: 0;
}
.param-input {
  flex: 1;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  padding: 6px 10px;
  font-size: 13px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  color: var(--color-text);
  outline: none;
}
.param-input:focus {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 40%, transparent);
}
.body-textarea {
  width: 100%;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  padding: 8px 12px;
  font-size: 12px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  color: var(--color-text);
  outline: none;
  resize: vertical;
}
.body-textarea:focus {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 40%, transparent);
}
.extra-headers {
  font-size: 12px;
}
.extra-summary {
  cursor: pointer;
  color: var(--color-text-muted);
  font-size: 12px;
}
.extra-summary:hover {
  color: var(--color-text-secondary);
}
.actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.btn-send {
  border-radius: var(--radius-md);
  background: var(--color-accent);
  color: #fff;
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 500;
  border: none;
  cursor: pointer;
  transition: background-color 0.15s;
}
.btn-send:hover {
  background: var(--color-accent-hover);
}
.btn-send:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.btn-curl {
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: none;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: background-color 0.15s;
}
.btn-curl:hover {
  background: var(--color-bg-tertiary);
}
.response-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-top: 1px solid var(--color-border);
  padding-top: 12px;
}
.response-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 13px;
}
.response-status {
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  font-weight: 700;
}
.response-error {
  color: var(--color-error);
}
.response-time {
  font-size: 12px;
  color: var(--color-text-muted);
}
.response-headers-details {
  font-size: 12px;
}
.response-headers-pre {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-secondary);
  white-space: pre-wrap;
}
</style>
