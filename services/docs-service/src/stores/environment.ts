import { defineStore } from 'pinia'
import { useStorage } from '@vueuse/core'

export const useEnvironmentStore = defineStore('environment', () => {
  const current = useStorage<string>('docs-env', 'dev')

  function setCurrent(env: string) {
    current.value = env
  }

  return { current, setCurrent }
})
