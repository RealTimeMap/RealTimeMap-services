import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { HealthStatus } from '@/types'
import { checkHealth } from '@/api/health'
import { getServiceUrl } from '@/utils/env'
import { useServicesStore } from './services'
import { useEnvironmentStore } from './environment'

export const useHealthStore = defineStore('health', () => {
  const statuses = ref<Record<string, HealthStatus>>({})

  async function checkService(serviceId: string) {
    const servicesStore = useServicesStore()
    const envStore = useEnvironmentStore()
    const service = servicesStore.services.find(s => s.id === serviceId)
    if (!service) return

    const baseUrl = getServiceUrl(serviceId, envStore.current)
    if (!baseUrl) {
      statuses.value[serviceId] = { state: 'unknown', timestamp: Date.now(), message: 'Окружение не настроено' }
      return
    }

    statuses.value[serviceId] = { state: 'checking', timestamp: Date.now() }
    statuses.value[serviceId] = await checkHealth(baseUrl, service.healthPath)
  }

  async function checkAll() {
    const servicesStore = useServicesStore()
    await Promise.allSettled(servicesStore.services.map(s => checkService(s.id)))
  }

  return { statuses, checkService, checkAll }
})
