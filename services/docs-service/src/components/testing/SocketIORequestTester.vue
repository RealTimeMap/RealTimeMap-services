<script setup lang="ts">
import { ref, computed, onBeforeUnmount } from 'vue'
import { io, Socket } from 'socket.io-client'
import type { SocketIOEvent } from '@/types'
import { useEnvironmentStore } from '@/stores/environment'
import { useAuthStore } from '@/stores/auth'
import { getServiceUrl } from '@/utils/env'
import { ENVIRONMENT_LABELS } from '@/constants'
import CodeBlock from '@/components/common/CodeBlock.vue'

const props = defineProps<{ event: SocketIOEvent; serviceId: string }>()

const envStore = useEnvironmentStore()
const authStore = useAuthStore()

const baseUrl = computed(() => getServiceUrl(props.serviceId, envStore.current))

const namespace = computed(() => {
  const ns = props.event.namespace
  if (!ns) return ''
  return ns.startsWith('/') ? ns : '/' + ns
})

const fullUrl = computed(() => {
  const base = baseUrl.value.replace(/\/$/, '')
  return `${base}${namespace.value}`
})

const socket = ref<Socket | null>(null)
const connectionState = ref<'disconnected' | 'connecting' | 'connected' | 'error'>('disconnected')
const connectionError = ref<string | null>(null)

type LogEntry = {
  id: number
  ts: string
  type: 'system' | 'sent' | 'received' | 'ack' | 'error'
  label: string
  data?: unknown
}
const logs = ref<LogEntry[]>([])
let logIdCounter = 0

function addLog(type: LogEntry['type'], label: string, data?: unknown) {
  const ts = new Date().toLocaleTimeString('ru-RU', { hour12: false })
  logs.value.unshift({ id: ++logIdCounter, ts, type, label, data })
  if (logs.value.length > 200) logs.value.length = 200
}

function clearLogs() {
  logs.value = []
}

const extraHeaders = ref('')
const sendPayloadText = ref(props.event.payload?.example ? JSON.stringify(props.event.payload.example, null, 2) : '')
const customEventName = ref(props.event.name)
const useAck = ref(true)

const listenedEvents = ref<string[]>([])
const newListenEvent = ref('')

