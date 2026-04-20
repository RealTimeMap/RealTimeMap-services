<script setup lang="ts">
import type { Parameter } from '@/types'

defineProps<{ parameters: Parameter[] }>()
</script>

<template>
  <div class="table-wrap">
    <table class="param-table">
      <thead>
        <tr>
          <th>Параметр</th>
          <th>Тип</th>
          <th>В</th>
          <th>Описание</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="param in parameters" :key="param.name">
          <td>
            <code class="param-name">{{ param.name }}</code>
            <span v-if="param.required" class="required">*</span>
          </td>
          <td class="param-type">{{ param.type }}</td>
          <td><span class="location-badge">{{ param.location }}</span></td>
          <td class="param-desc">
            {{ param.description }}
            <span v-if="param.example" class="param-example">
              Пример: <code>{{ param.example }}</code>
            </span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.table-wrap {
  overflow-x: auto;
}
.param-table {
  width: 100%;
  font-size: 13px;
  border-collapse: collapse;
}
.param-table th {
  text-align: left;
  padding: 8px 16px 8px 0;
  font-size: 11px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  border-bottom: 1px solid var(--color-border);
}
.param-table td {
  padding: 10px 16px 10px 0;
  border-bottom: 1px solid color-mix(in srgb, var(--color-border) 50%, transparent);
}
.param-name {
  font-size: 12px;
  background: var(--color-bg-tertiary);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}
.required {
  color: var(--color-error);
  font-size: 12px;
  margin-left: 4px;
}
.param-type {
  font-size: 12px;
  color: var(--color-text-muted);
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
}
.location-badge {
  font-size: 11px;
  border-radius: var(--radius-full);
  background: var(--color-bg-tertiary);
  padding: 2px 8px;
  color: var(--color-text-secondary);
}
.param-desc {
  color: var(--color-text-secondary);
}
.param-example {
  margin-left: 8px;
  font-size: 12px;
  color: var(--color-text-muted);
}
.param-example code {
  font-size: 11px;
}
</style>
