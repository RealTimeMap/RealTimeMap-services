<script setup lang="ts">
import { ref } from 'vue'

defineProps<{ code: string; language?: string }>()

const copied = ref(false)

async function copy(code: string) {
  await navigator.clipboard.writeText(code)
  copied.value = true
  setTimeout(() => { copied.value = false }, 1500)
}
</script>

<template>
  <div class="code-block">
    <button class="copy-btn" @click="copy(code)">
      {{ copied ? 'Скопировано!' : 'Копировать' }}
    </button>
    <pre class="code-pre"><code>{{ code }}</code></pre>
  </div>
</template>

<style scoped>
.code-block {
  position: relative;
  border-radius: var(--radius-md);
  background: var(--color-bg-tertiary);
  border: 1px solid var(--color-border);
}
.code-block:hover .copy-btn {
  opacity: 1;
}
.copy-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  border-radius: var(--radius-md);
  padding: 4px 8px;
  font-size: 11px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  color: var(--color-text-secondary);
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s;
}
.copy-btn:hover {
  background: var(--color-border);
}
.code-pre {
  padding: 16px;
  overflow-x: auto;
  font-size: 12px;
  line-height: 1.6;
  margin: 0;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
}
</style>