function parseExtraHeaders(): Record<string, string> {
  const h: Record<string, string> = {}
  if (!extraHeaders.value.trim()) return h
  for (const line of extraHeaders.value.split('\n')) {
    const idx = line.indexOf(':')
    if (idx > 0) h[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
  }
  return h
}

function buildAuth(): Record<string, string> {
  const auth: Record<string, string> = {}
  const token = authStore.getToken(envStore.current)
  if (token) auth.token = token
  return auth
}

function connect() {
  if (!baseUrl.value) {
    connectionError.value = 'URL для текущего окружения не задан'
    connectionState.value = 'error'
    return
  }
  if (socket.value) {
    disconnect()
  }
  connectionState.value = 'connecting'
  connectionError.value = null
  addLog('system', `Подключение к ${fullUrl.value}`)

  try {
    const s = io(fullUrl.value, {
      transports: ['websocket', 'polling'],
      auth: buildAuth(),
      extraHeaders: parseExtraHeaders(),
      reconnection: false,
      timeout: 10000,
    })

    s.on('connect', () => {
      connectionState.value = 'connected'
      addLog('system', `Подключено (id: ${s.id})`)
      // Подписка на серверные события из документации (server-to-client) автоматически
      if (props.event.direction === 'server-to-client' && !listenedEvents.value.includes(props.event.name)) {
        listenedEvents.value.push(props.event.name)
      }
      for (const name of listenedEvents.value) attachListener(name)
    })

    s.on('connect_error', (err: Error) => {
      connectionState.value = 'error'
      connectionError.value = err.message
      addLog('error', 'Ошибка соединения', err.message)
    })

    s.on('disconnect', (reason: string) => {
      connectionState.value = 'disconnected'
      addLog('system', `Отключено (${reason})`)
    })

    socket.value = s
  } catch (e) {
    connectionState.value = 'error'
    connectionError.value = e instanceof Error ? e.message : 'Неизвестная ошибка'
  }
}

function disconnect() {
  if (socket.value) {
    socket.value.removeAllListeners()
    socket.value.disconnect()
    socket.value = null
  }
  connectionState.value = 'disconnected'
}

function attachListener(name: string) {
  if (!socket.value) return
  socket.value.off(name)
  socket.value.on(name, (...args: unknown[]) => {
    const data = args.length === 1 ? args[0] : args
    addLog('received', name, data)
  })
}

function addListener() {
  const name = newListenEvent.value.trim()
  if (!name || listenedEvents.value.includes(name)) return
  listenedEvents.value.push(name)
  attachListener(name)
  newListenEvent.value = ''
}

function removeListener(name: string) {
  listenedEvents.value = listenedEvents.value.filter(n => n !== name)
  if (socket.value) socket.value.off(name)
}

function emit() {
  if (!socket.value || connectionState.value !== 'connected') {
    addLog('error', 'Не подключено к серверу')
    return
  }
  const eventName = customEventName.value.trim() || props.event.name
  let payload: unknown = undefined
  const txt = sendPayloadText.value.trim()
  if (txt) {
    try {
      payload = JSON.parse(txt)
    } catch {
      payload = txt
    }
  }

  addLog('sent', eventName, payload)

  if (useAck.value) {
    socket.value.timeout(15000).emit(eventName, payload, (err: unknown, response: unknown) => {
      if (err) {
        addLog('error', `${eventName} ack timeout`, err instanceof Error ? err.message : err)
      } else {
        addLog('ack', `${eventName} ack`, response)
      }
    })
  } else {
    socket.value.emit(eventName, payload)
  }
}

function logCodeExample(): string {
  return JSON.stringify(logs.value, null, 2)
}

onBeforeUnmount(() => {
  disconnect()
})

const stateLabel = computed(() => {
  switch (connectionState.value) {
    case 'connected': return 'Подключено'
    case 'connecting': return 'Подключение...'
    case 'error': return 'Ошибка'
    default: return 'Не подключено'
  }
})

function formatLog(entry: LogEntry): string {
  if (entry.data === undefined) return ''
  try {
    return JSON.stringify(entry.data, null, 2)
  } catch {
    return String(entry.data)
  }
}

const showLogJson = ref(false)
</script>

<template>
  <div class="tester">
    <div class="env-indicator">
      <span class="env-dot" :class="connectionState" />
      {{ ENVIRONMENT_LABELS[envStore.current] ?? envStore.current }}
      <span class="env-arrow">→</span>
      <code class="env-url">{{ fullUrl || '<URL не задан>' }}</code>
      <span class="state-label" :class="connectionState">{{ stateLabel }}</span>
    </div>

    <div v-if="connectionError" class="error-banner">{{ connectionError }}</div>

    <!-- Подключение -->
    <div class="connection-controls">
      <button
        v-if="connectionState !== 'connected'"
        class="btn-connect"
        :disabled="connectionState === 'connecting' || !baseUrl"
        @click="connect"
      >
        {{ connectionState === 'connecting' ? 'Подключение...' : 'Подключиться' }}
      </button>
      <button v-else class="btn-disconnect" @click="disconnect">Отключиться</button>
    </div>

    <!-- Доп. заголовки и auth -->
    <details class="extra-headers">
      <summary class="extra-summary">Дополнительные заголовки (extraHeaders)</summary>
      <textarea v-model="extraHeaders" class="body-textarea" rows="3" placeholder="Header-Name: value&#10;Another-Header: value" />
      <p class="hint">Токен из настроек авторизации передаётся через `auth.token` при подключении.</p>
    </details>

    <!-- Отправка события (для client-to-server) -->
    <div v-if="event.direction === 'client-to-server'" class="param-group">
      <label class="param-label">Отправка события</label>

      <div class="emit-row">
        <code class="param-name">event</code>
        <input v-model="customEventName" class="param-input" placeholder="имя события" />
      </div>

      <div class="emit-row">
        <code class="param-name">payload</code>
        <textarea v-model="sendPayloadText" class="body-textarea" rows="6" placeholder='{ "key": "value" }' />
      </div>

      <label class="ack-toggle">
        <input v-model="useAck" type="checkbox" />
        <span>Ожидать ack (таймаут 15 сек)</span>
      </label>

      <div class="actions">
        <button class="btn-send" :disabled="connectionState !== 'connected'" @click="emit">
          Отправить
        </button>
      </div>
    </div>

    <!-- Слушатели событий -->
    <div class="param-group">
      <label class="param-label">Прослушиваемые события</label>
      <div class="listeners">
        <span v-for="name in listenedEvents" :key="name" class="listener-chip">
          {{ name }}
          <button class="chip-remove" :disabled="!!socket && connectionState === 'connected' && false" @click="removeListener(name)" title="Удалить">
            ×
          </button>
        </span>
        <span v-if="!listenedEvents.length" class="hint">Нет подписок</span>
      </div>
      <div class="add-listener-row">
        <input v-model="newListenEvent" class="param-input" placeholder="имя события" @keyup.enter="addListener" />
        <button class="btn-add-field" @click="addListener">+ Добавить</button>
      </div>
    </div>

    <!-- Лог -->
    <div class="log-section">
      <div class="log-header">
        <label class="param-label">Лог событий</label>
        <div class="log-actions">
          <button class="btn-curl" @click="showLogJson = !showLogJson">
            {{ showLogJson ? 'Скрыть JSON' : 'Показать JSON' }}
          </button>
          <button class="btn-curl" @click="clearLogs">Очистить</button>
        </div>
      </div>

      <CodeBlock v-if="showLogJson && logs.length" :code="logCodeExample()" language="json" />

      <div v-else class="log-list">
        <div v-if="!logs.length" class="empty-log">Событий пока нет</div>
        <div
          v-for="entry in logs"
          :key="entry.id"
          class="log-entry"
          :class="entry.type"
        >
          <div class="log-row">
            <span class="log-ts">{{ entry.ts }}</span>
            <span class="log-type">{{ entry.type }}</span>
            <code class="log-label">{{ entry.label }}</code>
          </div>
          <pre v-if="entry.data !== undefined" class="log-data">{{ formatLog(entry) }}</pre>
        </div>
      </div>
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
  flex-wrap: wrap;
}
.env-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-text-muted);
  transition: background-color 0.2s;
}
.env-dot.connected { background: var(--color-success); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-success) 25%, transparent); }
.env-dot.connecting { background: var(--color-warning); animation: pulse 1.2s infinite; }
.env-dot.error { background: var(--color-error); }
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
.env-arrow { color: var(--color-text-muted); }
.env-url { font-size: 11px; }
.state-label {
  margin-left: auto;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.state-label.connected { color: var(--color-success); }
.state-label.connecting { color: var(--color-warning); }
.state-label.error { color: var(--color-error); }
.state-label.disconnected { color: var(--color-text-muted); }

.error-banner {
  border-radius: var(--radius-md);
  border: 1px solid color-mix(in srgb, var(--color-error) 25%, transparent);
  background: color-mix(in srgb, var(--color-error) 8%, transparent);
  color: var(--color-error);
  padding: 8px 12px;
  font-size: 12px;
}

.connection-controls { display: flex; gap: 8px; }
.btn-connect, .btn-disconnect, .btn-send {
  border-radius: var(--radius-md);
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 500;
  border: none;
  cursor: pointer;
  transition: background-color 0.15s;
}
.btn-connect, .btn-send {
  background: var(--color-accent);
  color: #fff;
}
.btn-connect:hover, .btn-send:hover { background: var(--color-accent-hover); }
.btn-connect:disabled, .btn-send:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-disconnect {
  background: color-mix(in srgb, var(--color-error) 12%, transparent);
  color: var(--color-error);
  border: 1px solid color-mix(in srgb, var(--color-error) 25%, transparent);
}
.btn-disconnect:hover { background: color-mix(in srgb, var(--color-error) 18%, transparent); }

.param-group { display: flex; flex-direction: column; gap: 8px; }
.param-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--color-text-muted);
}
.param-name {
  font-size: 12px;
  width: 80px;
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
.emit-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.body-textarea {
  flex: 1;
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
.ack-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--color-text-secondary);
  cursor: pointer;
}
.actions { display: flex; gap: 8px; }
.btn-curl {
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: none;
  padding: 6px 10px;
  font-size: 11px;
  color: var(--color-text-secondary);
  cursor: pointer;
}
.btn-curl:hover { background: var(--color-bg-tertiary); }

