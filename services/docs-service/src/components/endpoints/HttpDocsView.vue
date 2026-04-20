<script setup lang="ts">
import { computed, ref } from 'vue'
import type { HttpDocs, HttpEndpoint } from '@/types'
import HttpMethodBadge from '@/components/common/HttpMethodBadge.vue'
import HttpEndpointAccordion from './HttpEndpointAccordion.vue'

const props = defineProps<{ docs: HttpDocs; serviceId: string }>()

const search = ref('')

const grouped = computed(() => {
  const map = new Map<string, HttpEndpoint[]>()
  for (const ep of props.docs.endpoints) {
    for (const tag of ep.tags.length ? ep.tags : ['Без тега']) {
      if (!map.has(tag)) map.set(tag, [])
      map.get(tag)!.push(ep)
    }
  }
  return map
})

const filteredGrouped = computed(() => {
  if (!search.value) return grouped.value
  const q = search.value.toLowerCase()
  const result = new Map<string, HttpEndpoint[]>()
  for (const [tag, endpoints] of grouped.value) {
    const filtered = endpoints.filter(
      ep => ep.path.toLowerCase().includes(q) || ep.summary.toLowerCase().includes(q) || ep.method.toLowerCase().includes(q),
    )
    if (filtered.length) result.set(tag, filtered)
  }
  return result
})

const collapsedGroups = ref(new Set<string>())

function toggleGroup(tag: string) {
  if (collapsedGroups.value.has(tag)) collapsedGroups.value.delete(tag)
  else collapsedGroups.value.add(tag)
}

function scrollToEndpoint(id: string) {
  document.getElementById('endpoint-' + id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
</script>

<template>
  <div class="http-layout">
    <!-- Sidebar -->
    <aside class="sidebar">
      <div class="sidebar-sticky">
        <input v-model="search" type="text" placeholder="Поиск endpoints..." class="search-input" />

        <nav class="sidebar-nav">
          <div v-for="[tag, endpoints] in filteredGrouped" :key="tag">
            <button class="group-toggle" @click="toggleGroup(tag)">
              {{ tag }}
              <svg class="group-chevron" :class="{ collapsed: collapsedGroups.has(tag) }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M6 9l6 6 6-6" />
              </svg>
            </button>
            <div v-if="!collapsedGroups.has(tag)" class="group-items">
              <button v-for="ep in endpoints" :key="ep.id" class="sidebar-item" @click="scrollToEndpoint(ep.id)">
                <HttpMethodBadge :method="ep.method" />
                <span class="sidebar-path truncate">{{ ep.path }}</span>
              </button>
            </div>
          </div>
        </nav>
      </div>
    </aside>

    <!-- Main content -->
    <div class="main-content">
      <input v-model="search" type="text" placeholder="Поиск endpoints..." class="search-input mobile-search" />

      <template v-for="[tag, endpoints] in filteredGrouped" :key="tag">
        <h3 class="group-title">{{ tag }}</h3>
        <HttpEndpointAccordion v-for="ep in endpoints" :key="ep.id" :endpoint="ep" :service-id="serviceId" />
      </template>

      <div v-if="filteredGrouped.size === 0" class="empty-state">
        Endpoints не найдены
      </div>
    </div>
  </div>
</template>

<style scoped>
.http-layout {
  display: flex;
  gap: 24px;
}
.sidebar {
  width: 288px;
  flex-shrink: 0;
}
@media (max-width: 1024px) {
  .sidebar {
    display: none;
  }
}
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
.search-input::placeholder {
  color: var(--color-text-muted);
}
.search-input:focus {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 40%, transparent);
}
.mobile-search {
  display: none;
  margin-bottom: 16px;
}
@media (max-width: 1024px) {
  .mobile-search {
    display: block;
  }
}
.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.group-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  background: none;
  border: none;
  cursor: pointer;
}
.group-toggle:hover {
  color: var(--color-text-secondary);
}
.group-chevron {
  width: 12px;
  height: 12px;
  transition: transform 0.2s;
}
.group-chevron.collapsed {
  transform: rotate(-90deg);
}
.group-items {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
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
.sidebar-item:hover {
  background: var(--color-bg-tertiary);
}
.sidebar-path {
  font-size: 12px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  color: var(--color-text-secondary);
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
</style>
