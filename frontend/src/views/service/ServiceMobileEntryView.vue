<template>
  <component :is="currentView" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import ServiceListView from './ServiceListView.vue'
import ServiceWorkspace from './ServiceWorkspace.vue'

const route = useRoute()
const currentView = computed(() => {
  if (typeof route.query.service === 'string' && route.query.service) {
    return ServiceWorkspace
  }
  return ServiceListView
})
</script>