.extra-headers { font-size: 12px; }
.extra-summary { cursor: pointer; color: var(--color-text-muted); font-size: 12px; }
.extra-summary:hover { color: var(--color-text-secondary); }
.hint { font-size: 11px; color: var(--color-text-muted); margin: 6px 0 0; }

.listeners {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-height: 24px;
  align-items: center;
}
.listener-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border-radius: var(--radius-full);
  padding: 2px 4px 2px 10px;
  font-size: 11px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  background: color-mix(in srgb, var(--color-info, #3b82f6) 10%, transparent);
  color: var(--color-info, #3b82f6);
  border: 1px solid color-mix(in srgb, var(--color-info, #3b82f6) 25%, transparent);
}
.chip-remove {
  border: none;
  background: none;
  color: inherit;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0 4px;
}
.add-listener-row { display: flex; gap: 8px; align-items: center; }
.btn-add-field {
  border: none;
  background: none;
  color: var(--color-accent);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  padding: 4px 0;
}
.btn-add-field:hover { text-decoration: underline; }

/* Log */
.log-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-top: 1px solid var(--color-border);
  padding-top: 12px;
}
.log-header { display: flex; align-items: center; justify-content: space-between; }
.log-actions { display: flex; gap: 6px; }
.log-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 360px;
  overflow-y: auto;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-bg-tertiary);
  padding: 8px;
}
.empty-log {
  padding: 20px;
  text-align: center;
  color: var(--color-text-muted);
  font-size: 12px;
}
.log-entry {
  border-radius: var(--radius-sm);
  padding: 6px 8px;
  border-left: 3px solid var(--color-border);
}
.log-entry.system { border-left-color: var(--color-text-muted); }
.log-entry.sent { border-left-color: var(--color-socketio); background: color-mix(in srgb, var(--color-socketio) 5%, transparent); }
.log-entry.received { border-left-color: var(--color-info, #3b82f6); background: color-mix(in srgb, var(--color-info, #3b82f6) 5%, transparent); }
.log-entry.ack { border-left-color: var(--color-success); background: color-mix(in srgb, var(--color-success) 5%, transparent); }
.log-entry.error { border-left-color: var(--color-error); background: color-mix(in srgb, var(--color-error) 5%, transparent); }
.log-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
}
.log-ts { color: var(--color-text-muted); font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace; }
.log-type {
  font-size: 10px;
  text-transform: uppercase;
  font-weight: 700;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
}
.log-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text);
}
.log-data {
  margin: 4px 0 0;
  padding: 6px 8px;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-sm);
  font-size: 11px;
  color: var(--color-text-secondary);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 240px;
  overflow-y: auto;
}
</style>
