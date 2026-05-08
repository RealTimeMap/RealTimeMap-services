<script setup lang="ts">
import { ref, computed } from 'vue'
import type { SocketIOEvent } from '@/types'
import { useSharedStore } from '@/stores/shared'
import SocketIODirectionBadge from '@/components/common/SocketIODirectionBadge.vue'
import SchemaViewer from '@/components/common/SchemaViewer.vue'
import CodeBlock from '@/components/common/CodeBlock.vue'
import SocketIORequestTester from '@/components/testing/SocketIORequestTester.vue'

const props = defineProps<{ event: SocketIOEvent; serviceId: string }>()

const isOpen = ref(false)
const activeTab = ref<'overview' | 'schemas' | 'testing'>('overview')
const sharedStore = useSharedStore()

const directionLabel = computed(() =>
  props.event.direction === 'client-to-server' ? 'Client → Server' : 'Server → Client',
)

const sharedErrors = computed(() => {
  if (!props.event.errors?.length) return []
  return props.event.errors
    .map(id => sharedStore.getError(id))
    .filter((e): e is NonNullable<typeof e> => e != null)
})

const ackHasContent = computed(() => {
  const ack = props.event.ack
  if (!ack) return false
  return !!(ack.schema?.length || ack.example !== undefined || ack.branches?.length)
})

function exampleCode(language: 'js' | 'ts'): string {
  const ev = props.event
  const ns = ev.namespace ? (ev.namespace.startsWith('/') ? ev.namespace : '/' + ev.namespace) : ''
  const importLine = language === 'ts'
    ? `import { io, type Socket } from 'socket.io-client'\n\n`
    : `import { io } from 'socket.io-client'\n\n`
  const socketTyped = language === 'ts' ? `const socket: Socket = io` : `const socket = io`
  const payload = ev.payload?.example !== undefined ? JSON.stringify(ev.payload.example, null, 2) : '{}'

  if (ev.direction === 'client-to-server') {
    if (ev.ack) {
      return `${importLine}${socketTyped}('https://your-host${ns}', { auth: { token: '<JWT>' } })

socket.on('connect', () => {
  socket.timeout(15000).emit('${ev.name}', ${payload}, (err, response) => {
    if (err) {
      console.error('ack timeout', err)
      return
    }
    console.log('ack', response)
  })
})`
    }
    return `${importLine}${socketTyped}('https://your-host${ns}', { auth: { token: '<JWT>' } })

socket.on('connect', () => {
  socket.emit('${ev.name}', ${payload})
})`
  }

  return `${importLine}${socketTyped}('https://your-host${ns}', { auth: { token: '<JWT>' } })

socket.on('${ev.name}', (data) => {
  console.log('${ev.name}', data)
})`
}

const codeLang = ref<'js' | 'ts'>('ts')
</script>

