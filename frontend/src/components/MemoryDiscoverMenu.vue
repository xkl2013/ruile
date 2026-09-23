<template>
  <section class="memory-discover-menu" :class="{ 'memory-discover-menu--collapsed': collapsed }">
    <template v-for="item in items" :key="item.key">
      <t-tooltip v-if="collapsed" :content="item.label" placement="right">
        <button
          type="button"
          class="memory-discover-menu-item"
          :class="{ active: isActive(item.key) }"
          :aria-current="isActive(item.key) ? 'page' : undefined"
          @click="openRoute(item.path)"
        >
          <t-icon :name="item.icon" class="memory-discover-menu-item-icon" />
        </button>
      </t-tooltip>

      <button
        v-else
        type="button"
        class="memory-discover-menu-item"
        :class="{ active: isActive(item.key) }"
        :title="item.label"
        :aria-current="isActive(item.key) ? 'page' : undefined"
        @click="openRoute(item.path)"
      >
        <t-icon :name="item.icon" class="memory-discover-menu-item-icon" />
        <span class="memory-discover-menu-item-name">{{ item.label }}</span>
      </button>
    </template>
  </section>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import {
  ORGANIZE_MENU_ROUTES,
  ORGANIZE_MEMORY_ASSET_ROUTES,
  type OrganizeTab,
} from '@/views/organize/organizeRoutes'

withDefaults(defineProps<{ collapsed?: boolean }>(), {
  collapsed: false,
})

const route = useRoute()
const router = useRouter()

const items = ORGANIZE_MENU_ROUTES.filter(
  (item) => item.key === 'memory' || item.key === 'discover',
)

const isActive = (key: OrganizeTab) => {
  if (key === 'memory') {
    return ORGANIZE_MEMORY_ASSET_ROUTES.some(
      (item) => item.routeName === String(route.name || ''),
    ) || route.name === 'organizeMemory'
  }
  return route.name === 'organizeDiscover'
}

const openRoute = async (path: string) => {
  if (route.path === path) return
  await router.push(path)
}
</script>

<style scoped lang="less">
.memory-discover-menu {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 1px 0 4px;
}

.memory-discover-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 30px;
  padding: 0 12px 0 calc(var(--sidebar-inset-x) + 12px);
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  text-align: left;
  transition: background-color 0.18s ease, color 0.18s ease;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }

  &.active {
    background: var(--td-bg-color-secondarycontainer);
  }
}

.memory-discover-menu-item-icon {
  flex-shrink: 0;
  color: var(--td-text-color-secondary);
}

.memory-discover-menu-item.active .memory-discover-menu-item-icon {
  color: var(--td-brand-color);
}

.memory-discover-menu-item-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: inherit;
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.memory-discover-menu--collapsed {
  gap: 2px;
  padding: 1px 0 4px;

  .memory-discover-menu-item {
    justify-content: center;
    min-height: 38px;
    padding: 8px 10px;
    border-radius: 4px;
  }
}
</style>
