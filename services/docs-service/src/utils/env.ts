/**
 * Получает URL окружения сервиса из переменных окружения Vite.
 * Формат: VITE_SVC_<SERVICE_ID>_<ENV>
 * Пример: VITE_SVC_AUTH_SERVICE_DEV
 */
export function getServiceUrl(serviceId: string, env: string): string {
  const key = `VITE_SVC_${serviceId.replace(/-/g, '_').toUpperCase()}_${env.toUpperCase()}`
  return (import.meta.env[key] as string) ?? ''
}
