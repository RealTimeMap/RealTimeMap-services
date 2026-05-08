export type Protocol = 'http' | 'kafka' | 'grpc' | 'socketio'

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export type HealthState = 'healthy' | 'unhealthy' | 'unknown' | 'checking'

export type ParameterLocation = 'path' | 'query' | 'header'

export type GrpcMethodType = 'unary' | 'server-streaming' | 'client-streaming' | 'bidirectional'

export type SocketIODirection = 'client-to-server' | 'server-to-client'

export interface Environment {
  name: string
  url: string
}

export interface ServiceIndex {
  id: string
  name: string
  description: string
  protocols: Protocol[]
  availableEnvs: string[]
  healthPath: string
  team?: string
  repository?: string
}

export interface ServiceMeta {
  id: string
  name: string
  description: string
  fullDescription?: string
  protocols: Protocol[]
  availableEnvs: string[]
  healthPath: string
  team?: string
  repository?: string
}

export interface SchemaField {
  name: string
  type: string
  required?: boolean
  description?: string
  example?: unknown
  children?: SchemaField[]
  enum?: string[]
}

export interface Parameter {
  name: string
  type: string
  required: boolean
  description: string
  example?: string
  location: ParameterLocation
}

export interface HttpResponse {
  statusCode: number
  description: string
  schema?: SchemaField[]
  schemaRef?: string
  example?: unknown
}

export type ContentType = 'json' | 'form-data' | 'x-www-form-urlencoded' | 'text' | 'binary'

export interface HttpEndpoint {
  id: string
  method: HttpMethod
  path: string
  summary: string
  description?: string
  tags: string[]
  auth?: boolean
  parameters?: Parameter[]
  requestBody?: {
    description?: string
    contentType?: ContentType
    schema: SchemaField[]
    example?: unknown
  }
  responses: HttpResponse[]
  errors?: string[]
  pagination?: string
}

export interface HttpDocs {
  endpoints: HttpEndpoint[]
}

export interface SocketIOAckBranch {
  name: string
  description?: string
  schema: SchemaField[]
  example?: unknown
}

export interface SocketIOEvent {
  id: string
  name: string
  direction: SocketIODirection
  namespace?: string
  room?: string
  auth?: boolean
  summary: string
  description?: string
  tags?: string[]
  payload?: {
    description?: string
    schema: SchemaField[]
    example?: unknown
  }
  ack?: {
    description?: string
    schema?: SchemaField[]
    example?: unknown
    branches?: SocketIOAckBranch[]
  }
  errors?: string[]
  codeExample?: string
}

export interface SocketIONamespaceInfo {
  path: string
  description?: string
  auth?: boolean
}

export interface SocketIODocs {
  baseUrl?: string
  transport?: ('websocket' | 'polling')[]
  namespaces?: SocketIONamespaceInfo[]
  events: SocketIOEvent[]
}

export interface KafkaTopic {
  id: string
  name: string
  summary: string
  description?: string
  producers?: string[]
  consumers?: string[]
  partitionKey?: string
  retention?: string
  schema: SchemaField[]
  example?: unknown
}

export interface KafkaDocs {
  topics: KafkaTopic[]
}

export interface GrpcMethod {
  id: string
  service: string
  method: string
  type: GrpcMethodType
  summary: string
  description?: string
  request: {
    schema: SchemaField[]
    example?: unknown
  }
  response: {
    schema: SchemaField[]
    example?: unknown
  }
  statusCodes?: { code: string; description: string }[]
  protoExample?: string
  supportsGrpcWeb?: boolean
}

export interface GrpcDocs {
  methods: GrpcMethod[]
}

export interface HealthStatus {
  state: HealthState
  timestamp: number
  latency?: number
  httpStatus?: number
  message?: string
}

// --- Shared definitions ---

export interface SharedError {
  id: string
  statusCode: number
  description: string
  schema: SchemaField[]
  example: unknown
}

export interface SharedPagination {
  id: string
  name: string
  description: string
  wrapperSchema: SchemaField[]
  queryParams: Parameter[]
  example: unknown
}

export interface SharedSchema {
  id: string
  name: string
  description: string
  fields: SchemaField[]
}

export interface SharedData {
  errors: Record<string, SharedError>
  pagination: Record<string, SharedPagination>
  schemas: Record<string, SharedSchema>
}
