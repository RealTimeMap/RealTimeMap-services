import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ServiceIndex } from '@/types'
import { fetchServicesIndex } from '@/api/docs'

export const useServicesStore = defineStore('services', () => {
  const services = ref<ServiceIndex[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function load() {
    if (services.value.length > 0) return
    loading.value = true
    error.value = null
    try {
      services.value = await fetchServicesIndex()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
    } finally {
      loading.value = false
    }
  }

  return { services, loading, error, load }
})
