import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { SharedError, SharedPagination, SharedSchema, SchemaField } from '@/types'
import { fetchSharedErrors, fetchSharedPagination, fetchSharedSchemas } from '@/api/docs'

export const useSharedStore = defineStore('shared', () => {
  const errors = ref<Record<string, SharedError>>({})
  const pagination = ref<Record<string, SharedPagination>>({})
  const schemas = ref<Record<string, SharedSchema>>({})
  const loaded = ref(false)

  async function load() {
    if (loaded.value) return
    const [errData, pagData, schemaData] = await Promise.all([
      fetchSharedErrors(),
      fetchSharedPagination(),
      fetchSharedSchemas(),
    ])
    errors.value = errData.errors
    pagination.value = pagData.pagination
    schemas.value = schemaData.schemas
    loaded.value = true
  }

  function getError(id: string): SharedError | undefined {
    return errors.value[id]
  }

  function getPagination(id: string): SharedPagination | undefined {
    return pagination.value[id]
  }

  function getSchema(id: string): SharedSchema | undefined {
    return schemas.value[id]
  }

  function resolveSchemaRef(ref: string): SchemaField[] | undefined {
    return schemas.value[ref]?.fields
  }

  return { errors, pagination, schemas, loaded, load, getError, getPagination, getSchema, resolveSchemaRef }
})
