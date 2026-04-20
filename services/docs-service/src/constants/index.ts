import type { Protocol, HttpMethod, HealthState } from '@/types'
import type { CSSProperties } from 'vue'

export const PROTOCOL_LABELS: Record<Protocol, string> = {
  http: 'HTTP/REST',
  kafka: 'Kafka',
  grpc: 'gRPC',
  socketio: 'Socket.IO',
}

export const PROTOCOL_COLORS: Record<Protocol, string> = {
  http: 'var(--color-http)',
  kafka: 'var(--color-kafka)',
  grpc: 'var(--color-grpc)',
  socketio: 'var(--color-socketio)',
}

export function protocolBadgeStyle(protocol: Protocol): CSSProperties {
  const color = PROTOCOL_COLORS[protocol]
  return {
    color,
    backgroundColor: `color-mix(in srgb, ${color} 10%, transparent)`,
    borderColor: `color-mix(in srgb, ${color} 20%, transparent)`,
  }
}

export const HTTP_METHOD_COLORS: Record<HttpMethod, string> = {
  GET: 'var(--color-get)',
  POST: 'var(--color-post)',
  PUT: 'var(--color-put)',
  PATCH: 'var(--color-patch)',
  DELETE: 'var(--color-delete)',
}

export function httpMethodBadgeStyle(method: HttpMethod): CSSProperties {
  const color = HTTP_METHOD_COLORS[method]
  return {
    color,
    backgroundColor: `color-mix(in srgb, ${color} 10%, transparent)`,
    borderColor: `color-mix(in srgb, ${color} 20%, transparent)`,
  }
}

export const HEALTH_LABELS: Record<HealthState, string> = {
  healthy: 'Доступен',
  unhealthy: 'Недоступен',
  unknown: 'Неизвестно',
  checking: 'Проверка...',
}

export const HEALTH_DOT_COLORS: Record<HealthState, string> = {
  healthy: 'var(--color-healthy)',
  unhealthy: 'var(--color-unhealthy)',
  unknown: 'var(--color-unknown)',
  checking: 'var(--color-unknown)',
}

export const HEALTH_CHECK_INTERVAL = 25000

export const ENVIRONMENT_LABELS: Record<string, string> = {
  dev: 'Development',
  staging: 'Staging',
  prod: 'Production',
}