<template>
  <div :id="'event-' + event.id" class="accordion" :class="{ open: isOpen }">
    <button class="accordion-header" @click="isOpen = !isOpen">
      <SocketIODirectionBadge :direction="event.direction" compact />
      <code class="event-name">{{ event.name }}</code>
      <span v-if="event.namespace" class="namespace-badge">{{ event.namespace }}</span>
      <span class="event-summary truncate">{{ event.summary }}</span>
      <span v-if="event.ack" class="ack-badge" title="Поддерживает acknowledgement">ack</span>
      <span v-if="event.auth" class="auth-badge" title="Требуется авторизация">
        <svg class="auth-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
          <path d="M7 11V7a5 5 0 0 1 10 0v4" />
        </svg>
      </span>
      <svg class="chevron" :class="{ rotated: isOpen }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M6 9l6 6 6-6" />
      </svg>
    </button>

    <div v-if="isOpen" class="accordion-body">
      <p v-if="event.description" class="event-desc">{{ event.description }}</p>

      <!-- Метаинформация -->
      <div class="meta-grid">
        <div class="meta-item">
          <span class="meta-label">Направление</span>
          <span class="meta-value">{{ directionLabel }}</span>
        </div>
        <div v-if="event.namespace" class="meta-item">
          <span class="meta-label">Namespace</span>
          <code class="meta-value mono">{{ event.namespace }}</code>
        </div>
        <div v-if="event.room" class="meta-item">
          <span class="meta-label">Room</span>
          <code class="meta-value mono">{{ event.room }}</code>
        </div>
        <div class="meta-item">
          <span class="meta-label">Acknowledgement</span>
          <span class="meta-value">{{ event.ack ? 'Да' : 'Нет' }}</span>
        </div>
      </div>

      <!-- Вкладки -->
      <div class="tab-bar">
        <button class="tab-btn" :class="{ active: activeTab === 'overview' }" @click="activeTab = 'overview'">
          Обзор
        </button>
        <button class="tab-btn" :class="{ active: activeTab === 'schemas' }" @click="activeTab = 'schemas'">
          Схемы
        </button>
        <button class="tab-btn" :class="{ active: activeTab === 'testing' }" @click="activeTab = 'testing'">
          Тестирование
        </button>
      </div>

      <!-- Вкладка: Обзор -->
      <div v-if="activeTab === 'overview'" class="tab-content">
        <section v-if="event.payload">
          <h4 class="section-title">Payload</h4>
          <p v-if="event.payload.description" class="body-desc">{{ event.payload.description }}</p>
          <div v-if="event.payload.example !== undefined" class="example-block">
            <h5 class="subsection-title">Пример</h5>
            <CodeBlock :code="JSON.stringify(event.payload.example, null, 2)" language="json" />
          </div>
        </section>

        <section v-if="ackHasContent">
          <h4 class="section-title">Acknowledgement</h4>
          <p v-if="event.ack?.description" class="body-desc">{{ event.ack.description }}</p>

          <!-- Если есть варианты (branches) — показываем как табы успех/ошибка -->
          <div v-if="event.ack?.branches?.length" class="ack-branches">
            <div v-for="branch in event.ack.branches" :key="branch.name" class="ack-branch">
              <div class="ack-branch-header">
                <span class="ack-branch-name">{{ branch.name }}</span>
                <span v-if="branch.description" class="ack-branch-desc">{{ branch.description }}</span>
              </div>
              <div v-if="branch.example !== undefined" class="example-block">
                <CodeBlock :code="JSON.stringify(branch.example, null, 2)" language="json" />
              </div>
            </div>
          </div>

          <div v-if="event.ack?.example !== undefined" class="example-block">
            <h5 class="subsection-title">Пример ack</h5>
            <CodeBlock :code="JSON.stringify(event.ack.example, null, 2)" language="json" />
          </div>
        </section>

        <section v-if="sharedErrors.length">
          <h4 class="section-title">Возможные ошибки</h4>
          <div class="errors-list">
            <div v-for="err in sharedErrors" :key="err.id" class="error-item">
              <span class="error-code">{{ err.statusCode }}</span>
              <span class="error-desc">{{ err.description }}</span>
            </div>
          </div>
        </section>

        <section>
          <h4 class="section-title">Пример кода</h4>
          <div class="code-lang-switch">
            <button class="lang-chip" :class="{ active: codeLang === 'ts' }" @click="codeLang = 'ts'">TypeScript</button>
            <button class="lang-chip" :class="{ active: codeLang === 'js' }" @click="codeLang = 'js'">JavaScript</button>
          </div>
          <CodeBlock :code="event.codeExample ?? exampleCode(codeLang)" :language="codeLang" />
        </section>
      </div>

      <!-- Вкладка: Схемы -->
      <div v-if="activeTab === 'schemas'" class="tab-content">
        <section v-if="event.payload?.schema?.length">
          <h4 class="section-title">Схема payload</h4>
          <SchemaViewer :fields="event.payload.schema" />
        </section>

        <section v-if="event.ack?.schema?.length">
          <h4 class="section-title">Схема ack</h4>
          <SchemaViewer :fields="event.ack.schema" />
        </section>

        <section v-if="event.ack?.branches?.length">
          <h4 class="section-title">Варианты ack</h4>
          <div v-for="branch in event.ack.branches" :key="branch.name" class="schema-block">
            <div class="schema-block-header">
              <span class="branch-name-badge">{{ branch.name }}</span>
              <span v-if="branch.description" class="schema-block-desc">{{ branch.description }}</span>
            </div>
            <SchemaViewer :fields="branch.schema" />
          </div>
        </section>

        <div v-if="!event.payload?.schema?.length && !event.ack?.schema?.length && !event.ack?.branches?.length" class="empty-state">
          Схемы не определены для этого события
        </div>
      </div>

      <!-- Вкладка: Тестирование -->
      <div v-if="activeTab === 'testing'" class="tab-content">
        <SocketIORequestTester :event="event" :service-id="serviceId" />
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
.accordion.open { background: var(--color-bg-secondary); }
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
.event-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text);
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
}
.namespace-badge {
  font-size: 11px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  color: var(--color-text-muted);
  background: var(--color-bg-tertiary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: 1px 6px;
}
.event-summary {
  font-size: 13px;
  color: var(--color-text-secondary);
  flex: 1;
}
.ack-badge {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-success);
  background: color-mix(in srgb, var(--color-success) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--color-success) 25%, transparent);
  border-radius: var(--radius-sm);
  padding: 2px 6px;
}
.auth-badge {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  color: var(--color-warning);
}
.auth-icon { width: 14px; height: 14px; }
.chevron {
  width: 16px;
  height: 16px;
  color: var(--color-text-muted);
  flex-shrink: 0;
  transition: transform 0.2s;
}
.chevron.rotated { transform: rotate(180deg); }
.accordion-body {
  padding: 0 16px 16px;
  border-top: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.event-desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  padding-top: 16px;
  margin: 0;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 8px;
  padding: 12px;
  border-radius: var(--radius-md);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
}
.meta-item { display: flex; flex-direction: column; gap: 2px; }
.meta-label {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
}
.meta-value {
  font-size: 13px;
  color: var(--color-text);
}
.meta-value.mono {
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  font-size: 12px;
}

