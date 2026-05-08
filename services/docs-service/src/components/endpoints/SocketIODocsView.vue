<script setup lang="ts">
import { computed, ref } from 'vue'
import type { SocketIODocs, SocketIOEvent } from '@/types'
import SocketIODirectionBadge from '@/components/common/SocketIODirectionBadge.vue'
import SocketIOEventAccordion from './SocketIOEventAccordion.vue'

const props = defineProps<{ docs: SocketIODocs; serviceId: string }>()

const search = ref('')

type Group = {
  direction: 'client-to-server' | 'server-to-client'
  namespace: string
  events: SocketIOEvent[]
}

function groupKey(direction: string, namespace: string): string {
  return `${direction}::${namespace}`
}

const groups = computed<Group[]>(() => {
  const map = new Map<string, Group>()
  for (const ev of props.docs.events) {
    const ns = ev.namespace ?? '/'
    const key = groupKey(ev.direction, ns)
    if (!map.has(key)) {
      map.set(key, { direction: ev.direction, namespace: ns, events: [] })
    }
    map.get(key)!.events.push(ev)
  }
  // Сортируем: сначала client-to-server, затем server-to-client; внутри — по namespace
  return Array.from(map.values()).sort((a, b) => {
    if (a.direction !== b.direction) {
      return a.direction === 'client-to-server' ? -1 : 1
    }
    return a.namespace.localeCompare(b.namespace)
  })
})

const filteredGroups = computed(() => {
  if (!search.value) return groups.value
  const q = search.value.toLowerCase()
  const result: Group[] = []
  for (const g of groups.value) {
    const filtered = g.events.filter(
      ev =>
        ev.name.toLowerCase().includes(q) ||
        ev.summary.toLowerCase().includes(q) ||
        (ev.namespace?.toLowerCase().includes(q) ?? false),
    )
    if (filtered.length) result.push({ ...g, events: filtered })
  }
  return result
})

const collapsedGroups = ref(new Set<string>())
function toggleGroup(key: string) {
  if (collapsedGroups.value.has(key)) collapsedGroups.value.delete(key)
  else collapsedGroups.value.add(key)
}

