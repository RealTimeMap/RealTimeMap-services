<script setup lang="ts">
import type { SchemaField } from '@/types'

defineProps<{ fields: SchemaField[]; depth?: number }>()
</script>

<template>
  <div class="schema">
    <div
      v-for="field in fields"
      :key="field.name"
      class="schema-field"
      :style="{ paddingLeft: (depth ?? 0) * 16 + 'px' }"
    >
      <div class="field-row">
        <div class="field-name-wrap">
          <code class="field-name">{{ field.name }}</code>
          <span v-if="field.required" class="required">*</span>
        </div>
        <span class="field-type">{{ field.type }}</span>
        <span v-if="field.enum" class="field-enum">enum: {{ field.enum.join(' | ') }}</span>
        <span v-if="field.description" class="field-desc">{{ field.description }}</span>
      </div>
      <SchemaViewer v-if="field.children?.length" :fields="field.children" :depth="(depth ?? 0) + 1" />
    </div>
  </div>
</template>

<style scoped>
.schema {
  font-size: 13px;
}
.schema-field {
  border-bottom: 1px solid color-mix(in srgb, var(--color-border) 30%, transparent);
}
.schema-field:last-child {
  border-bottom: none;
}
.field-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 8px 0;
}
.field-name-wrap {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.field-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text);
}
.required {
  color: var(--color-error);
  font-size: 12px;
}
.field-type {
  font-size: 12px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  color: var(--color-text-muted);
}
.field-enum {
  font-size: 12px;
  color: var(--color-text-muted);
}
.field-desc {
  font-size: 12px;
  color: var(--color-text-secondary);
  flex: 1;
}
</style>
