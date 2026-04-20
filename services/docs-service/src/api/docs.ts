import type { ServiceIndex, ServiceMeta, HttpDocs, SocketIODocs, KafkaDocs, GrpcDocs, Protocol, SharedError, SharedPagination, SharedSchema } from '@/types'

const BASE = import.meta.env.BASE_URL + 'docs-data'

async function fetchJson<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`)
  if (!res.ok) throw new Error(`Не удалось загрузить ${path}: ${res.status}`)
  return res.json()
}

export function fetchServicesIndex(): Promise<ServiceIndex[]> {
  return fetchJson<ServiceIndex[]>('/services.index.json')
}

export function fetchServiceMeta(serviceId: string): Promise<ServiceMeta> {
  return fetchJson<ServiceMeta>(`/services/${serviceId}/service.json`)
}

const PROTOCOL_FILES: Record<Protocol, string> = {
  http: 'http.json',
  kafka: 'kafka.json',
  grpc: 'grpc.json',
  socketio: 'socketio.json',
}

type ProtocolDocsMap = {
  http: HttpDocs
  kafka: KafkaDocs
  grpc: GrpcDocs
  socketio: SocketIODocs
}

export function fetchProtocolDocs<P extends Protocol>(
  serviceId: string,
  protocol: P,
): Promise<ProtocolDocsMap[P]> {
  return fetchJson(`/services/${serviceId}/${PROTOCOL_FILES[protocol]}`)
}

export function fetchSharedErrors(): Promise<{ errors: Record<string, SharedError> }> {
  return fetchJson('/shared/errors.json')
}

export function fetchSharedPagination(): Promise<{ pagination: Record<string, SharedPagination> }> {
  return fetchJson('/shared/pagination.json')
}

export function fetchSharedSchemas(): Promise<{ schemas: Record<string, SharedSchema> }> {
  return fetchJson('/shared/schemas.json')
}
