<script setup lang="ts">
import { useEnvironmentStore } from '@/stores/environment'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'
import { ENVIRONMENT_LABELS } from '@/constants'

const envStore = useEnvironmentStore()
const authStore = useAuthStore()
const { isDark, toggle: toggleTheme } = useTheme()

const environments = Object.entries(ENVIRONMENT_LABELS)

function handleLogout() {
  authStore.logout()
  window.location.reload()
}
</script>

<template>
  <header class="header">
    <div class="header-inner">
      <router-link to="/" class="logo">
        <svg class="logo-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
          <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
        </svg>
        Документация RTM сервисов
      </router-link>

      <div class="header-actions">
        <select
          :value="envStore.current"
          class="env-select"
          @change="envStore.setCurrent(($event.target as HTMLSelectElement).value)"
        >
          <option v-for="[key, label] in environments" :key="key" :value="key">{{ label }}</option>
        </select>

        <button class="theme-btn" :title="isDark ? 'Светлая тема' : 'Тёмная тема'" @click="toggleTheme">
          <svg v-if="isDark" class="theme-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="5" />
            <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
          </svg>
          <svg v-else class="theme-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
          </svg>
        </button>

        <template v-if="authStore.isAuthenticated">
          <span class="user-info" :title="'ID: ' + (authStore.userInfo?.userId ?? '')">
            <svg class="user-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
              <circle cx="12" cy="7" r="4" />
            </svg>
            {{ authStore.userInfo?.userName || 'Пользователь' }}
          </span>
          <button class="logout-btn" title="Выйти" @click="handleLogout">
            <svg class="logout-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
              <polyline points="16 17 21 12 16 7" />
              <line x1="21" y1="12" x2="9" y2="12" />
            </svg>
          </button>
        </template>
      </div>
    </div>
  </header>
</template>

<style scoped>
.header {
  position: sticky;
  top: 0;
  z-index: 50;
  border-bottom: 1px solid var(--color-border);
  background-color: color-mix(in srgb, var(--color-bg) 80%, transparent);
  backdrop-filter: blur(8px);
}
.header-inner {
  max-width: 1280px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  height: 56px;
}
.logo {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text);
  text-decoration: none;
}
.logo-icon {
  width: 24px;
  height: 24px;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.env-select {
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  padding: 6px 12px;
  font-size: 13px;
  color: var(--color-text);
  outline: none;
}
.env-select:focus {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 40%, transparent);
}
.theme-btn {
  border-radius: var(--radius-md);
  padding: 8px;
  background: none;
  border: none;
  color: var(--color-text);
  cursor: pointer;
  transition: background-color 0.15s;
}
.theme-btn:hover {
  background-color: var(--color-bg-tertiary);
}
.theme-icon {
  width: 20px;
  height: 20px;
}
.user-info {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--color-text-secondary);
  padding: 4px 10px;
  border-radius: var(--radius-md);
  background: var(--color-bg-tertiary);
}
.user-icon {
  width: 16px;
  height: 16px;
}
.logout-btn {
  border-radius: var(--radius-md);
  padding: 8px;
  background: none;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: color 0.15s, background-color 0.15s;
}
.logout-btn:hover {
  color: var(--color-error);
  background-color: color-mix(in srgb, var(--color-error) 10%, transparent);
}
.logout-icon {
  width: 18px;
  height: 18px;
}

@media screen and (max-width: 550px) {
  .logo {
    display: none;
  }
}
</style>
