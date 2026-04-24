<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useEnvironmentStore } from '@/stores/environment'
import { ENVIRONMENT_LABELS } from '@/constants'

const router = useRouter()
const authStore = useAuthStore()
const envStore = useEnvironmentStore()

const token = ref(authStore.currentToken())
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  if (!token.value.trim()) {
    error.value = 'Введите токен'
    return
  }

  loading.value = true
  error.value = ''

  authStore.setToken(envStore.current, token.value.trim())
  const ok = await authStore.validate()

  if (ok) {
    const redirect = (router.currentRoute.value.query.redirect as string) || '/'
    router.push(redirect)
  } else {
    error.value = 'Невалидный токен или сервис авторизации недоступен'
  }

  loading.value = false
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <svg class="login-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
          <path d="M7 11V7a5 5 0 0 1 10 0v4" />
        </svg>
        <h1 class="login-title">Авторизация</h1>
        <p class="login-subtitle">Для доступа к документации введите Bearer токен</p>
      </div>

      <div class="login-env">
        <span class="login-env-label">Окружение:</span>
        <strong>{{ ENVIRONMENT_LABELS[envStore.current] ?? envStore.current }}</strong>
      </div>

      <form class="login-form" @submit.prevent="handleLogin">
        <label class="field-label" for="token-input">Bearer Token</label>
        <textarea
          id="token-input"
          v-model="token"
          class="token-input"
          rows="3"
          placeholder="eyJhbGciOiJIUzI1NiIs..."
          autocomplete="off"
          spellcheck="false"
        />

        <div v-if="error" class="login-error">{{ error }}</div>

        <button class="btn-login" type="submit" :disabled="loading">
          {{ loading ? 'Проверка...' : 'Войти' }}
        </button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: calc(100vh - 56px);
  padding: 24px 16px;
}
.login-card {
  width: 100%;
  max-width: 420px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  padding: 32px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.login-header {
  text-align: center;
}
.login-icon {
  width: 32px;
  height: 32px;
  color: var(--color-accent);
  margin-bottom: 8px;
}
.login-title {
  font-size: 20px;
  font-weight: 700;
  margin: 0 0 4px;
}
.login-subtitle {
  font-size: 13px;
  color: var(--color-text-muted);
  margin: 0;
}
.login-env {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-secondary);
  padding: 8px 12px;
  border-radius: var(--radius-md);
  background: var(--color-bg-tertiary);
}
.login-env-label {
  color: var(--color-text-muted);
}
.login-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.field-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
}
.token-input {
  width: 100%;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  background: var(--color-bg);
  padding: 10px 12px;
  font-size: 12px;
  font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;
  color: var(--color-text);
  outline: none;
  resize: vertical;
}
.token-input:focus {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 40%, transparent);
}
.login-error {
  font-size: 12px;
  color: var(--color-error);
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  background: color-mix(in srgb, var(--color-error) 5%, transparent);
  border: 1px solid color-mix(in srgb, var(--color-error) 20%, transparent);
}
.btn-login {
  border-radius: var(--radius-md);
  background: var(--color-accent);
  color: #fff;
  padding: 10px 16px;
  font-size: 14px;
  font-weight: 500;
  border: none;
  cursor: pointer;
  transition: background-color 0.15s;
}
.btn-login:hover {
  background: var(--color-accent-hover);
}
.btn-login:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
