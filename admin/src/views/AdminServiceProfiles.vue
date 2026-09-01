<template>
  <section class="service-config-page">
    <div class="service-config-page__header">
      <div>
        <h2>服务配置</h2>
        <p>服务项继续在这里展示；员工分身在成员管理维护，AI 根据分身描述匹配可用服务能力。</p>
      </div>
    </div>

    <div class="service-config-summary">
      <article>
        <span>服务项</span>
        <strong>{{ serviceItems.length }}</strong>
        <em>保留展示</em>
      </article>
      <article>
        <span>分身来源</span>
        <strong>成员管理</strong>
        <em>必填描述</em>
      </article>
      <article>
        <span>启用方式</span>
        <strong>AI 匹配</strong>
        <em>按岗位职责判断</em>
      </article>
    </div>

    <section class="service-config-panel">
      <div class="panel-title">
        <span>
          <strong>服务项</strong>
          <em>这些服务能力保留在服务配置中查看；具体成员能力由员工分身描述驱动。</em>
        </span>
        <t-tag theme="warning" variant="light">分身不在此页维护</t-tag>
      </div>

      <t-alert
        v-if="loadFailed"
        theme="warning"
        message="服务项接口暂不可用，当前展示内置服务项。"
        class="service-config-alert"
      />

      <div class="service-item-grid">
        <article v-for="item in serviceItems" :key="item.agent_domain" class="service-item-card">
          <div class="service-item-card__head">
            <span class="service-item-card__icon">
              <t-icon :name="serviceIcon(item.agent_domain)" />
            </span>
            <span>
              <strong>{{ item.display_name }}</strong>
              <em>{{ serviceCategory(item.agent_domain) }}</em>
            </span>
            <t-tag :theme="item.default_enabled ? 'success' : 'primary'" variant="light">
              {{ item.default_enabled ? '基础项' : '按分身启用' }}
            </t-tag>
          </div>

          <p>{{ item.description }}</p>

          <dl>
            <div>
              <dt>输入来源</dt>
              <dd>{{ serviceInputSource(item.agent_domain) }}</dd>
            </div>
            <div>
              <dt>输出结果</dt>
              <dd>{{ serviceOutput(item.agent_domain) }}</dd>
            </div>
            <div>
              <dt>成员分身</dt>
              <dd>在成员管理中配置</dd>
            </div>
          </dl>
        </article>
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { listServiceAgentTemplates, type ServiceAgentTemplate } from '@/api/service'

type ServiceItem = ServiceAgentTemplate

const loading = ref(false)
const loadFailed = ref(false)
const remoteItems = ref<ServiceItem[]>([])

const fallbackItems: ServiceItem[] = [
  {
    agent_domain: 'memory_router',
    display_name: '记忆识别',
    description: '识别记忆属于哪个服务场景，作为后续服务能力匹配的基础。',
    default_enabled: true,
    user_visible: false,
    work_doc_directory: '路由/',
  },
  {
    agent_domain: 'lead_intake',
    display_name: '线索录入',
    description: '从咨询、试听、报名意向记忆中整理线索草稿和缺失信息。',
    default_enabled: false,
    user_visible: false,
    work_doc_directory: '线索/',
  },
  {
    agent_domain: 'sales_consulting',
    display_name: '招生咨询',
    description: '生成异议处理、邀约话术、试听后跟进和下一步建议。',
    default_enabled: false,
    user_visible: false,
    work_doc_directory: '线索/',
  },
  {
    agent_domain: 'customer_service',
    display_name: '客户服务',
    description: '整理客户摘要、跟进记录、续费窗口和服务闭环事项。',
    default_enabled: false,
    user_visible: false,
    work_doc_directory: '客户/',
  },
  {
    agent_domain: 'schedule_coordination',
    display_name: '排课调课',
    description: '识别请假、补课、排课和调课信号，生成待确认安排。',
    default_enabled: false,
    user_visible: false,
    work_doc_directory: '排课/',
  },
  {
    agent_domain: 'after_sale_risk',
    display_name: '售后风险',
    description: '识别投诉、不满、退款、退费等高风险服务信号，推动处理闭环。',
    default_enabled: false,
    user_visible: false,
    work_doc_directory: '售后风险/',
  },
  {
    agent_domain: 'daily_review',
    display_name: '日报复盘',
    description: '按用户触发汇总服务提醒、风险、动作闭环和知识补齐建议。',
    default_enabled: false,
    user_visible: false,
    work_doc_directory: '日报/',
  },
]