.tab-bar {
  display: flex;
  gap: 0;
  border-bottom: 1px solid var(--color-border);
  margin-top: 4px;
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
.tab-btn:hover { color: var(--color-text-secondary); }
.tab-btn.active {
  color: var(--color-accent);
  border-bottom-color: var(--color-accent);
}

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
.subsection-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-muted);
  margin: 8px 0 6px;
}
.body-desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin: 0 0 8px;
}
.example-block { margin-top: 4px; }

.ack-branches { display: flex; flex-direction: column; gap: 12px; }
.ack-branch {
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  padding: 10px 12px;
}
.ack-branch-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.ack-branch-name {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-accent);
}
.ack-branch-desc {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.errors-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.error-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  background: color-mix(in srgb, var(--color-error) 5%, transparent);
  border: 1px solid color-mix(in srgb, var(--color-error) 15%, transparent);
}
.error-code {
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  font-size: 12px;
  font-weight: 700;
  color: var(--color-error);
  flex-shrink: 0;
}
.error-desc {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.code-lang-switch {
  display: flex;
  gap: 4px;
  margin-bottom: 8px;
}
.lang-chip {
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
.lang-chip.active {
  border-color: var(--color-accent);
  background: color-mix(in srgb, var(--color-accent) 10%, transparent);
  color: var(--color-accent);
}

.schema-block {
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  padding: 12px;
}
.schema-block + .schema-block { margin-top: 12px; }
.schema-block-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.branch-name-badge {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-accent);
  background: color-mix(in srgb, var(--color-accent) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--color-accent) 25%, transparent);
  border-radius: var(--radius-sm);
  padding: 2px 8px;
}
.schema-block-desc {
  font-size: 12px;
  color: var(--color-text-secondary);
}
.empty-state {
  padding: 32px 0;
  text-align: center;
  color: var(--color-text-muted);
  font-size: 13px;
}
</style>
