<template>
  <section class="agent-management">
    <nav class="agent-management__tabs" aria-label="智能体管理">
      <button
        type="button"
        :class="{ 'is-active': activeTab === 'agents' }"
        @click="setActiveTab('agents')"
      >
        <t-icon name="control-platform" />
        <span>智能体</span>
      </button>
      <button
        type="button"
        :class="{ 'is-active': activeTab === 'import' }"
        @click="setActiveTab('import')"
      >
        <t-icon name="upload" />
        <span>导入专家</span>
      </button>
    </nav>

    <div class="agent-management__body">
      <AgentList v-if="activeTab === 'agents'" />
      <AdminExpertPackages v-else embedded />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AgentList from '@/views/agent/AgentList.vue'

type AgentManagementTab = 'agents' | 'import'

const route = useRoute()
const router = useRouter()
const AdminExpertPackages = defineAsyncComponent(
  () => import('@admin/views/AdminExpertPackages.vue'),
)

const activeTab = computed<AgentManagementTab>(() => (
  route.query.tab === 'import' ? 'import' : 'agents'
))

function setActiveTab(tab: AgentManagementTab) {
  if (activeTab.value === tab) return
  const query = { ...route.query }
  delete query.edit
  delete query.section
  if (tab === 'agents') {
    delete query.tab
  } else {
    query.tab = tab
  }
  void router.replace({ name: 'adminAgents', query })
}
</script>

<style scoped>
.agent-management {
  display: grid;
  min-width: 0;
  gap: 18px;
}

.agent-management__tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  border-bottom: 1px solid var(--admin-border);
}

.agent-management__tabs button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 42px;
  padding: 0 14px;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--admin-text-secondary);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

.agent-management__tabs button:hover {
  color: var(--admin-text);
  background: var(--admin-surface-soft);
}

.agent-management__tabs button.is-active {
  border-bottom-color: var(--td-brand-color);
  color: var(--td-brand-color);
  font-weight: 600;
}

.agent-management__body {
  min-width: 0;
}
</style>
