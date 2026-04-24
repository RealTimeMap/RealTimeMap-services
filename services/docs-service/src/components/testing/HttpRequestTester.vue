<script setup lang="ts">
import { ref, computed } from 'vue'
import type { HttpEndpoint, Parameter, ContentType } from '@/types'
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

const CONTENT_TYPE_OPTIONS: { value: ContentType; label: string; mime: string }[] = [
  { value: 'json', label: 'JSON', mime: 'application/json' },
  { value: 'form-data', label: 'Form Data', mime: 'multipart/form-data' },
  { value: 'x-www-form-urlencoded', label: 'URL Encoded', mime: 'application/x-www-form-urlencoded' },
  { value: 'text', label: 'Text', mime: 'text/plain' },
  { value: 'binary', label: 'Binary', mime: 'application/octet-stream' },
]

const contentType = ref<ContentType>(props.endpoint.requestBody?.contentType ?? 'json')

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

// Form-data поля
const formFields = ref<{ key: string; value: string; type: 'text' | 'file' }[]>(
  initFormFields(),
)
const fileInputs = ref<Record<number, File | null>>({})

function initFormFields(): { key: string; value: string; type: 'text' | 'file' }[] {
  const schema = props.endpoint.requestBody?.schema
  if (!schema?.length) return [{ key: '', value: '', type: 'text' }]
  return schema.map(f => ({
    key: f.name,
    value: f.example != null ? String(f.example) : '',
    type: (f.type === 'file' || f.type === 'File') ? 'file' as const : 'text' as const,
  }))
}

// Binary file
const binaryFile = ref<File | null>(null)

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

function addFormField() {
  formFields.value.push({ key: '', value: '', type: 'text' })
}

function removeFormField(index: number) {
  formFields.value.splice(index, 1)
  delete fileInputs.value[index]
}

function onFileSelect(index: number, event: Event) {
  const input = event.target as HTMLInputElement
  fileInputs.value[index] = input.files?.[0] ?? null
}

function onBinaryFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  binaryFile.value = input.files?.[0] ?? null
}

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

function getMimeType(): string {
  return CONTENT_TYPE_OPTIONS.find(o => o.value === contentType.value)?.mime ?? 'application/json'
}

