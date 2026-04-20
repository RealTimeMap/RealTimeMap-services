import { useStorage } from '@vueuse/core'
import { watchEffect } from 'vue'

export function useTheme() {
  const isDark = useStorage('docs-theme-dark', false)

  watchEffect(() => {
    document.documentElement.classList.toggle('dark', isDark.value)
  })

  function toggle() {
    isDark.value = !isDark.value
  }

  return { isDark, toggle }
}
