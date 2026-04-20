import type { HealthStatus } from '@/types'

export async function checkHealth(baseUrl: string, healthPath: string): Promise<HealthStatus> {
  const start = performance.now()
  try {
    console.log(`${baseUrl}${healthPath}`)
    const res = await fetch(`${baseUrl}${healthPath}`, {
      method: 'GET',
      signal: AbortSignal.timeout(5000),
    })
    const latency = Math.round(performance.now() - start)
    return {
      state: res.ok ? 'healthy' : 'unhealthy',
      timestamp: Date.now(),
      latency,
      httpStatus: res.status,
    }
  } catch {
    return {
      state: 'unhealthy',
      timestamp: Date.now(),
      message: 'Сервис недоступен',
    }
  }
}