function buildHeaders(): Record<string, string> {
  const h: Record<string, string> = {}
  for (const [key, val] of Object.entries(headerParams.value)) {
    if (val) h[key] = val
  }
  const token = authStore.getToken(envStore.current)
  if (token && !h['Authorization']) h['Authorization'] = `Bearer ${token}`

  if (props.endpoint.requestBody && props.endpoint.method !== 'GET') {
    // Для form-data браузер сам выставит Content-Type с boundary
    if (contentType.value !== 'form-data' && contentType.value !== 'binary') {
      h['Content-Type'] = getMimeType()
    }
  }

  if (extraHeaders.value.trim()) {
    for (const line of extraHeaders.value.split('\n')) {
      const idx = line.indexOf(':')
      if (idx > 0) h[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
    }
  }
  return h
}

function buildBody(): BodyInit | undefined {
  if (props.endpoint.method === 'GET' || !props.endpoint.requestBody) return undefined

  switch (contentType.value) {
    case 'json':
    case 'text':
      return bodyText.value || undefined

    case 'form-data': {
      const fd = new FormData()
      for (let i = 0; i < formFields.value.length; i++) {
        const f = formFields.value[i]
        if (!f.key) continue
        if (f.type === 'file') {
          const file = fileInputs.value[i]
          if (file) fd.append(f.key, file)
        } else {
          fd.append(f.key, f.value)
        }
      }
      return fd
    }

    case 'x-www-form-urlencoded': {
      const params = new URLSearchParams()
      for (const f of formFields.value) {
        if (f.key) params.set(f.key, f.value)
      }
      return params.toString()
    }

    case 'binary':
      return binaryFile.value ?? undefined
  }
}

function toCurl(): string {
  const url = buildUrl()
  const headers = buildHeaders()
  let cmd = `curl -X ${props.endpoint.method} '${url}'`
  for (const [k, v] of Object.entries(headers)) cmd += ` \\\n  -H '${k}: ${v}'`

  if (props.endpoint.method !== 'GET' && props.endpoint.requestBody) {
    switch (contentType.value) {
      case 'json':
      case 'text':
        if (bodyText.value) cmd += ` \\\n  -d '${bodyText.value.replace(/\n/g, '')}'`
        break
      case 'form-data':
        for (const f of formFields.value) {
          if (!f.key) continue
          if (f.type === 'file') cmd += ` \\\n  -F '${f.key}=@<file>'`
          else cmd += ` \\\n  -F '${f.key}=${f.value}'`
        }
        break
      case 'x-www-form-urlencoded':
        for (const f of formFields.value) {
          if (f.key) cmd += ` \\\n  --data-urlencode '${f.key}=${f.value}'`
        }
        break
      case 'binary':
        cmd += ` \\\n  --data-binary @<file>`
        break
    }
  }
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
  const body = buildBody()
  const start = performance.now()

  try {
    const res = await fetch(url, {
      method: props.endpoint.method,
      headers,
      body,
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

const hasBody = computed(() => props.endpoint.requestBody && props.endpoint.method !== 'GET')
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

    <!-- Content Type + Request Body -->
    <div v-if="hasBody" class="param-group">
      <div class="body-header">
        <label class="param-label">Тело запроса</label>
        <div class="content-type-selector">
          <button
            v-for="opt in CONTENT_TYPE_OPTIONS"
            :key="opt.value"
            class="ct-chip"
            :class="{ active: contentType === opt.value }"
            @click="contentType = opt.value"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>

      <!-- JSON / Text -->
      <textarea
        v-if="contentType === 'json' || contentType === 'text'"
        v-model="bodyText"
        class="body-textarea"
        rows="6"
        :placeholder="contentType === 'json' ? '{ }' : 'Текст...'"
      />

      <!-- Form Data / URL Encoded -->
      <div v-if="contentType === 'form-data' || contentType === 'x-www-form-urlencoded'" class="form-fields">
        <div v-for="(field, i) in formFields" :key="i" class="form-field-row">
          <input v-model="field.key" class="param-input form-key" placeholder="Ключ" />

          <template v-if="contentType === 'form-data'">
            <select v-model="field.type" class="field-type-select">
              <option value="text">Text</option>
              <option value="file">File</option>
            </select>
          </template>

          <input
            v-if="field.type === 'text'"
            v-model="field.value"
            class="param-input form-value"
            placeholder="Значение"
          />
          <input
            v-else
            type="file"
            class="file-input"
            @change="onFileSelect(i, $event)"
          />

          <button class="btn-remove" @click="removeFormField(i)" title="Удалить поле">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 6L6 18M6 6l12 12" />
            </svg>
          </button>
        </div>
        <button class="btn-add-field" @click="addFormField">+ Добавить поле</button>
      </div>

      <!-- Binary -->
      <div v-if="contentType === 'binary'" class="binary-upload">
        <input type="file" class="file-input" @change="onBinaryFileSelect" />
        <span v-if="binaryFile" class="file-name">{{ binaryFile.name }} ({{ (binaryFile.size / 1024).toFixed(1) }} KB)</span>
      </div>
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

/* Body header с content-type selector */
.body-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.content-type-selector {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}
.ct-chip {
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border);
  padding: 3px 10px;
  font-size: 11px;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: none;
  cursor: pointer;
  transition: all 0.15s;
}
.ct-chip:hover {
  border-color: var(--color-border-hover);
}
.ct-chip.active {
  border-color: var(--color-accent);
  background: color-mix(in srgb, var(--color-accent) 10%, transparent);
  color: var(--color-accent);
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

/* Form fields */
.form-fields {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.form-field-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.form-key {
  flex: 0 0 140px;
}
.form-value {
  flex: 1;
}
.field-type-select {
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  padding: 6px 8px;
  font-size: 12px;
  color: var(--color-text);
  outline: none;
  cursor: pointer;
}
.file-input {
  flex: 1;
  font-size: 12px;
  color: var(--color-text-secondary);
}
.file-input::file-selector-button {
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  background: var(--color-bg-tertiary);
  padding: 4px 10px;
  font-size: 11px;
  color: var(--color-text);
  cursor: pointer;
  margin-right: 8px;
}
.btn-remove {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  border: none;
  background: none;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: color 0.15s, background-color 0.15s;
}
.btn-remove:hover {
  color: var(--color-error);
  background: color-mix(in srgb, var(--color-error) 10%, transparent);
}
.btn-add-field {
  align-self: flex-start;
  border: none;
  background: none;
  color: var(--color-accent);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  padding: 4px 0;
}
.btn-add-field:hover {
  text-decoration: underline;
}

/* Binary upload */
.binary-upload {
  display: flex;
  align-items: center;
  gap: 12px;
}
.file-name {
  font-size: 12px;
  color: var(--color-text-secondary);
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