const serviceItems = computed<ServiceItem[]>(() => {
  const merged = new Map(fallbackItems.map((item) => [item.agent_domain, item]))
  remoteItems.value.forEach((item) => {
    merged.set(item.agent_domain, {
      ...merged.get(item.agent_domain),
      ...item,
    })
  })
  return fallbackItems
    .map((item) => merged.get(item.agent_domain))
    .filter((item): item is ServiceItem => Boolean(item))
})

async function loadServiceItems() {
  if (loading.value) return
  loading.value = true
  loadFailed.value = false
  try {
    const response = await listServiceAgentTemplates()
    remoteItems.value = response?.data || []
  } catch (error) {
    console.warn('[AdminServiceProfiles] Failed to load service agent templates:', error)
    remoteItems.value = []
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

function serviceIcon(domain: string) {
  const icons: Record<string, string> = {
    memory_router: 'setting-1',
    lead_intake: 'user-add',
    sales_consulting: 'chat',
    customer_service: 'service',
    schedule_coordination: 'calendar',
    after_sale_risk: 'error-circle',
    daily_review: 'file',
  }
  return icons[domain] || 'setting-1'
}

function serviceCategory(domain: string) {
  const categories: Record<string, string> = {
    memory_router: '基础识别',
    lead_intake: '招生前置',
    sales_consulting: '招生沟通',
    customer_service: '服务跟进',
    schedule_coordination: '教务协同',
    after_sale_risk: '风险闭环',
    daily_review: '经营复盘',
  }
  return categories[domain] || '服务能力'
}

function serviceInputSource(domain: string) {
  const sources: Record<string, string> = {
    memory_router: '成员分身、记忆内容',
    daily_review: '服务提醒、处理状态',
  }
  return sources[domain] || '成员分身、客户服务记忆'
}

function serviceOutput(domain: string) {
  const outputs: Record<string, string> = {
    memory_router: '服务场景和能力匹配结果',
    lead_intake: '线索草稿和缺失信息',
    sales_consulting: '沟通话术和下一步建议',
    customer_service: '客户摘要和跟进事项',
    schedule_coordination: '待确认排课安排',
    after_sale_risk: '风险处理建议和闭环事项',
    daily_review: '服务日报和知识补齐建议',
  }
  return outputs[domain] || '服务提醒和处理建议'
}

onMounted(() => {
  void loadServiceItems()
})
</script>

<style scoped>
.service-config-page {
  display: grid;
  gap: 16px;
}

.service-config-page__header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.service-config-page__header h2 {
  margin: 0 0 6px;
  font-size: 22px;
  line-height: 1.35;
  color: var(--td-text-color-primary);
}

.service-config-page__header p {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 14px;
  line-height: 1.7;
}

.service-config-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.service-config-summary article {
  display: grid;
  gap: 4px;
  min-width: 0;
  padding: 14px 16px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.service-config-summary span,
.service-config-summary em {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.4;
  font-style: normal;
}

.service-config-summary strong {
  color: var(--td-text-color-primary);
  font-size: 18px;
  line-height: 1.35;
}

.service-config-panel {
  min-width: 0;
  padding: 18px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.panel-title {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
  margin-bottom: 14px;
}

.panel-title span {
  display: grid;
  gap: 4px;
}

.panel-title strong {
  color: var(--td-text-color-primary);
  font-size: 16px;
  line-height: 1.4;
}

.panel-title em {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
  font-style: normal;
}

.service-config-alert {
  margin-bottom: 14px;
}

.service-item-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.service-item-card {
  display: grid;
  gap: 14px;
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 8px;
  background: var(--td-bg-color-page);
}

.service-item-card__head {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
}

.service-item-card__icon {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border-radius: 8px;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
}

.service-item-card__head span:not(.service-item-card__icon) {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.service-item-card__head strong {
  color: var(--td-text-color-primary);
  font-size: 15px;
  line-height: 1.35;
}

.service-item-card__head em {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.35;
  font-style: normal;
}

.service-item-card p {
  min-height: 42px;
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.65;
}

.service-item-card dl {
  display: grid;
  gap: 8px;
  margin: 0;
}

.service-item-card dl div {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 10px;
  align-items: baseline;
}

.service-item-card dt,
.service-item-card dd {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
}

.service-item-card dt {
  color: var(--td-text-color-placeholder);
}

.service-item-card dd {
  color: var(--td-text-color-primary);
}

@media (max-width: 900px) {
  .service-config-page__header {
    display: grid;
  }

  .service-config-summary,
  .service-item-grid {
    grid-template-columns: 1fr;
  }
}
</style>
