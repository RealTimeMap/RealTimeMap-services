<script setup lang="ts">
import { onMounted } from 'vue'
import AppHeader from '@/components/common/AppHeader.vue'
import { useServicesStore } from '@/stores/services'
import { useSharedStore } from '@/stores/shared'

const servicesStore = useServicesStore()
const sharedStore = useSharedStore()
onMounted(() => {
  servicesStore.load()
  sharedStore.load()
})
</script>

<template>
  <div class="app-layout">
    <AppHeader />
    <main class="app-main">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.app-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.app-main {
  flex: 1;
}
</style>