function scrollToEvent(id: string) {
  document.getElementById('event-' + id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function groupTitle(group: Group): string {
  const dir = group.direction === 'client-to-server' ? 'Client → Server' : 'Server → Client'
  return group.namespace === '/' ? dir : `${dir} · ${group.namespace}`
}
</script>

<template>
  <div class="socketio-layout">
    <!-- Общая информация о соединении -->
    <div v-if="docs.baseUrl || docs.transport?.length || docs.namespaces?.length" class="connection-info">
      <h3 class="info-title">Информация о подключении</h3>
      <div class="info-grid">
        <div v-if="docs.baseUrl" class="info-item">
          <span class="info-label">Base URL</span>
          <code class="info-value mono">{{ docs.baseUrl }}</code>
        </div>
        <div v-if="docs.transport?.length" class="info-item">
          <span class="info-label">Транспорт</span>
          <span class="info-value">{{ docs.transport.join(', ') }}</span>
        </div>
      </div>
      <div v-if="docs.namespaces?.length" class="namespaces-section">
        <h4 class="section-title">Namespaces</h4>
        <div class="namespaces-list">
          <div v-for="ns in docs.namespaces" :key="ns.path" class="namespace-item">
            <code class="ns-path">{{ ns.path }}</code>
            <span v-if="ns.auth" class="ns-auth" title="Требуется авторизация">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                <path d="M7 11V7a5 5 0 0 1 10 0v4" />
              </svg>
            </span>
            <span v-if="ns.description" class="ns-desc">{{ ns.description }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="content-row">
      <!-- Sidebar -->
      <aside class="sidebar">
        <div class="sidebar-sticky">
          <input v-model="search" type="text" placeholder="Поиск событий..." class="search-input" />

          <nav class="sidebar-nav">
            <div v-for="group in filteredGroups" :key="groupKey(group.direction, group.namespace)">
              <button class="group-toggle" @click="toggleGroup(groupKey(group.direction, group.namespace))">
                <SocketIODirectionBadge :direction="group.direction" compact />
                <span class="ns-label" :title="group.namespace">{{ group.namespace }}</span>
                <svg
                  class="group-chevron"
                  :class="{ collapsed: collapsedGroups.has(groupKey(group.direction, group.namespace)) }"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path d="M6 9l6 6 6-6" />
                </svg>
              </button>
              <div v-if="!collapsedGroups.has(groupKey(group.direction, group.namespace))" class="group-items">
                <button v-for="ev in group.events" :key="ev.id" class="sidebar-item" @click="scrollToEvent(ev.id)">
                  <span class="sidebar-event-name truncate">{{ ev.name }}</span>
                  <span v-if="ev.ack" class="ack-mini">ack</span>
                  <svg v-if="ev.auth" class="sidebar-lock" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                    <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                  </svg>
                </button>
              </div>
            </div>
          </nav>
        </div>
      </aside>

      <!-- Main content -->
      <div class="main-content">
        <input v-model="search" type="text" placeholder="Поиск событий..." class="search-input mobile-search" />

        <template v-for="group in filteredGroups" :key="groupKey(group.direction, group.namespace)">
          <h3 class="group-title">{{ groupTitle(group) }}</h3>
          <SocketIOEventAccordion v-for="ev in group.events" :key="ev.id" :event="ev" :service-id="serviceId" />
        </template>

        <div v-if="filteredGroups.length === 0" class="empty-state">
          События не найдены
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.socketio-layout {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.connection-info {
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  padding: 16px;
}
.info-title {
  font-size: 14px;
  font-weight: 600;
  margin: 0 0 12px;
}
.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}
.info-item { display: flex; flex-direction: column; gap: 4px; }
.info-label {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--color-text-muted);
  letter-spacing: 0.05em;
}
.info-value { font-size: 13px; color: var(--color-text); }
.info-value.mono {
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  font-size: 12px;
}
.namespaces-section { margin-top: 12px; }
.section-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  margin: 0 0 8px;
}
.namespaces-list { display: flex; flex-direction: column; gap: 6px; }
.namespace-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
}
.ns-path {
  font-size: 12px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  color: var(--color-text);
  flex-shrink: 0;
}
.ns-auth {
  display: flex;
  width: 14px;
  height: 14px;
  color: var(--color-warning);
}
.ns-auth svg { width: 100%; height: 100%; }
.ns-desc { font-size: 12px; color: var(--color-text-secondary); }

.content-row { display: flex; gap: 24px; }
.sidebar { width: 288px; flex-shrink: 0; }
@media (max-width: 1024px) { .sidebar { display: none; } }
.sidebar-sticky {
  position: sticky;
  top: 72px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.search-input {
  width: 100%;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  padding: 8px 12px;
  font-size: 13px;
  color: var(--color-text);
  outline: none;
}
.search-input::placeholder { color: var(--color-text-muted); }
.search-input:focus {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 40%, transparent);
}
.mobile-search { display: none; margin-bottom: 16px; }
@media (max-width: 1024px) { .mobile-search { display: block; } }

.sidebar-nav { display: flex; flex-direction: column; gap: 8px; }
.group-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-text-secondary);
}
.group-toggle:hover { color: var(--color-text); }
.ns-label {
  font-size: 11px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  color: var(--color-text-muted);
  flex: 1;
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.group-chevron { width: 12px; height: 12px; transition: transform 0.2s; }
.group-chevron.collapsed { transform: rotate(-90deg); }
.group-items { display: flex; flex-direction: column; gap: 2px; padding-left: 4px; }
.sidebar-item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: var(--radius-md);
  text-align: left;
  background: none;
  border: none;
  cursor: pointer;
  color: inherit;
  transition: background-color 0.15s;
}
.sidebar-item:hover { background: var(--color-bg-tertiary); }
.sidebar-event-name {
  font-size: 12px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  color: var(--color-text-secondary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ack-mini {
  font-size: 9px;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--color-success);
  background: color-mix(in srgb, var(--color-success) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--color-success) 25%, transparent);
  border-radius: var(--radius-sm);
  padding: 1px 4px;
}
.sidebar-lock {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
  color: var(--color-warning);
}

.main-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.group-title {
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  margin: 8px 0 0;
}
.empty-state {
  padding: 48px 0;
  text-align: center;
  color: var(--color-text-muted);
  font-size: 13px;
}

.truncate {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
