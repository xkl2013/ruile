<template>
  <div class="service-hub-page">
    <ServiceCreateDialog
      v-model:visible="createDialogVisible"
      :source="createSource"
      @submit="submitForm"
    />

    <t-dialog
      v-model:visible="blueprintConfirmVisible"
      header="确认服务空间蓝图"
      width="620px"
      :confirm-btn="{
        content: '确认并创建',
        theme: 'primary',
        loading: blueprintConfirming,
      }"
      :cancel-btn="{ content: '返回修改' }"
      :close-on-overlay-click="!blueprintConfirming"
      :close-btn="!blueprintConfirming"
      @confirm="confirmPendingBlueprint"
      @cancel="cancelPendingBlueprint"
      @close="cancelPendingBlueprint"
    >
      <div v-if="pendingBlueprint" class="service-blueprint-confirm">
        <div class="service-blueprint-confirm-head">
          <div>
            <span class="service-blueprint-confirm-kicker">指令已解析</span>
            <strong>{{ pendingPayload?.name }}</strong>
          </div>
          <span class="service-blueprint-confirm-status">待确认</span>
        </div>
        <p class="service-blueprint-confirm-copy">
          系统将按下面的空间形态、服务主体、档案字段和首页摘要初始化服务。确认后才会启用空间。
        </p>
        <div class="service-blueprint-confirm-grid">
          <div>
            <span>空间形态</span>
            <strong>{{ spaceTypeLabel(pendingBlueprint.proposed_space_type) }}</strong>
          </div>
          <div>
            <span>服务主体</span>
            <strong>{{ subjectPolicyLabel(pendingBlueprint) }}</strong>
          </div>
          <div>
            <span>档案字段</span>
            <strong>{{ pendingBlueprint.profile_schema.length }} 个</strong>
          </div>
          <div>
            <span>首页摘要</span>
            <strong>{{ pendingBlueprint.summary_schema.length }} 个模块</strong>
          </div>
        </div>
        <section class="service-blueprint-confirm-section">
          <span>档案字段</span>
          <div class="service-blueprint-confirm-tags">
            <em v-for="field in pendingBlueprint.profile_schema" :key="field.key">{{ field.label }}</em>
          </div>
        </section>
        <section class="service-blueprint-confirm-section">
          <span>首页摘要</span>
          <div class="service-blueprint-confirm-tags">
            <em v-for="section in pendingBlueprint.summary_schema" :key="section.key">{{ section.label }}</em>
          </div>
        </section>
      </div>
    </t-dialog>

    <t-dialog
      v-model:visible="reminderDetailVisible"
      header="事项协作"
      width="680px"
      :confirm-btn="null"
      :cancel-btn="null"
    >
      <div v-if="selectedReminder" class="service-reminder-collaboration">
        <div class="service-reminder-collaboration-head">
          <div>
            <strong>{{ selectedReminder.title }}</strong>
            <span>{{ reminderStatusLabel(selectedReminder.status) }} · {{ selectedReminder.depth || 0 }} 层</span>
          </div>
          <span v-if="selectedReminder.parent_reminder_id">已有父事项</span>
        </div>
        <div v-if="reminderCollaborationLoading" class="service-hub-panel-empty">正在加载协作信息。</div>
        <template v-else>
          <section class="service-reminder-collaboration-section">
            <div class="service-reminder-collaboration-section-head">
              <strong>负责人</strong>
              <button type="button" @click="saveReminderAssignees">保存</button>
            </div>
            <t-input v-model="reminderAssigneeDraft" placeholder="用户 ID，多个用逗号分隔" />
            <div v-if="reminderAssignees.length" class="service-reminder-token-list">
              <span v-for="assignee in reminderAssignees" :key="assignee.id">{{ assignee.user_id }}</span>
            </div>
          </section>
          <section class="service-reminder-collaboration-section">
            <strong>评论</strong>
            <div v-if="reminderComments.length" class="service-reminder-comment-list">
              <article v-for="comment in reminderComments" :key="comment.id">
                <div><strong>{{ comment.user_id }}</strong><small>{{ formatSourceDate(comment.created_at) }}</small></div>
                <p>{{ comment.content }}</p>
              </article>
            </div>
            <div v-else class="service-hub-panel-empty">还没有评论。</div>
            <div class="service-reminder-comment-compose">
              <t-textarea v-model="reminderCommentDraft" :autosize="{ minRows: 2, maxRows: 4 }" placeholder="记录处理意见或协作结果" />
              <button type="button" :disabled="reminderCollaborationSaving" @click="submitReminderComment">添加评论</button>
            </div>
          </section>
          <section class="service-reminder-collaboration-section">
            <strong>操作历史</strong>
            <div v-if="reminderHistory.length" class="service-reminder-history-list">
              <div v-for="history in reminderHistory" :key="history.id">
                <span>{{ history.action }}</span>
                <small>{{ history.user_id }} · {{ formatSourceDate(history.created_at) }}</small>
              </div>
            </div>
            <div v-else class="service-hub-panel-empty">暂无操作历史。</div>
          </section>
        </template>
      </div>
    </t-dialog>

    <section v-if="view === 'list'" class="service-hub-list-view">
      <header class="service-hub-page-header">
        <div>
          <h1>服务</h1>
          <p>每个服务一个空间，专家在里面干活</p>
        </div>
        <div class="service-hub-header-art" aria-hidden="true">
          <div class="service-hub-art-window">
            <span></span><span></span><span></span>
          </div>
          <div class="service-hub-art-card">
            <span></span><span></span>
          </div>
          <div class="service-hub-art-dot"></div>
        </div>
      </header>

      <div class="service-hub-create-row">
        <t-button theme="primary" class="service-hub-primary-button" @click="openCreate()">
          <template #icon><t-icon name="add" /></template>
          新建服务
        </t-button>
      </div>

      <div class="service-hub-list-main">
        <div class="service-hub-section-head">
          <span class="service-hub-section-title">我的服务</span>
          <div v-if="services.length > 0" class="service-hub-section-tools">
            <t-select v-model="sortMode" class="service-hub-sort" size="small" :options="sortOptions" />
            <t-input v-model="serviceQuery" class="service-hub-search" size="small" placeholder="搜索服务">
              <template #prefix-icon><t-icon name="search" /></template>
            </t-input>
          </div>
        </div>

        <template v-if="services.length > 0">
          <div v-if="filteredServices.length" class="service-hub-grid">
            <article
              v-for="service in filteredServices"
              :key="service.id"
              class="service-hub-card"
              @click="openWorkspace(service.id)"
            >
              <div class="service-hub-card-top">
                <div class="service-hub-card-icon">
                  <t-icon :name="templateFor(service)?.icon || 'folder'" />
                </div>
                <strong>{{ service.name }}</strong>
                <t-dropdown trigger="click" placement="bottom-right" @click.stop>
                  <button type="button" class="service-hub-more" aria-label="更多操作" @click.stop>
                    <t-icon name="more" />
                  </button>
                  <template #dropdown>
                    <t-dropdown-menu>
                      <t-dropdown-item @click="openCreate(service)">复制配置创建</t-dropdown-item>
                      <t-dropdown-item @click="archiveService(service)">
                        {{ service.state === 'archived' ? '恢复服务' : '归档服务' }}
                      </t-dropdown-item>
                    </t-dropdown-menu>
                  </template>
                </t-dropdown>
              </div>
              <div class="service-hub-card-tags">
                <span class="service-hub-tag">{{ templateFor(service)?.name || '自定义服务' }}</span>
                <span class="service-hub-role">{{ service.role }}</span>
              </div>
              <p class="service-hub-card-description">{{ service.description || '还没有填写服务描述' }}</p>
              <div class="service-hub-card-meta">
                <span>{{ service.updatedLabel }}</span>
                <span>{{ service.members }} 位成员</span>
              </div>
            </article>
          </div>
          <div v-else class="service-hub-no-result">没有找到匹配的服务</div>
        </template>
        <div v-else class="service-hub-empty-state">
          <div>
            <h2>还没有服务</h2>
            <p>从下方模板开始，配置会自动带入；也可以新建空白服务。</p>
          </div>
          <t-button variant="outline" @click="openCreate()">从空白创建</t-button>
        </div>

        <div class="service-hub-section-head service-hub-template-head">
          <span class="service-hub-section-title">从模板创建</span>
          <t-input v-model="templateQuery" class="service-hub-search" size="small" placeholder="搜索模板">
            <template #prefix-icon><t-icon name="search" /></template>
          </t-input>
        </div>
        <div class="service-hub-template-groups">
          <section v-for="group in filteredTemplateGroups" :key="group.type" class="service-hub-template-group">
            <div class="service-hub-template-group-label">{{ group.type }}</div>
            <div class="service-hub-grid">
              <button
                v-for="template in group.items"
                :key="template.id"
                type="button"
                class="service-hub-template-card"
                @click="openCreate(template)"
              >
                <div class="service-hub-template-top">
                  <div class="service-hub-template-icon"><t-icon :name="template.icon" /></div>
                  <strong>{{ template.name }}</strong>
                </div>
                <p>{{ template.description }}</p>
                <span>{{ template.subjectLabel || '服务主体' }} · 创建后可调整</span>
              </button>
            </div>
          </section>
        </div>
      </div>
    </section>

    <section v-else class="service-hub-workspace-view">
      <header class="service-hub-space-head">
        <button type="button" class="service-hub-back-button" @click="backToList">
          <t-icon name="chevron-left" />
          返回服务列表
        </button>
        <span class="service-hub-space-identity">
          <span class="service-hub-space-icon" aria-hidden="true">
            <t-icon :name="templateFor(activeService)?.icon || 'folder'" />
          </span>
          <span class="service-hub-space-name">{{ activeService?.name }}</span>
        </span>
        <span class="service-hub-space-template">{{ templateFor(activeService)?.name || '自定义服务' }}</span>
        <div class="service-hub-space-actions">
          <button
            v-for="tool in headTools"
            :key="tool.key"
            type="button"
            class="service-hub-head-tool"
            :class="{ active: panel === tool.key }"
            @click="panel = panel === tool.key ? '' : tool.key"
          >
            <t-icon :name="tool.icon" />
            <span>{{ tool.label }}</span>
            <small>{{ tool.count }}</small>
          </button>
          <span class="service-hub-active-state">{{ serviceStateLabel(activeService?.state) }}</span>
          <button type="button" class="service-hub-circle-button" title="新建会话" aria-label="新建会话" @click="startNewSession">＋</button>
          <t-dropdown trigger="click" placement="bottom-right">
            <button type="button" class="service-hub-circle-button" title="更多操作" aria-label="更多操作">
              <t-icon name="more" />
            </button>
            <template #dropdown>
              <t-dropdown-menu>
                <t-dropdown-item @click="openCreate(activeService)">复制配置创建</t-dropdown-item>
                <t-dropdown-item @click="toggleActiveServicePause">
                  {{ activeService?.state === 'paused' ? '恢复服务' : '暂停服务' }}
                </t-dropdown-item>
                <t-dropdown-item @click="archiveService(activeService)">
                  {{ activeService?.state === 'archived' ? '恢复归档服务' : '归档服务' }}
                </t-dropdown-item>
                <t-dropdown-item theme="error" @click="deleteActiveService">删除服务</t-dropdown-item>
              </t-dropdown-menu>
            </template>
          </t-dropdown>
        </div>
      </header>

      <div class="service-hub-workbench">
        <article class="service-hub-chat">
          <header class="service-hub-chat-head">
            <strong>{{ activeSession?.title || '开始一段新的工作' }}</strong>
            <span>{{ activeSession?.expert || '服务助理' }}</span>
          </header>
          <div class="service-hub-chat-body">
            <ChatView
              v-if="activeChatSessionId"
              :key="`${activeSession?.id}:${activeChatSessionId}`"
              ref="serviceChatViewRef"
              :session_id="activeChatSessionId"
              :service-id="activeService?.id || ''"
              :agent-id="serviceChatAgentId"
              :kb-ids="activeServiceKnowledgeBaseIds"
              embedded-input-placeholder="围绕当前服务整理摘要、话术和下一步"
              embedded-mode
              hosted-mode
            >
              <template #empty-suggestions>
                <div class="service-hub-chat-prompts">
                  <strong>从一段工作开始</strong>
                  <p>把要整理、分析或推进的事情告诉专家，工作过程与产物会持续沉淀在服务空间。</p>
                  <button
                    v-for="prompt in serviceChatPrompts"
                    :key="prompt"
                    type="button"
                    @click="sendServiceChatPrompt(prompt)"
                  >
                    {{ prompt }}
                  </button>
                </div>
              </template>
            </ChatView>
            <div v-else class="service-hub-chat-state">
              <t-icon :name="activeChatSessionLoading ? 'loading' : 'chat'" :class="{ 'is-loading': activeChatSessionLoading }" />
              <span>{{ activeChatSessionLoading ? '正在准备会话' : activeChatSessionError || '会话暂不可用' }}</span>
              <t-button v-if="activeChatSessionError" variant="text" theme="primary" size="small" @click="retryActiveChatSession">
                重试
              </t-button>
            </div>

            <aside v-if="panel" class="service-hub-side-panel">
              <header>
                <strong>{{ panelTitle }}</strong>
                <t-button
                  v-if="panel === 'summary'"
                  variant="text"
                  size="small"
                  :loading="planningDataLoading"
                  @click="refreshActiveSummary"
                >
                  刷新
                </t-button>
                <t-button
                  v-if="panel === 'settings'"
                  theme="primary"
                  size="small"
                  :loading="settingsSaving"
                  @click="saveServiceSettings"
                >
                  保存
                </t-button>
                <button type="button" class="service-hub-side-panel-close" aria-label="关闭面板" @click="panel = ''"><t-icon name="close" /></button>
              </header>
              <div class="service-hub-side-panel-body">
                <div v-if="panel !== 'subjects' && panel !== 'context' && planningDataLoading" class="service-hub-panel-empty">正在加载空间结构。</div>
                <div v-else-if="panel !== 'subjects' && panel !== 'context' && planningDataError" class="service-hub-panel-empty service-hub-panel-error">
                  {{ planningDataError }}
                </div>
                <template v-else-if="panel === 'context'">
                  <div v-if="contextSourcesLoading" class="service-hub-panel-empty">正在加载来源资料。</div>
                  <div v-else-if="contextSourcesError" class="service-hub-panel-empty service-hub-panel-error">
                    {{ contextSourcesError }}
                  </div>
                  <template v-else>
                    <article
                      v-for="source in contextSources"
                      :key="source.id"
                      class="service-hub-context-source"
                      :class="{ 'is-highlighted': sourceMatchesRoute(source) }"
                    >
                      <div class="service-hub-context-source-head">
                        <strong>{{ source.source_title || '未命名整理结果' }}</strong>
                        <span v-if="sourceMatchesRoute(source)" class="service-hub-context-source-badge">刚带入</span>
                        <button
                          type="button"
                          class="service-hub-context-source-remove"
                          title="移除来源"
                          aria-label="移除来源"
                          @click="removeContextSource(source.id)"
                        >
                          <t-icon name="close" />
                        </button>
                      </div>
                      <p>{{ source.source_summary || '整理结果已作为服务背景资料保存。' }}</p>
                      <div class="service-hub-context-source-meta">
                        <span>{{ source.memory_ids?.length || 0 }} 条记忆</span>
                        <span>{{ formatSourceDate(source.created_at) }}</span>
                      </div>
                      <div class="service-hub-context-source-actions">
                        <button type="button" @click="viewSourceOutput(source)">查看整理结果</button>
                        <button
                          v-if="source.memory_ids?.length"
                          type="button"
                          @click="viewSourceMemory(source.memory_ids[0])"
                        >
                          查看源记忆
                        </button>
                      </div>
                    </article>
                    <div v-if="!contextSources.length" class="service-hub-panel-empty">
                      还没有带入整理结果。可从整理详情页将结果用于当前服务。
                    </div>
                  </template>
                </template>
                <template v-else-if="panel === 'reminders'">
                  <div v-if="remindersLoading" class="service-hub-panel-empty">正在加载服务事项。</div>
                  <div v-else-if="remindersError" class="service-hub-panel-empty service-hub-panel-error">
                    {{ remindersError }}
                  </div>
                  <template v-else>
                    <form class="service-hub-reminder-form" @submit.prevent="createActiveReminder">
                      <t-input v-model="reminderDraftTitle" size="small" placeholder="事项标题，例如：回访会员家庭 A" />
                      <t-textarea
                        v-model="reminderDraftSummary"
                        size="small"
                        :autosize="{ minRows: 2, maxRows: 4 }"
                        placeholder="补充事项背景和处理目标"
                      />
                      <div class="service-hub-reminder-form-row">
                        <t-select v-model="reminderDraftPriority" size="small" :options="reminderPriorityOptions" />
                        <t-input v-model="reminderDraftDueText" size="small" placeholder="时间，例如：本周五前" />
                      </div>
                      <t-select
                        v-model="reminderDraftParentId"
                        size="small"
                        clearable
                        :options="reminderParentOptions"
                        placeholder="可选：归入父事项"
                      />
                      <t-input
                        v-model="reminderDraftAssignees"
                        size="small"
                        placeholder="负责人用户 ID，多个用逗号分隔"
                      />
                      <t-button type="submit" block theme="primary" size="small" :loading="remindersSaving">
                        新建事项
                      </t-button>
                    </form>
                    <div v-if="serviceReminders.length" class="service-hub-reminder-list">
                      <article v-for="reminder in serviceReminders" :key="reminder.id" class="service-hub-reminder-item">
                        <div class="service-hub-reminder-item-head">
                          <strong>{{ reminder.title }}</strong>
                          <span :class="`service-hub-reminder-priority is-${reminder.priority}`">
                            {{ reminderPriorityLabel(reminder.priority) }}
                          </span>
                        </div>
                        <p v-if="reminder.summary">{{ reminder.summary }}</p>
                        <div class="service-hub-reminder-meta">
                          <span>{{ reminderStatusLabel(reminder.status) }}</span>
                          <span v-if="reminder.due_text">{{ reminder.due_text }}</span>
                          <span v-if="reminder.parent_reminder_id">子事项 · {{ reminder.depth || 1 }} 层</span>
                        </div>
                        <div class="service-hub-reminder-actions">
                          <button type="button" @click="openReminderCollaboration(reminder)">协作</button>
                          <button
                            v-for="nextStatus in nextReminderStatuses(reminder)"
                            :key="nextStatus.id"
                            type="button"
                            @click="changeReminderStatus(reminder, nextStatus.status_key)"
                          >
                            {{ nextStatus.label }}
                          </button>
                          <button type="button" class="is-danger" @click="removeActiveReminder(reminder)">删除</button>
                        </div>
                      </article>
                    </div>
                    <div v-else class="service-hub-panel-empty">当前服务还没有事项。可以先记录一个需要跟进的动作。</div>
                  </template>
                </template>
                <template v-else-if="panel === 'settings'">
                  <div class="service-hub-settings-form">
                    <label>
                      <span>服务名称</span>
                      <t-input v-model="settingsName" size="small" />
                    </label>
                    <label>
                      <span>服务描述</span>
                      <t-textarea v-model="settingsDescription" size="small" :autosize="{ minRows: 2, maxRows: 4 }" />
                    </label>
                    <label>
                      <span>工作指令</span>
                      <t-textarea v-model="settingsInstruction" size="small" :autosize="{ minRows: 5, maxRows: 10 }" />
                    </label>
                    <div class="service-hub-settings-divider">档案字段</div>
                    <div v-if="settingsProfileSchema.length" class="service-hub-profile-field-list">
                      <div v-for="field in settingsProfileSchema" :key="field.key" class="service-hub-profile-field-row">
                        <t-input v-model="field.label" size="small" />
                        <span>{{ field.key }}</span>
                        <label class="service-hub-profile-field-required">
                          <input v-model="field.required" type="checkbox" />
                          必填
                        </label>
                        <button type="button" class="is-danger" @click="removeProfileField(field.key)">删除</button>
                      </div>
                    </div>
                    <div v-else class="service-hub-panel-empty">暂无档案字段。</div>
                    <div class="service-hub-profile-field-add">
                      <t-input v-model="newProfileFieldKey" size="small" placeholder="字段 key，例如 member_level" />
                      <t-input v-model="newProfileFieldLabel" size="small" placeholder="字段名称" />
                      <button type="button" @click="addProfileField">添加字段</button>
                    </div>
                    <div class="service-hub-settings-divider">事项状态</div>
                    <div v-if="reminderStatuses.length" class="service-hub-status-list">
                      <div v-for="status in reminderStatuses" :key="status.id" class="service-hub-status-row">
                        <t-input
                          :model-value="status.label"
                          size="small"
                          @blur="updateStatusLabel(status, $event)"
                        />
                        <span :class="{ 'is-disabled': !status.enabled }">{{ status.status_key }}</span>
                        <button type="button" @click="toggleReminderStatus(status)">
                          {{ status.enabled ? '停用' : '启用' }}
                        </button>
                        <button v-if="!status.is_system" type="button" class="is-danger" @click="removeReminderStatus(status)">删除</button>
                      </div>
                    </div>
                    <div class="service-hub-status-add">
                      <t-input v-model="newStatusKey" size="small" placeholder="状态 key，例如 waiting_reply" />
                      <t-input v-model="newStatusLabel" size="small" placeholder="显示名称" />
                      <t-select v-model="newStatusCategory" size="small" :options="statusCategoryOptions" />
                      <button type="button" @click="addReminderStatus">添加状态</button>
                    </div>
                  </div>
                </template>
                <template v-else-if="panel === 'artifacts'">
                  <div v-if="artifactsLoading" class="service-hub-panel-empty">正在加载服务产物。</div>
                  <div v-else-if="artifactsError" class="service-hub-panel-empty service-hub-panel-error">{{ artifactsError }}</div>
                  <button
                    v-for="artifact in activeArtifacts"
                    :key="artifact.id"
                    type="button"
                    class="service-hub-side-artifact"
                    :class="{ active: selectedArtifact?.id === artifact.id }"
                    @click="selectedArtifact = artifact"
                  >
                    <span class="service-hub-artifact-icon">{{ artifact.format || artifact.kind }}</span>
                    <span>
                      <strong>{{ artifact.title || artifact.original_name || '未命名产物' }}</strong>
                      <small>v{{ artifact.version }} · {{ artifact.lifecycle }}</small>
                    </span>
                  </button>
                  <div v-if="!activeArtifacts.length && !artifactsLoading" class="service-hub-panel-empty">当前服务还没有产物。</div>
                  <div v-if="selectedArtifact" class="service-hub-artifact-detail">
                    <strong>{{ selectedArtifact.title || selectedArtifact.original_name || '未命名产物' }}</strong>
                    <p>{{ selectedArtifact.summary || '该产物暂无摘要。' }}</p>
                    <div class="service-hub-artifact-links">
                      <a :href="artifactStreamUrl(selectedArtifact, 'preview')" target="_blank" rel="noreferrer">预览</a>
                      <a v-if="selectedArtifact.downloadable" :href="artifactStreamUrl(selectedArtifact, 'download')" target="_blank" rel="noreferrer">下载</a>
                      <button
                        v-if="selectedArtifact.lifecycle === 'temporary'"
                        type="button"
                        @click="changeArtifactLifecycle(selectedArtifact, 'saved')"
                      >
                        保存
                      </button>
                      <button
                        v-if="selectedArtifact.lifecycle !== 'archived'"
                        type="button"
                        class="is-danger"
                        @click="changeArtifactLifecycle(selectedArtifact, 'archived')"
                      >
                        归档
                      </button>
                    </div>
                  </div>
                </template>
                <template v-else-if="panel === 'profile'">
                  <div v-if="serviceProfile?.schema?.length" class="service-hub-planning-list">
                    <div v-for="field in serviceProfile.schema" :key="field.key" class="service-hub-planning-row">
                      <span>{{ field.label }}</span>
                      <strong>{{ displayPlanningValue(serviceProfile.values?.[field.key]) }}</strong>
                    </div>
                  </div>
                  <div v-else class="service-hub-panel-empty">空间档案还没有生成字段。</div>
                </template>
                <template v-else-if="panel === 'subjects'">
                  <div v-if="subjectsLoading" class="service-hub-panel-empty">正在加载服务主体。</div>
                  <div v-else-if="subjectsError" class="service-hub-panel-empty service-hub-panel-error">
                    {{ subjectsError }}
                  </div>
                  <template v-else>
                    <div class="service-hub-subject-form">
                      <t-input v-model="subjectDraftName" size="small" placeholder="主体名称，例如：会员家庭 A" />
                      <div class="service-hub-subject-form-row">
                        <t-input v-model="subjectDraftType" size="small" placeholder="主体类型，例如：member_family" />
                        <t-input v-model="subjectDraftKey" size="small" placeholder="唯一标识，例如：member-001" />
                      </div>
                      <t-button
                        block
                        theme="primary"
                        size="small"
                        :loading="subjectsSaving"
                        @click="createActiveSubject"
                      >
                        添加主体
                      </t-button>
                    </div>
                    <div v-if="serviceSubjects.length" class="service-hub-subject-list">
                      <article v-for="subject in serviceSubjects" :key="subject.id" class="service-hub-subject-item">
                        <div class="service-hub-subject-item-head">
                          <strong>{{ subject.display_name || subject.subject_key }}</strong>
                          <span>{{ subject.subject_type }}</span>
                        </div>
                        <small>{{ subject.subject_key }}</small>
                      </article>
                    </div>
                    <div v-else class="service-hub-panel-empty">当前服务还没有主体。</div>
                  </template>
                </template>
                <template v-else>
                  <div v-if="serviceSummary?.schema?.length" class="service-hub-planning-list">
                    <div v-for="section in serviceSummary.schema" :key="section.key" class="service-hub-planning-section">
                      <div>
                        <strong>{{ section.label }}</strong>
                        <span>{{ section.refresh_policy === 'on_fact_change' ? '事实变化后刷新' : '按需刷新' }}</span>
                      </div>
                      <p>{{ summarySectionText(serviceSummary.sections?.[section.key]) }}</p>
                    </div>
                  </div>
                  <div v-else class="service-hub-panel-empty">首页摘要还没有生成模块。</div>
                </template>
              </div>
            </aside>
          </div>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import ChatView from '@/views/chat/index.vue'
import { BUILTIN_SMART_REASONING_ID } from '@/api/agent'
import ServiceCreateDialog from './ServiceCreateDialog.vue'
import {
  applyServiceTemplate,
  confirmServiceBlueprint,
  createServiceSpace,
  createServiceSubject,
  deleteServiceContextSource,
  getServiceProfile,
  getServiceSummary,
  importServiceOrganizeOutput,
  createServiceReminder,
  deleteServiceReminder,
  listServiceReminderAssignees,
  listServiceReminderComments,
  listServiceReminderHistory,
  addServiceReminderComment,
  replaceServiceReminderAssignees,
  listServiceReminderStatusTransitions,
  listServiceReminderStatuses,
  listServiceReminders,
  listServiceArtifacts,
  updateServiceArtifactLifecycle,
  updateServiceSpace,
  updateServiceProfile,
  deleteServiceSpace,
  createServiceReminderStatus,
  updateServiceReminderStatus,
  deleteServiceReminderStatus,
  updateServiceReminder,
  listServiceSubjects,
  listServiceContextSources,
  previewServiceBlueprint,
  previewServiceBlueprintForService,
  refreshServiceSummary,
  setServiceSpaceState,
  type ServiceContextSource,
  type ServiceReminder,
  type ServiceReminderAssignee,
  type ServiceReminderComment,
  type ServiceReminderHistory,
  type ServiceReminderStatus,
  type ServiceReminderStatusTransition,
  type ServiceSubject,
  type ServiceSpaceBlueprint,
  type ServiceSpaceProfile,
  type ServiceSpaceProfileField,
  type ServiceSpaceSummary,
  type ServiceArtifact as ApiServiceArtifact,
} from '@/api/service'
import {
  SESSION_MUTATION_EVENT,
  type SessionMutationDetail,
} from '@/components/sessionMutations'
import {
  createServiceSession,
  getFirstServiceSession,
  getService,
  getServiceExpert,
  getServiceTemplate,
  getSession,
  loadServiceHub,
  serviceHubState,
  serviceTemplates,
  type ServiceRecord,
  type ServiceSession,
  type ServiceTemplate,
} from './serviceHubState'

type HubView = 'list' | 'workspace'
type HubPanel = '' | 'artifacts' | 'context' | 'reminders' | 'subjects' | 'profile' | 'summary' | 'settings'
type SortMode = 'recent' | 'created' | 'name'

const route = useRoute()
const router = useRouter()
const serviceBasePath = computed(() => route.meta.mobileEntry ? '/mobile/service' : '/platform/service')
const serviceChatAgentId = BUILTIN_SMART_REASONING_ID

const view = ref<HubView>('list')
const serviceQuery = ref('')
const templateQuery = ref('')
const sortMode = ref<SortMode>('recent')
const createDialogVisible = ref(false)
const createSource = ref<ServiceTemplate | ServiceRecord | null>(null)
const blueprintConfirmVisible = ref(false)
const blueprintConfirming = ref(false)
const pendingBlueprint = ref<ServiceSpaceBlueprint | null>(null)
const pendingPayload = ref<{
  name: string
  description: string
  instruction: string
  templateId: string
  expertIds: string[]
  knowledgeBaseIds: string[]
} | null>(null)
const pendingServiceId = ref('')
const pendingContextSourceId = ref('')
const pendingContextSourceType = ref('')
const contextSourceFocusId = ref('')
const selectedArtifact = ref<ApiServiceArtifact | null>(null)
const serviceArtifactsList = ref<ApiServiceArtifact[]>([])
const artifactsLoading = ref(false)
const artifactsError = ref('')
const panel = ref<HubPanel>('')
type ServiceChatViewExpose = {
  triggerSend?: (question: string) => void
}
const serviceChatViewRef = ref<ServiceChatViewExpose | null>(null)
const activeChatSessionLoadingId = ref('')
const activeChatSessionError = ref('')
const serviceProfile = ref<ServiceSpaceProfile | null>(null)
const serviceSummary = ref<ServiceSpaceSummary | null>(null)
const planningDataLoading = ref(false)
const planningDataError = ref('')
const serviceSubjects = ref<ServiceSubject[]>([])
const subjectsTotal = ref(0)
const subjectsLoading = ref(false)
const subjectsSaving = ref(false)
const subjectsError = ref('')
const subjectDraftName = ref('')
const subjectDraftType = ref('service_subject')
const subjectDraftKey = ref('')
const contextSources = ref<ServiceContextSource[]>([])
const contextSourcesLoading = ref(false)
const contextSourcesError = ref('')
const serviceReminders = ref<ServiceReminder[]>([])
const reminderStatuses = ref<ServiceReminderStatus[]>([])
const reminderTransitions = ref<ServiceReminderStatusTransition[]>([])
const remindersLoading = ref(false)
const remindersSaving = ref(false)
const remindersError = ref('')
const reminderDraftTitle = ref('')
const reminderDraftSummary = ref('')
const reminderDraftPriority = ref<'high' | 'medium' | 'low'>('medium')
const reminderDraftDueText = ref('')
const reminderDraftParentId = ref('')
const reminderDraftAssignees = ref('')
const reminderDetailVisible = ref(false)
const selectedReminder = ref<ServiceReminder | null>(null)
const reminderAssignees = ref<ServiceReminderAssignee[]>([])
const reminderComments = ref<ServiceReminderComment[]>([])
const reminderHistory = ref<ServiceReminderHistory[]>([])
const reminderCollaborationLoading = ref(false)
const reminderCommentDraft = ref('')
const reminderAssigneeDraft = ref('')
const reminderCollaborationSaving = ref(false)
const settingsSaving = ref(false)
const settingsName = ref('')
const settingsDescription = ref('')
const settingsInstruction = ref('')
const settingsProfileSchema = ref<ServiceSpaceProfileField[]>([])
const newProfileFieldKey = ref('')
const newProfileFieldLabel = ref('')
const newStatusKey = ref('')
const newStatusLabel = ref('')
const newStatusCategory = ref<'open' | 'in_progress' | 'done' | 'dismissed'>('open')

const sortOptions = [
  { label: '按最近活动', value: 'recent' },
  { label: '按创建时间', value: 'created' },
  { label: '按名称', value: 'name' },
]
const reminderPriorityOptions = [
  { label: '普通优先级', value: 'medium' },
  { label: '高优先级', value: 'high' },
  { label: '低优先级', value: 'low' },
]
const statusCategoryOptions = [
  { label: '开放', value: 'open' },
  { label: '处理中', value: 'in_progress' },
  { label: '已完成', value: 'done' },
  { label: '已关闭', value: 'dismissed' },
]

const activeService = computed(() => getService(serviceHubState.activeServiceId))
const activeSession = computed(() => getSession(serviceHubState.activeSessionId))
const activeChatSessionId = computed(() => activeSession.value?.chatSessionId || '')
const activeChatSessionLoading = computed(() => activeChatSessionLoadingId.value === activeSession.value?.id)
const activeServiceKnowledgeBaseIds = computed(() => activeService.value?.knowledgeBaseIds || [])
const services = computed(() => {
  if (serviceHubState.mode === 'archived') return serviceHubState.services.filter((service) => service.state === 'archived')
  return serviceHubState.services.filter((service) => service.state !== 'archived')
})
const filteredServices = computed(() => {
  const query = serviceQuery.value.trim().toLowerCase()
  const rows = services.value.filter((service) => `${service.name} ${service.description}`.toLowerCase().includes(query))
  return [...rows].sort((a, b) => {
    if (sortMode.value === 'name') return a.name.localeCompare(b.name, 'zh-CN')
    return b.updatedAt - a.updatedAt
  })
})
const filteredTemplateGroups = computed(() => {
  const query = templateQuery.value.trim().toLowerCase()
  const groups = new Map<string, ServiceTemplate[]>()
  serviceTemplates
    .filter((template) => `${template.name} ${template.description} ${template.spaceTypeLabel || ''} ${template.subjectLabel || ''}`
      .toLowerCase()
      .includes(query))
    .forEach((template) => {
      const type = template.spaceTypeLabel || '其他'
      const items = groups.get(type) || []
      items.push(template)
      groups.set(type, items)
    })
  return Array.from(groups, ([type, items]) => ({ type, items }))
})
const serviceChatPrompts = [
  '帮我整理本周需要优先推进的重点工作',
  '把相关资料归纳成一份可执行的清单',
]
const activeArtifacts = computed(() => {
  return serviceArtifactsList.value
})
const reminderStatusMap = computed(() => new Map(reminderStatuses.value.map((item) => [item.id, item])))
const reminderStatusByKey = computed(() => new Map(reminderStatuses.value.map((item) => [item.status_key, item])))
const templateFor = (service: ServiceRecord | undefined) => getServiceTemplate(service)
const panelTitle = computed(() => {
  switch (panel.value) {
    case 'profile':
      return '空间档案'
    case 'summary':
      return '首页摘要'
    case 'subjects':
      return '服务主体'
    case 'context':
      return '来源资料'
    case 'reminders':
      return '服务事项'
    case 'settings':
      return '服务设置'
    default:
      return '产物'
  }
})
const headTools = computed(() => [
  { key: 'artifacts' as const, label: '产物', icon: 'file', count: activeArtifacts.value.length },
  ...(route.meta.mobileEntry
    ? []
    : [{ key: 'reminders' as const, label: '事项', icon: 'check-circle', count: serviceReminders.value.length }]),
  ...(route.meta.mobileEntry
      ? []
      : [
        { key: 'context' as const, label: '来源', icon: 'link', count: contextSources.value.length },
        { key: 'subjects' as const, label: '主体', icon: 'usergroup', count: subjectsTotal.value },
        { key: 'profile' as const, label: '档案', icon: 'user', count: serviceProfile.value?.schema?.length || 0 },
        { key: 'summary' as const, label: '摘要', icon: 'view-list', count: serviceSummary.value?.schema?.length || 0 },
        { key: 'settings' as const, label: '设置', icon: 'setting', count: 0 },
      ]),
])
const reminderParentOptions = computed(() =>
  serviceReminders.value
    .filter((reminder) => reminder.id !== selectedReminder.value?.id && (reminder.depth || 0) < 4)
    .map((reminder) => ({
      label: `${'　'.repeat(reminder.depth || 0)}${reminder.title}`,
      value: reminder.id,
    })),
)

const serviceStateLabel = (state?: ServiceRecord['state']) => {
  switch (state) {
    case 'active':
      return '进行中'
    case 'paused':
      return '已暂停'
    case 'archived':
      return '已归档'
    case 'draft':
      return '草稿'
    default:
      return '服务'
  }
}

const artifactStreamUrl = (artifact: ApiServiceArtifact, mode: 'preview' | 'download') =>
  `/api/v1/services/${encodeURIComponent(artifact.service_id)}/artifacts/${encodeURIComponent(artifact.artifact_id)}/${mode}`

const createLocalIdempotencyKey = () =>
  typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID()
    : `artifact-${Date.now()}-${Math.random().toString(16).slice(2)}`

const loadArtifacts = async (serviceId: string) => {
  artifactsLoading.value = true
  artifactsError.value = ''
  try {
    const response = await listServiceArtifacts(serviceId, { page: 1, page_size: 100 })
    serviceArtifactsList.value = response?.data?.items || []
    selectedArtifact.value = serviceArtifactsList.value[0] || null
  } catch (error) {
    console.error('[ServiceHub] Failed to load service artifacts:', error)
    serviceArtifactsList.value = []
    selectedArtifact.value = null
    artifactsError.value = '服务产物暂不可用，请稍后重试'
  } finally {
    artifactsLoading.value = false
  }
}

const changeArtifactLifecycle = async (
  artifact: ApiServiceArtifact,
  lifecycle: ApiServiceArtifact['lifecycle'],
) => {
  try {
    const response = await updateServiceArtifactLifecycle(artifact.service_id, artifact.artifact_id, {
      lifecycle,
      idempotency_key: createLocalIdempotencyKey(),
    })
    if (response?.data) {
      const index = serviceArtifactsList.value.findIndex((item) => item.id === response.data.id)
      if (index >= 0) serviceArtifactsList.value[index] = response.data
      selectedArtifact.value = response.data
      MessagePlugin.success(lifecycle === 'archived' ? '产物已归档' : '产物已保存')
    }
  } catch (error) {
    console.error('[ServiceHub] Failed to update artifact lifecycle:', error)
    MessagePlugin.error('产物状态更新失败，请稍后重试')
  }
}
const openCreate = (source?: ServiceTemplate | ServiceRecord | null) => {
  createSource.value = source || null
  createDialogVisible.value = true
}

const createIdempotencyKey = () => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `service-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

const spaceTypeLabel = (spaceType?: string) => {
  switch (spaceType) {
    case 'customer_service':
      return '客户服务'
    case 'operations':
      return '运营管理'
    case 'research':
      return '研究沉淀'
    default:
      return '服务空间'
  }
}

const subjectPolicyLabel = (blueprint: ServiceSpaceBlueprint) => {
  const types = blueprint.subject_policy?.allowed_types || []
  if (!blueprint.subject_policy?.required) return '可选业务主体'
  return types.length ? types.join('、') : '动态业务主体'
}

const openWorkspace = async (serviceId: string, sessionId?: string) => {
  await loadServiceHub()
  const service = getService(serviceId)
  if (!service) return
  serviceHubState.activeServiceId = serviceId
  const session = sessionId
    ? getSession(sessionId)
    : getFirstServiceSession(serviceId)
  const active = session || await ensureServiceSession(serviceId)
  if (!active) return
  serviceHubState.activeSessionId = active.id
  view.value = 'workspace'
  await router.replace({
    path: serviceBasePath.value,
    query: { service: serviceHubState.activeServiceId, session: serviceHubState.activeSessionId },
  })
}

const displayPlanningValue = (value: unknown) => {
  if (value === undefined || value === null || value === '') return '待补充事实'
  if (Array.isArray(value)) return value.join('、')
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

const summarySectionText = (value: unknown) => {
  if (!value || typeof value !== 'object') return '等待资料沉淀后生成摘要'
  const section = value as Record<string, unknown>
  if (typeof section.content === 'string' && section.content.trim()) return section.content
  if (typeof section.status === 'string') return section.status
  return '等待资料沉淀后生成摘要'
}

const loadPlanningData = async (serviceId: string) => {
  planningDataLoading.value = true
  planningDataError.value = ''
  const [profileResult, summaryResult] = await Promise.allSettled([
    getServiceProfile(serviceId),
    getServiceSummary(serviceId),
  ])
  serviceProfile.value = profileResult.status === 'fulfilled' ? profileResult.value?.data || null : null
  serviceSummary.value = summaryResult.status === 'fulfilled' ? summaryResult.value?.data || null : null
  if (!serviceProfile.value && !serviceSummary.value) {
    planningDataError.value = '空间档案和摘要暂不可用'
  }
  planningDataLoading.value = false
}

const loadSubjects = async (serviceId: string) => {
  subjectsLoading.value = true
  subjectsError.value = ''
  try {
    const response = await listServiceSubjects(serviceId, { page: 1, page_size: 100 })
    serviceSubjects.value = response?.data?.items || []
    subjectsTotal.value = response?.data?.total || serviceSubjects.value.length
  } catch (error) {
    console.error('[ServiceHub] Failed to load service subjects:', error)
    serviceSubjects.value = []
    subjectsTotal.value = 0
    subjectsError.value = '服务主体暂不可用，请稍后重试'
  } finally {
    subjectsLoading.value = false
  }
}

const loadContextSources = async (serviceId: string) => {
  if (route.meta.mobileEntry) return
  contextSourcesLoading.value = true
  contextSourcesError.value = ''
  try {
    const response = await listServiceContextSources(serviceId)
    contextSources.value = response?.data || []
  } catch (error) {
    console.error('[ServiceHub] Failed to load context sources:', error)
    contextSources.value = []
    contextSourcesError.value = '来源资料暂不可用，请稍后重试'
  } finally {
    contextSourcesLoading.value = false
  }
}

const loadReminders = async (serviceId: string) => {
  if (route.meta.mobileEntry) return
  remindersLoading.value = true
  remindersError.value = ''
  try {
    const [reminderResponse, statusResponse, transitionResponse] = await Promise.all([
      listServiceReminders(serviceId, { page: 1, page_size: 100 }),
      listServiceReminderStatuses(serviceId),
      listServiceReminderStatusTransitions(serviceId),
    ])
    serviceReminders.value = reminderResponse?.data?.items || []
    reminderStatuses.value = statusResponse?.data || []
    reminderTransitions.value = transitionResponse?.data || []
  } catch (error) {
    console.error('[ServiceHub] Failed to load service reminders:', error)
    serviceReminders.value = []
    reminderStatuses.value = []
    reminderTransitions.value = []
    remindersError.value = '服务事项暂不可用，请稍后重试'
  } finally {
    remindersLoading.value = false
  }
}

const reminderStatusLabel = (status: string) =>
  reminderStatusByKey.value.get(status)?.label || status || '待处理'

const reminderPriorityLabel = (priority: ServiceReminder['priority']) => {
  switch (priority) {
    case 'high':
      return '高'
    case 'low':
      return '低'
    default:
      return '普通'
  }
}

const nextReminderStatuses = (reminder: ServiceReminder) => {
  const current = reminderStatusByKey.value.get(reminder.status)
  if (!current) return []
  return reminderTransitions.value
    .filter((transition) => transition.from_status_id === current.id && transition.enabled)
    .map((transition) => reminderStatusMap.value.get(transition.to_status_id))
    .filter((status): status is ServiceReminderStatus => Boolean(status && status.enabled))
}

const createActiveReminder = async () => {
  const serviceId = activeService.value?.id
  const title = reminderDraftTitle.value.trim()
  if (!serviceId || !title) {
    MessagePlugin.warning('请填写事项标题')
    return
  }
  remindersSaving.value = true
  try {
    const response = await createServiceReminder(serviceId, {
      title,
      summary: reminderDraftSummary.value.trim(),
      priority: reminderDraftPriority.value,
      due_text: reminderDraftDueText.value.trim(),
      parent_reminder_id: reminderDraftParentId.value || undefined,
      assignee_user_ids: reminderDraftAssignees.value
        .split(',')
        .map((value) => value.trim())
        .filter(Boolean),
    })
    if (response?.data) {
      serviceReminders.value = [response.data, ...serviceReminders.value]
      reminderDraftTitle.value = ''
      reminderDraftSummary.value = ''
      reminderDraftDueText.value = ''
      reminderDraftParentId.value = ''
      reminderDraftAssignees.value = ''
      MessagePlugin.success('事项已创建')
    }
  } catch (error) {
    console.error('[ServiceHub] Failed to create service reminder:', error)
    MessagePlugin.error('事项创建失败，请稍后重试')
  } finally {
    remindersSaving.value = false
  }
}

const changeReminderStatus = async (reminder: ServiceReminder, status: string) => {
  const serviceId = activeService.value?.id
  if (!serviceId || !status || status === reminder.status) return
  try {
    const response = await updateServiceReminder(serviceId, reminder.id, { status })
    if (response?.data) {
      const index = serviceReminders.value.findIndex((item) => item.id === reminder.id)
      if (index >= 0) serviceReminders.value[index] = response.data
      MessagePlugin.success(`事项已更新为${reminderStatusLabel(status)}`)
    }
  } catch (error) {
    console.error('[ServiceHub] Failed to update service reminder:', error)
    MessagePlugin.error('事项状态更新失败，请检查状态流转规则')
  }
}

const removeActiveReminder = async (reminder: ServiceReminder) => {
  const serviceId = activeService.value?.id
  if (!serviceId) return
  try {
    await deleteServiceReminder(serviceId, reminder.id)
    serviceReminders.value = serviceReminders.value.filter((item) => item.id !== reminder.id)
    MessagePlugin.success('事项已删除')
  } catch (error) {
    console.error('[ServiceHub] Failed to delete service reminder:', error)
    MessagePlugin.error('事项删除失败，请稍后重试')
  }
}

const openReminderCollaboration = async (reminder: ServiceReminder) => {
  const serviceId = activeService.value?.id
  if (!serviceId) return
  selectedReminder.value = reminder
  reminderDetailVisible.value = true
  reminderCollaborationLoading.value = true
  reminderCommentDraft.value = ''
  try {
    const [assignees, comments, history] = await Promise.all([
      listServiceReminderAssignees(serviceId, reminder.id),
      listServiceReminderComments(serviceId, reminder.id),
      listServiceReminderHistory(serviceId, reminder.id),
    ])
    reminderAssignees.value = assignees?.data || []
    reminderComments.value = comments?.data || []
    reminderHistory.value = history?.data || []
    reminderAssigneeDraft.value = reminderAssignees.value.map((item) => item.user_id).join(', ')
  } catch (error) {
    console.error('[ServiceHub] Failed to load reminder collaboration:', error)
    MessagePlugin.error('事项协作信息加载失败')
  } finally {
    reminderCollaborationLoading.value = false
  }
}

const saveReminderAssignees = async () => {
  const serviceId = activeService.value?.id
  const reminderId = selectedReminder.value?.id
  if (!serviceId || !reminderId) return
  reminderCollaborationSaving.value = true
  try {
    const response = await replaceServiceReminderAssignees(
      serviceId,
      reminderId,
      reminderAssigneeDraft.value.split(',').map((value) => value.trim()).filter(Boolean),
    )
    reminderAssignees.value = response?.data || []
    MessagePlugin.success('负责人已更新')
  } catch (error) {
    console.error('[ServiceHub] Failed to save reminder assignees:', error)
    MessagePlugin.error('负责人更新失败')
  } finally {
    reminderCollaborationSaving.value = false
  }
}

const submitReminderComment = async () => {
  const serviceId = activeService.value?.id
  const reminderId = selectedReminder.value?.id
  const content = reminderCommentDraft.value.trim()
  if (!serviceId || !reminderId || !content) {
    MessagePlugin.warning('请先填写评论内容')
    return
  }
  reminderCollaborationSaving.value = true
  try {
    const response = await addServiceReminderComment(serviceId, reminderId, content)
    if (response?.data) reminderComments.value.push(response.data)
    reminderCommentDraft.value = ''
    const history = await listServiceReminderHistory(serviceId, reminderId)
    reminderHistory.value = history?.data || reminderHistory.value
    MessagePlugin.success('评论已添加')
  } catch (error) {
    console.error('[ServiceHub] Failed to add reminder comment:', error)
    MessagePlugin.error('评论添加失败')
  } finally {
    reminderCollaborationSaving.value = false
  }
}

const sourceMatchesRoute = (source: ServiceContextSource) =>
  Boolean(contextSourceFocusId.value)
  && (source.source_id === contextSourceFocusId.value || source.id === contextSourceFocusId.value)

const formatSourceDate = (value?: string) => {
  if (!value) return '刚刚带入'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '刚刚带入'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date).replace('/', '-')
}

const viewSourceOutput = async (source: ServiceContextSource) => {
  if (!source.source_id) return
  await router.push({
    path: `/platform/organize/outputs/${encodeURIComponent(source.source_id)}`,
    query: { from: 'service' },
  })
}

const viewSourceMemory = async (memoryId: string) => {
  if (!memoryId) return
  await router.push(`/platform/organize/editor/memory/${encodeURIComponent(memoryId)}`)
}

const removeContextSource = async (sourceId: string) => {
  const serviceId = activeService.value?.id
  if (!serviceId || !sourceId) return
  try {
    await deleteServiceContextSource(serviceId, sourceId)
    contextSources.value = contextSources.value.filter((source) => source.id !== sourceId)
    MessagePlugin.success('来源已从服务中移除')
  } catch (error) {
    console.error('[ServiceHub] Failed to remove context source:', error)
    MessagePlugin.error('来源移除失败，请稍后重试')
  }
}

const createActiveSubject = async () => {
  const serviceId = activeService.value?.id
  const displayName = subjectDraftName.value.trim()
  const subjectType = subjectDraftType.value.trim()
  const subjectKey = subjectDraftKey.value.trim() || displayName
  if (!serviceId || !displayName || !subjectType || !subjectKey) {
    MessagePlugin.warning('请填写主体名称、主体类型和唯一标识')
    return
  }
  subjectsSaving.value = true
  try {
    const response = await createServiceSubject(serviceId, {
      subject_type: subjectType,
      subject_key: subjectKey,
      display_name: displayName,
      visibility_scope: 'private',
    })
    if (response?.data) {
      serviceSubjects.value = [response.data, ...serviceSubjects.value]
      subjectsTotal.value += 1
      subjectDraftName.value = ''
      subjectDraftKey.value = ''
      MessagePlugin.success('主体已添加')
    }
  } catch (error) {
    console.error('[ServiceHub] Failed to create service subject:', error)
    MessagePlugin.error('主体添加失败，请检查类型和唯一标识')
  } finally {
    subjectsSaving.value = false
  }
}

const refreshActiveSummary = async () => {
  if (!activeService.value?.id || planningDataLoading.value) return
  planningDataLoading.value = true
  planningDataError.value = ''
  try {
    const response = await refreshServiceSummary(activeService.value.id)
    serviceSummary.value = response?.data || null
    MessagePlugin.success('首页摘要已刷新')
  } catch (error) {
    console.error('[ServiceHub] Failed to refresh service summary:', error)
    planningDataError.value = '首页摘要刷新失败，请稍后重试'
    MessagePlugin.error(planningDataError.value)
  } finally {
    planningDataLoading.value = false
  }
}

const loadServiceSettings = async (serviceId: string) => {
  const service = getService(serviceId)
  if (!service) return
  settingsName.value = service.name
  settingsDescription.value = service.description || ''
  settingsInstruction.value = service.instruction || ''
  const [profileResult] = await Promise.allSettled([
    getServiceProfile(serviceId),
    reminderStatuses.value.length ? Promise.resolve() : loadReminders(serviceId),
  ])
  if (profileResult.status === 'fulfilled') {
    settingsProfileSchema.value = (profileResult.value?.data?.schema || []).map((field) => ({ ...field }))
  }
}

const saveServiceSettings = async () => {
  const service = activeService.value
  if (!service || !settingsName.value.trim()) {
    MessagePlugin.warning('服务名称不能为空')
    return
  }
  settingsSaving.value = true
  try {
    const response = await updateServiceSpace(service.id, {
      name: settingsName.value.trim(),
      description: settingsDescription.value.trim(),
      instruction: settingsInstruction.value.trim(),
    })
    await updateServiceProfile(service.id, serviceProfile.value?.values || {}, settingsProfileSchema.value)
    const updated = response?.data
    if (updated) {
      service.name = updated.name
      service.description = updated.description || ''
      service.instruction = updated.instruction || ''
      MessagePlugin.success('服务设置已保存')
    }
  } catch (error) {
    console.error('[ServiceHub] Failed to save service settings:', error)
    MessagePlugin.error('服务设置保存失败')
  } finally {
    settingsSaving.value = false
  }
}

const addProfileField = () => {
  const key = newProfileFieldKey.value.trim()
  const label = newProfileFieldLabel.value.trim()
  if (!key || !label) {
    MessagePlugin.warning('请填写字段 key 和字段名称')
    return
  }
  if (settingsProfileSchema.value.some((field) => field.key === key)) {
    MessagePlugin.warning('字段 key 不能重复')
    return
  }
  settingsProfileSchema.value.push({
    key,
    label,
    value_type: 'text',
    source: 'manual',
    required: false,
    sensitive: false,
    display_order: settingsProfileSchema.value.length + 1,
  })
  newProfileFieldKey.value = ''
  newProfileFieldLabel.value = ''
}

const removeProfileField = (key: string) => {
  settingsProfileSchema.value = settingsProfileSchema.value
    .filter((field) => field.key !== key)
    .map((field, index) => ({ ...field, display_order: index + 1 }))
}

const addReminderStatus = async () => {
  const serviceId = activeService.value?.id
  const statusKey = newStatusKey.value.trim()
  const label = newStatusLabel.value.trim()
  if (!serviceId || !statusKey || !label) {
    MessagePlugin.warning('请填写状态 key 和显示名称')
    return
  }
  try {
    const response = await createServiceReminderStatus(serviceId, {
      status_key: statusKey,
      label,
      category: newStatusCategory.value,
      display_order: reminderStatuses.value.length + 1,
    })
    if (response?.data) reminderStatuses.value.push(response.data)
    newStatusKey.value = ''
    newStatusLabel.value = ''
    MessagePlugin.success('事项状态已添加')
  } catch (error) {
    console.error('[ServiceHub] Failed to create reminder status:', error)
    MessagePlugin.error('事项状态添加失败')
  }
}

const updateStatusLabel = async (status: ServiceReminderStatus, payload: unknown) => {
  const target = payload && typeof payload === 'object' ? (payload as { target?: { value?: unknown } }).target : null
  const nextLabel = String(typeof payload === 'string' ? payload : target?.value || '').trim()
  if (!nextLabel || nextLabel === status.label) return
  try {
    const response = await updateServiceReminderStatus(status.service_id, status.id, { label: nextLabel })
    if (response?.data) Object.assign(status, response.data)
    MessagePlugin.success('状态名称已更新')
  } catch (error) {
    console.error('[ServiceHub] Failed to update reminder status:', error)
    MessagePlugin.error('状态名称更新失败')
  }
}

const toggleReminderStatus = async (status: ServiceReminderStatus) => {
  try {
    const response = await updateServiceReminderStatus(status.service_id, status.id, { enabled: !status.enabled })
    if (response?.data) Object.assign(status, response.data)
  } catch (error) {
    console.error('[ServiceHub] Failed to toggle reminder status:', error)
    MessagePlugin.error('状态启停失败')
  }
}

const removeReminderStatus = async (status: ServiceReminderStatus) => {
  try {
    await deleteServiceReminderStatus(status.service_id, status.id)
    reminderStatuses.value = reminderStatuses.value.filter((item) => item.id !== status.id)
    MessagePlugin.success('事项状态已删除')
  } catch (error) {
    console.error('[ServiceHub] Failed to delete reminder status:', error)
    MessagePlugin.error('事项状态删除失败')
  }
}

const toggleActiveServicePause = async () => {
  const service = activeService.value
  if (!service) return
  const nextState = service.state === 'paused' ? 'active' : 'paused'
  try {
    await setServiceSpaceState(service.id, nextState)
    service.state = nextState
    MessagePlugin.success(nextState === 'paused' ? '服务已暂停' : '服务已恢复')
  } catch (error) {
    console.error('[ServiceHub] Failed to toggle service pause:', error)
    MessagePlugin.error('服务状态更新失败')
  }
}

const deleteActiveService = async () => {
  const service = activeService.value
  if (!service) return
  try {
    await deleteServiceSpace(service.id)
    await loadServiceHub(true)
    MessagePlugin.success('服务已删除')
    await backToList()
  } catch (error) {
    console.error('[ServiceHub] Failed to delete service:', error)
    MessagePlugin.error('服务删除失败')
  }
}

const backToList = async () => {
  view.value = 'list'
  panel.value = ''
  selectedArtifact.value = null
  await router.replace(serviceBasePath.value)
}

const syncFromRoute = async () => {
  await loadServiceHub()
  const queryService = typeof route.query.service === 'string' ? route.query.service : ''
  const querySession = typeof route.query.session === 'string' ? route.query.session : ''
  const queryAction = typeof route.query.action === 'string' ? route.query.action : ''
  const queryCreate = route.query.create === '1'
  const querySourceType = typeof route.query.source_type === 'string' ? route.query.source_type : ''
  const querySourceId = typeof route.query.source_id === 'string' ? route.query.source_id : ''
  const queryContextSource = typeof route.query.context_source === 'string' ? route.query.context_source : ''
  contextSourceFocusId.value = queryContextSource
  if (!route.meta.mobileEntry && queryCreate && querySourceType === 'organize_output' && querySourceId) {
    pendingContextSourceType.value = querySourceType
    pendingContextSourceId.value = querySourceId
    openCreate()
    return
  }
  if (queryService && getService(queryService)) {
    serviceHubState.activeServiceId = queryService
    const service = getService(queryService)
    if (queryAction === 'copy' && service) {
      openCreate(service)
      return
    }
    if (queryAction === 'archive' && service) {
      await archiveService(service, true)
      return
    }
    const routeSession = querySession ? getSession(querySession) : undefined
    const session = routeSession?.serviceId === queryService
      ? routeSession
      : getFirstServiceSession(queryService) || await ensureServiceSession(queryService)
    if (!session) return
    serviceHubState.activeSessionId = session.id
    view.value = 'workspace'
    await loadContextSources(queryService)
    if (queryContextSource) {
      panel.value = 'context'
    }
    return
  }
  view.value = 'list'
}

type CreateServicePayload = {
  name: string
  description: string
  instruction: string
  templateId: string
  expertIds: string[]
  knowledgeBaseIds: string[]
}

const serviceExpertsForPayload = (payload: CreateServicePayload) => payload.expertIds.length
  ? payload.expertIds.map((expertId, index) => {
      const expert = getServiceExpert(expertId)
      return {
        expert_ref: expertId,
        expert_name: expert?.name || expertId,
        expert_domain: expert?.domain,
        display_order: index,
        enabled: true,
      }
    })
  : [{
      expert_ref: BUILTIN_SMART_REASONING_ID,
      expert_name: '服务助理',
      display_order: 0,
      enabled: true,
    }]

const importPendingContextSource = async (serviceId: string) => {
  if (
    route.meta.mobileEntry
    || pendingContextSourceType.value !== 'organize_output'
    || !pendingContextSourceId.value
  ) {
    return false
  }
  try {
    await importServiceOrganizeOutput(serviceId, pendingContextSourceId.value)
    pendingContextSourceType.value = ''
    pendingContextSourceId.value = ''
    return true
  } catch (error) {
    console.error('[ServiceHub] Failed to import pending context source:', error)
    MessagePlugin.error('服务已创建，但整理结果带入失败，请从整理详情页重试')
    return false
  }
}

const openCreatedService = async (serviceId: string, message = '服务已创建') => {
  await loadServiceHub(true)
  const session = await ensureServiceSession(serviceId)
  await importPendingContextSource(serviceId)
  serviceHubState.activeServiceId = serviceId
  serviceHubState.activeSessionId = session.id
  createDialogVisible.value = false
  blueprintConfirmVisible.value = false
  view.value = 'workspace'
  await router.replace({
    path: serviceBasePath.value,
    query: { service: serviceId, session: session.id },
  })
  MessagePlugin.success(message)
}

const createServiceDirectly = async (payload: CreateServicePayload) => {
  const response = await createServiceSpace({
    name: payload.name.trim(),
    description: payload.description.trim(),
    instruction: payload.instruction.trim(),
    template_key: payload.templateId || undefined,
    knowledge_base_ids: payload.knowledgeBaseIds,
    selected_skills: [],
    activate: true,
    experts: serviceExpertsForPayload(payload),
  })
  const serviceId = response?.data?.id
  if (!serviceId) throw new Error('missing service id')
  await openCreatedService(serviceId)
}

const persistService = async (payload: CreateServicePayload) => {
  const name = payload.name.trim()
  if (!name) {
    MessagePlugin.error('请输入服务名称')
    return
  }
  const normalizedPayload = { ...payload, name }
  try {
    const selectedTemplate = serviceTemplates.find((template) => template.id === payload.templateId)
    if (route.meta.mobileEntry || !selectedTemplate?.autoApply) {
      await createServiceDirectly(normalizedPayload)
      return
    }

    const response = await applyServiceTemplate(selectedTemplate.id, {
      name,
      description: payload.description.trim(),
      instruction: payload.instruction.trim(),
      template_version: 1,
      knowledge_base_ids: payload.knowledgeBaseIds,
      experts: serviceExpertsForPayload(payload),
      idempotency_key: createIdempotencyKey(),
    })
    const serviceId = response?.data?.id
    if (!serviceId) throw new Error('missing service id')
    await openCreatedService(serviceId, '已按审核模板创建服务')
  } catch (error) {
    console.error('[ServiceHub] Failed to create service:', error)
    const message = error && typeof error === 'object' && 'message' in error
      ? String((error as { message?: unknown }).message || '')
      : error instanceof Error
        ? error.message
        : ''
    MessagePlugin.error(message ? `服务创建失败：${message}` : '服务创建失败，请稍后重试')
  }
}

const confirmPendingBlueprint = async () => {
  const payload = pendingPayload.value
  if (!payload || !pendingBlueprint.value || blueprintConfirming.value) return
  blueprintConfirming.value = true
  try {
    let serviceId = pendingServiceId.value
    if (!serviceId) {
      const response = await createServiceSpace({
        name: payload.name.trim(),
        description: payload.description.trim(),
        instruction: payload.instruction.trim(),
        knowledge_base_ids: payload.knowledgeBaseIds,
        selected_skills: [],
        activate: false,
        experts: serviceExpertsForPayload(payload),
      })
      serviceId = response?.data?.id || ''
      if (!serviceId) throw new Error('missing service id')
      pendingServiceId.value = serviceId
    }

    const previewResponse = await previewServiceBlueprintForService(serviceId, {
      instruction: payload.instruction.trim() || payload.description.trim() || payload.name.trim(),
    })
    const blueprint = previewResponse?.data
    if (!blueprint?.id) throw new Error('missing blueprint id')
    await confirmServiceBlueprint(serviceId, {
      blueprint_id: blueprint.id,
      expected_version: blueprint.version,
      activate: true,
      idempotency_key: createIdempotencyKey(),
    })
    pendingPayload.value = null
    pendingBlueprint.value = null
    pendingServiceId.value = ''
    await openCreatedService(serviceId, '服务空间已按蓝图创建')
  } catch (error) {
    console.error('[ServiceHub] Failed to confirm service blueprint:', error)
    const message = error && typeof error === 'object' && 'message' in error
      ? String((error as { message?: unknown }).message || '')
      : error instanceof Error
        ? error.message
        : ''
    MessagePlugin.error(message ? `蓝图确认失败：${message}` : '蓝图确认失败，请稍后重试')
  } finally {
    blueprintConfirming.value = false
  }
}

const cancelPendingBlueprint = () => {
  if (blueprintConfirming.value) return
  pendingBlueprint.value = null
  pendingPayload.value = null
  pendingServiceId.value = ''
}

const submitForm = async (payload: {
  name: string
  description: string
  instruction: string
  templateId: string
  expertIds: string[]
  knowledgeBaseIds: string[]
}) => {
  const normalizedPayload = payload as CreateServicePayload
  if (!route.meta.mobileEntry && !normalizedPayload.templateId) {
    try {
      const response = await previewServiceBlueprint({
        instruction: normalizedPayload.instruction.trim()
          || normalizedPayload.description.trim()
          || normalizedPayload.name.trim(),
      })
      if (!response?.data) throw new Error('missing blueprint')
      pendingPayload.value = normalizedPayload
      pendingBlueprint.value = response.data
      createDialogVisible.value = false
      blueprintConfirmVisible.value = true
      return
    } catch (error) {
      console.error('[ServiceHub] Failed to preview service blueprint:', error)
      MessagePlugin.error('空间蓝图生成失败，请补充指令后重试')
      return
    }
  }
  await persistService(normalizedPayload)
}

const serviceSessionRequests = new Map<string, Promise<ServiceSession>>()

const ensureServiceSession = async (serviceId: string) => {
  const existing = getFirstServiceSession(serviceId)
  if (existing) return existing
  const pending = serviceSessionRequests.get(serviceId)
  if (pending) return pending
  const request = createServiceSession(serviceId, {
    title: '开始一段新的工作',
  }).finally(() => {
    serviceSessionRequests.delete(serviceId)
  })
  serviceSessionRequests.set(serviceId, request)
  return request
}

const startNewSession = async () => {
  if (!activeService.value) return
  try {
    const session = await createServiceSession(activeService.value.id, {
      title: '开始一段新的工作',
    })
    serviceHubState.activeSessionId = session.id
    panel.value = ''
    selectedArtifact.value = null
    await router.replace({
      path: serviceBasePath.value,
      query: { service: activeService.value.id, session: session.id },
    })
  } catch (error) {
    console.error('[ServiceHub] Failed to create service session:', error)
    MessagePlugin.error('新会话创建失败，请稍后重试')
  }
}

const archiveService = async (service: ServiceRecord | undefined, navigate = view.value === 'workspace') => {
  if (!service) return
  const nextState = service.state === 'archived' ? 'active' : 'archived'
  try {
    await setServiceSpaceState(service.id, nextState)
    service.state = nextState
    MessagePlugin.success(nextState === 'archived' ? '服务已归档' : '服务已恢复')
    if (navigate || (view.value === 'workspace' && nextState === 'archived')) {
      await backToList()
    }
  } catch (error) {
    console.error('[ServiceHub] Failed to change service state:', error)
    MessagePlugin.error('服务状态更新失败')
  }
}

const chatSessionRequests = new Map<string, Promise<string>>()

const ensureChatSession = async (session = activeSession.value) => {
  if (!session) return ''
  if (session.chatSessionId) return session.chatSessionId
  const pending = chatSessionRequests.get(session.id)
  if (pending) return pending

  activeChatSessionLoadingId.value = session.id
  activeChatSessionError.value = ''
  const request = (async () => {
    try {
      const chatSessionId = session.chatSessionId || session.id
      session.chatSessionId = chatSessionId
      return session.chatSessionId
    } catch (error) {
      console.error('[ServiceHub] Failed to create chat session:', error)
      activeChatSessionError.value = '会话创建失败，请稍后重试'
      return ''
    } finally {
      chatSessionRequests.delete(session.id)
      if (activeChatSessionLoadingId.value === session.id) {
        activeChatSessionLoadingId.value = ''
      }
    }
  })()
  chatSessionRequests.set(session.id, request)
  return request
}

const retryActiveChatSession = () => {
  activeChatSessionError.value = ''
  void ensureChatSession()
}

const sendServiceChatPrompt = (prompt: string) => {
  serviceChatViewRef.value?.triggerSend?.(prompt)
}

const handleServiceSessionMutation = (event: Event) => {
  const detail = (event as CustomEvent<SessionMutationDetail>).detail
  if (!detail?.sessionId) return
  const session = serviceHubState.sessions.find((item) => item.chatSessionId === detail.sessionId)
  if (!session) return

  if (detail.patch?.title) {
    session.title = detail.patch.title
  }
  if (typeof detail.patch?.is_pinned === 'boolean') {
    session.pinned = detail.patch.is_pinned
  }
  if (detail.removed) {
    session.chatSessionId = undefined
    if (session.id === serviceHubState.activeSessionId) {
      void ensureChatSession(session)
    }
  }
}

watch(panel, (value) => {
  if ((value === 'profile' || value === 'summary') && activeService.value?.id) {
    void loadPlanningData(activeService.value.id)
  }
  if (value === 'subjects' && activeService.value?.id) {
    void loadSubjects(activeService.value.id)
  }
  if (value === 'reminders' && activeService.value?.id) {
    void loadReminders(activeService.value.id)
  }
  if (value === 'artifacts' && activeService.value?.id) {
    void loadArtifacts(activeService.value.id)
  }
  if (value === 'settings' && activeService.value?.id) {
    void loadServiceSettings(activeService.value.id)
  }
})

watch(
  () => activeService.value?.id,
  (serviceId) => {
    serviceProfile.value = null
    serviceSummary.value = null
    planningDataError.value = ''
    serviceSubjects.value = []
    subjectsTotal.value = 0
    subjectsError.value = ''
    serviceReminders.value = []
    reminderStatuses.value = []
    reminderTransitions.value = []
    remindersError.value = ''
    serviceArtifactsList.value = []
    selectedArtifact.value = null
    artifactsError.value = ''
    contextSources.value = []
    contextSourcesError.value = ''
    subjectDraftName.value = ''
    subjectDraftKey.value = ''
    if (serviceId) {
      void loadSubjects(serviceId)
      void loadContextSources(serviceId)
      void loadReminders(serviceId)
      void loadArtifacts(serviceId)
    }
    if ((panel.value === 'profile' || panel.value === 'summary') && serviceId) {
      void loadPlanningData(serviceId)
    }
  },
)

onMounted(async () => {
  window.addEventListener(SESSION_MUTATION_EVENT, handleServiceSessionMutation)
  await loadServiceHub()
  await syncFromRoute()
})
onUnmounted(() => window.removeEventListener(SESSION_MUTATION_EVENT, handleServiceSessionMutation))

watch(
  () => [
    route.query.service,
    route.query.session,
    route.query.action,
    route.query.create,
    route.query.source_type,
    route.query.source_id,
    route.query.context_source,
  ],
  () => syncFromRoute(),
  { immediate: true },
)

watch(
  activeSession,
  (session) => {
    if (session) void ensureChatSession(session)
  },
  { immediate: true },
)
</script>

<style scoped lang="less">
.service-hub-page {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.service-hub-list-view,
.service-hub-create-view,
.service-hub-workspace-view {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  box-sizing: border-box;
}

.service-hub-list-view,
.service-hub-create-view {
  overflow-y: auto;
  padding: 20px 28px 32px;
}

.service-hub-list-view {
  padding-right: 28px;
}

.service-hub-page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  min-height: 60px;
  margin-bottom: 16px;

  h1 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: 21px;
    font-weight: 500;
    line-height: 30px;
    letter-spacing: 0;
  }

  p {
    margin: 4px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }
}

.service-hub-header-art {
  position: relative;
  flex: none;
  width: 124px;
  height: 54px;
  opacity: 0.72;
}

.service-hub-art-window,
.service-hub-art-card {
  position: absolute;
  display: flex;
  align-items: center;
  gap: 4px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
}

.service-hub-art-window {
  top: 10px;
  left: 2px;
  width: 44px;
  height: 32px;
  padding: 0 8px;
  flex-wrap: wrap;
}

.service-hub-art-window span {
  width: 4px;
  height: 4px;
  border: 1px solid var(--td-component-border);
  border-radius: 50%;
}

.service-hub-art-window span:nth-child(3) {
  width: 21px;
  height: 1px;
  border: 0;
  border-radius: 0;
  background: var(--td-component-border);
}

.service-hub-art-card {
  top: 13px;
  left: 58px;
  width: 42px;
  height: 26px;
  padding: 0 8px;
  border-color: var(--td-brand-color);
}

.service-hub-art-card span {
  display: block;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--td-brand-color);
}

.service-hub-art-card span:last-child {
  width: 12px;
  height: 1px;
  border-radius: 0;
}

.service-hub-art-dot {
  position: absolute;
  top: 18px;
  right: 3px;
  width: 17px;
  height: 17px;
  border: 1px solid var(--td-component-border);
  border-radius: 50%;
}

.service-hub-create-row {
  margin-bottom: 24px;
}

.service-hub-primary-button {
  background: var(--td-brand-color);
  border: 0;
  color: var(--td-text-color-anti);
}

.service-hub-primary-button:hover {
  background: var(--td-brand-color-hover);
}

.service-hub-list-main {
  min-width: 0;
}

.service-hub-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.service-hub-section-title {
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
}

.service-hub-section-tools {
  display: flex;
  align-items: center;
  gap: 8px;
}

.service-hub-sort {
  width: 132px;
}

.service-hub-search {
  width: 180px;
}

.service-hub-template-head {
  margin-top: 26px;
}

.service-hub-template-groups {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.service-hub-template-group-label {
  margin-bottom: 6px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 500;
}

.service-hub-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.service-hub-card,
.service-hub-template-card {
  min-width: 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  cursor: pointer;
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.service-hub-card {
  min-height: 137px;
  padding: 10px 11px;
}

.service-hub-card:hover,
.service-hub-template-card:hover {
  border-color: var(--td-brand-color);
  box-shadow: 0 3px 12px rgba(0, 0, 0, 0.04);
}

.service-hub-card-top,
.service-hub-template-top {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 7px;
}

.service-hub-card-top strong,
.service-hub-template-top strong {
  min-width: 0;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 500;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-card-icon,
.service-hub-template-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  border-radius: 7px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-card-icon {
  width: 24px;
  height: 24px;
}

.service-hub-template-icon {
  width: 22px;
  height: 22px;
}

.service-hub-more {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 24px;
  height: 24px;
  margin-left: auto;
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-hub-more:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.service-hub-card-tags {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 8px;
}

.service-hub-tag,
.service-hub-role {
  display: inline-flex;
  align-items: center;
  min-height: 18px;
  padding: 1px 5px;
  border-radius: 3px;
  font-size: 11px;
  line-height: 16px;
  white-space: nowrap;
}

.service-hub-tag {
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-role {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
}

.service-hub-card-description {
  display: -webkit-box;
  min-height: 34px;
  margin: 7px 0 0;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 17px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.service-hub-card-meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-top: 7px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 16px;
}

.service-hub-template-card {
  display: flex;
  width: 100%;
  min-height: 108px;
  padding: 11px 12px;
  flex-direction: column;
  font: inherit;
  text-align: left;
}

.service-hub-template-card p {
  margin: 7px 0 0;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 17px;
}

.service-hub-template-card > span {
  display: block;
  margin-top: 5px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 16px;
}

.service-hub-no-result {
  padding: 32px 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  text-align: center;
}

.service-hub-empty-state {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-top: 6px;
  padding: 20px 22px;
  border: 1px dashed var(--td-component-border);
  border-radius: 12px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-empty-state h2 {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
}

.service-hub-empty-state p {
  margin: 6px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 22px;
}

.service-hub-create-view {
  max-width: 1080px;
}

.service-hub-create-header {
  margin-bottom: 22px;
}

.service-hub-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.service-hub-form-panel {
  padding: 20px 22px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-container);
}

.service-hub-field {
  margin-bottom: 16px;
}

.service-hub-field:last-child {
  margin-bottom: 0;
}

.service-hub-field > label {
  display: block;
  margin-bottom: 6px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
}

.service-hub-field > label em {
  color: var(--td-error-color);
  font-style: normal;
}

.service-hub-count {
  margin-top: 5px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  text-align: right;
}

.service-hub-error {
  display: block;
  margin-top: 5px;
  color: var(--td-error-color);
  font-size: 12px;
}

.service-hub-field-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.service-hub-field-head h2 {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
}

.service-hub-field-head p {
  margin: 4px 0 0;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.service-hub-picker {
  margin-bottom: 12px;
  padding: 8px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
}

.service-hub-picker-option,
.service-hub-choice-row {
  display: flex;
  align-items: center;
  width: 100%;
  border: 0;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font-family: var(--app-font-family);
  text-align: left;
}

.service-hub-picker-option {
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 6px;
}

.service-hub-picker-option:hover,
.service-hub-picker-option.selected,
.service-hub-choice-row:hover,
.service-hub-choice-row.selected {
  background: var(--td-brand-color-1);
}

.service-hub-picker-option > span,
.service-hub-choice-row > span:last-child {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 2px;
}

.service-hub-picker-option strong,
.service-hub-choice-row strong {
  font-size: 13px;
  font-weight: 500;
}

.service-hub-picker-option small,
.service-hub-choice-row small {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 17px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-expert-list,
.service-hub-choice-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.service-hub-expert-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
}

.service-hub-expert-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex: none;
  border-radius: 7px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
  font-size: 12px;
  font-weight: 600;
}

.service-hub-expert-row > div:nth-child(2) {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.service-hub-expert-row strong {
  font-size: 13px;
  font-weight: 500;
}

.service-hub-expert-row small {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-expert-row button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-hub-form-empty {
  padding: 10px 12px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.service-hub-choice-row {
  gap: 10px;
  padding: 9px 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
}

.service-hub-choice-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  flex: none;
  border: 1px solid var(--td-component-border);
  border-radius: 3px;
  color: transparent;
}

.service-hub-choice-row.selected .service-hub-choice-check {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color);
  color: var(--td-text-color-anti);
}

.service-hub-form-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 20px;
}

.service-hub-form-footer > div {
  display: flex;
  align-items: center;
  gap: 8px;
}

.service-hub-form-footer span {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.service-hub-workspace-view {
  padding: 0 24px 18px 28px;
  overflow: hidden;
}

.service-hub-space-head {
  display: flex;
  align-items: center;
  flex: none;
  min-width: 0;
  min-height: 64px;
  gap: 10px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-hub-back-button {
  display: inline-flex;
  align-items: center;
  flex: none;
  gap: 4px;
  min-height: 30px;
  padding: 5px 8px;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  white-space: nowrap;
}

.service-hub-back-button:hover {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.service-hub-back-button .t-icon {
  font-size: 16px;
}

.service-hub-space-identity {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
}

.service-hub-space-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex: none;
  border-radius: 7px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-space-name {
  max-width: 220px;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-space-template {
  max-width: 180px;
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-space-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  min-width: 0;
  margin-left: auto;
  gap: 6px;
}

.service-hub-head-tool {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 30px;
  gap: 5px;
  padding: 4px 8px;
  border: 1px solid transparent;
  border-radius: 5px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  white-space: nowrap;
}

.service-hub-head-tool:hover,
.service-hub-head-tool.active {
  border-color: var(--td-component-stroke);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-brand-color-7);
}

.service-hub-head-tool .t-icon {
  font-size: 15px;
}

.service-hub-head-tool small {
  min-width: 16px;
  padding: 0 3px;
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  line-height: 16px;
  text-align: center;
}

.service-hub-head-tool.active small {
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-active-state {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 8px;
  border-radius: 12px;
  background: var(--td-success-color-1);
  color: var(--td-success-color-7);
  font-size: 11px;
  line-height: 18px;
  white-space: nowrap;
}

.service-hub-active-state::before {
  width: 6px;
  height: 6px;
  margin-right: 5px;
  border-radius: 50%;
  background: currentColor;
  content: '';
}

.service-hub-circle-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  line-height: 1;
}

.service-hub-circle-button:hover {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-workspace-view > .service-hub-space-head + .service-hub-workbench {
  min-height: 0;
  margin-top: 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.service-hub-workbench {
  display: flex;
  min-width: 0;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.service-hub-chat {
  display: flex;
  position: relative;
  flex: 1;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}

.service-hub-chat-head {
  display: flex;
  align-items: center;
  flex: none;
  min-height: 52px;
  gap: 8px;
  padding: 0 16px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-hub-chat-head strong {
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-chat-head span {
  flex: none;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.service-hub-chat-body {
  display: flex;
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.service-hub-chat-prompts {
  display: flex;
  max-width: 640px;
  margin: 84px auto 0;
  padding: 0 12px;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
}

.service-hub-chat-prompts > strong {
  color: var(--td-text-color-primary);
  font-size: 22px;
  font-weight: 600;
  line-height: 30px;
}

.service-hub-chat-prompts > p {
  max-width: 560px;
  margin: 0 0 4px;
  color: var(--td-text-color-secondary);
  font-size: 14px;
  line-height: 22px;
}

.service-hub-chat-prompts > button {
  display: inline-flex;
  align-items: center;
  min-height: 34px;
  padding: 6px 11px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  line-height: 20px;
  text-align: left;
}

.service-hub-chat-prompts > button:hover {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-chat-body :deep(.chat) {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  min-height: 0;
  height: 100%;
  padding: 0;
}

.service-hub-chat-body :deep(.chat_scroll_box) {
  padding: 12px 18px 0;
}

.service-hub-chat-body :deep(.msg_list) {
  max-width: 960px;
}

.service-hub-side-panel {
  display: flex;
  width: min(360px, 36%);
  min-width: 280px;
  flex: none;
  flex-direction: column;
  border-left: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
}

.service-hub-side-panel > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 48px;
  padding: 0 14px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-hub-side-panel > header > strong {
  margin-right: auto;
  font-size: 13px;
  font-weight: 500;
}

.service-hub-side-panel-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-hub-side-panel-close:hover {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.service-hub-side-panel-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 12px;
}

.service-hub-side-artifact {
  display: flex;
  align-items: center;
  width: 100%;
  gap: 8px;
  padding: 9px 8px;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  text-align: left;
}

.service-hub-side-artifact:hover,
.service-hub-side-artifact.active {
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-side-artifact > span:last-child {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.service-hub-side-artifact strong,
.service-hub-side-artifact small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-side-artifact strong {
  font-size: 12px;
  font-weight: 500;
}

.service-hub-side-artifact small {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-hub-artifact-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex: none;
  border-radius: 4px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
  font-size: 10px;
}

.service-hub-artifact-preview {
  margin: 10px 0 0;
  padding: 10px;
  overflow: auto;
  border-radius: 5px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font: inherit;
  font-size: 11px;
  line-height: 18px;
  white-space: pre-wrap;
}

.service-hub-context-source {
  padding: 11px 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-container);
}

.service-hub-context-source.is-highlighted {
  border-color: var(--td-brand-color-4);
  background: var(--td-brand-color-1);
  box-shadow: inset 3px 0 0 var(--td-brand-color);
}

.service-hub-context-source + .service-hub-context-source {
  margin-top: 8px;
}

.service-hub-context-source-head {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.service-hub-context-source-head strong {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-context-source-badge {
  flex: none;
  color: var(--td-brand-color-7);
  font-size: 10px;
  line-height: 18px;
}

.service-hub-context-source-remove {
  display: inline-flex;
  width: 22px;
  height: 22px;
  align-items: center;
  justify-content: center;
  flex: none;
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-hub-context-source-remove:hover {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-error-color);
}

.service-hub-context-source p {
  display: -webkit-box;
  margin: 7px 0 0;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 17px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.service-hub-context-source-meta {
  display: flex;
  gap: 8px;
  margin-top: 8px;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.service-hub-context-source-actions {
  display: flex;
  gap: 10px;
  margin-top: 9px;
}

.service-hub-context-source-actions button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
  font-size: 11px;
}

.service-hub-context-source-actions button:hover {
  color: var(--td-brand-color-7);
}

.service-hub-reminder-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-reminder-form-row {
  display: grid;
  grid-template-columns: 112px minmax(0, 1fr);
  gap: 8px;
}

.service-hub-reminder-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 12px;
}

.service-hub-reminder-item {
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-container);
}

.service-hub-reminder-item-head,
.service-hub-reminder-meta,
.service-hub-reminder-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.service-hub-reminder-item-head {
  justify-content: space-between;
}

.service-hub-reminder-item-head strong {
  min-width: 0;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  overflow-wrap: anywhere;
}

.service-hub-reminder-item p {
  margin: 6px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 17px;
  overflow-wrap: anywhere;
}

.service-hub-reminder-priority {
  flex: none;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.service-hub-reminder-priority.is-high {
  color: var(--td-error-color);
}

.service-hub-reminder-priority.is-low {
  color: var(--td-text-color-placeholder);
}

.service-hub-reminder-meta {
  margin-top: 7px;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.service-hub-reminder-actions {
  flex-wrap: wrap;
  margin-top: 9px;
}

.service-hub-reminder-actions button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
  font-size: 11px;
}

.service-hub-reminder-actions button:hover {
  color: var(--td-brand-color-7);
}

.service-hub-reminder-actions button.is-danger {
  color: var(--td-error-color);
}

.service-reminder-collaboration {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.service-reminder-collaboration-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-reminder-collaboration-head > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.service-reminder-collaboration-head strong {
  color: var(--td-text-color-primary);
  font-size: 15px;
  line-height: 22px;
  overflow-wrap: anywhere;
}

.service-reminder-collaboration-head span {
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 18px;
}

.service-reminder-collaboration-section {
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.service-reminder-collaboration-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.service-reminder-collaboration-section-head strong,
.service-reminder-collaboration-section > strong {
  color: var(--td-text-color-primary);
  font-size: 12px;
  line-height: 18px;
}

.service-reminder-collaboration-section button,
.service-reminder-comment-compose button,
.service-hub-status-add button,
.service-hub-status-row button,
.service-hub-artifact-links button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
  font-size: 11px;
}

.service-reminder-token-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.service-reminder-token-list span {
  padding: 3px 7px;
  border-radius: 4px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 16px;
}

.service-reminder-comment-list,
.service-reminder-history-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.service-reminder-comment-list article,
.service-reminder-history-list > div {
  padding: 9px 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-reminder-comment-list article > div,
.service-reminder-history-list > div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.service-reminder-comment-list article strong,
.service-reminder-history-list span {
  color: var(--td-text-color-primary);
  font-size: 11px;
  font-weight: 600;
}

.service-reminder-comment-list article small,
.service-reminder-history-list small {
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.service-reminder-comment-list article p {
  margin: 6px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 18px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.service-reminder-comment-compose {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 7px;
}

.service-hub-settings-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.service-hub-settings-form > label {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.service-hub-settings-form > label > span,
.service-hub-settings-divider {
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 18px;
}

.service-hub-settings-divider {
  padding-top: 4px;
  border-top: 1px solid var(--td-component-stroke);
}

.service-hub-status-list {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.service-hub-status-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto auto;
  align-items: center;
  gap: 7px;
}

.service-hub-status-row > span {
  max-width: 130px;
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-status-row > span.is-disabled {
  text-decoration: line-through;
}

.service-hub-status-row button.is-danger,
.service-hub-artifact-links button.is-danger {
  color: var(--td-error-color);
}

.service-hub-profile-field-list {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.service-hub-profile-field-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(80px, 0.8fr) auto auto;
  align-items: center;
  gap: 7px;
}

.service-hub-profile-field-row > span {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-profile-field-required {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-size: 10px;
  white-space: nowrap;
}

.service-hub-profile-field-row button,
.service-hub-profile-field-add button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
  font-size: 11px;
  white-space: nowrap;
}

.service-hub-profile-field-row button.is-danger {
  color: var(--td-error-color);
}

.service-hub-profile-field-add {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
  gap: 7px;
}

.service-hub-status-add {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 7px;
}

.service-hub-status-add .t-select,
.service-hub-status-add button {
  min-width: 0;
}

.service-hub-status-add button {
  min-height: 28px;
  padding: 0 8px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 4px;
}

.service-hub-artifact-detail {
  display: flex;
  flex-direction: column;
  gap: 7px;
  margin-top: 12px;
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-artifact-detail > strong {
  color: var(--td-text-color-primary);
  font-size: 12px;
  line-height: 18px;
  overflow-wrap: anywhere;
}

.service-hub-artifact-detail > p {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 18px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.service-hub-artifact-links {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.service-hub-artifact-links a {
  color: var(--td-brand-color);
  font-size: 11px;
  text-decoration: none;
}

.service-hub-artifact-links a:hover,
.service-reminder-collaboration-section button:hover,
.service-reminder-comment-compose button:hover,
.service-hub-status-add button:hover,
.service-hub-status-row button:hover,
.service-hub-artifact-links button:hover {
  color: var(--td-brand-color-7);
}

.service-hub-subject-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-subject-form-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 8px;
}

.service-hub-subject-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 12px;
}

.service-hub-subject-item {
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-container);
}

.service-hub-subject-item-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.service-hub-subject-item-head strong {
  min-width: 0;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
  overflow-wrap: anywhere;
}

.service-hub-subject-item-head span,
.service-hub-subject-item small {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 18px;
}

.service-hub-subject-item-head span {
  flex: none;
  max-width: 45%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-subject-item small {
  display: block;
  margin-top: 4px;
  overflow-wrap: anywhere;
}

.service-hub-panel-empty {
  padding: 28px 10px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 20px;
  text-align: center;
}

.service-hub-panel-error {
  color: var(--td-error-color);
}

.service-hub-planning-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.service-hub-planning-row,
.service-hub-planning-section {
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
}

.service-hub-planning-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.service-hub-planning-row span,
.service-hub-planning-section span {
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 18px;
}

.service-hub-planning-row strong {
  max-width: 58%;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
  text-align: right;
  word-break: break-word;
}

.service-hub-planning-section > div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}

.service-hub-planning-section > div > strong {
  font-size: 12px;
  font-weight: 500;
}

.service-hub-planning-section p {
  margin: 8px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 19px;
}

.service-hub-chat-state {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 8px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.service-hub-chat-state .t-icon {
  font-size: 24px;
}

.service-hub-chat-state .is-loading {
  animation: service-hub-spin 0.9s linear infinite;
}

.service-blueprint-confirm {
  color: var(--td-text-color-primary);
}

.service-blueprint-confirm-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.service-blueprint-confirm-head > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.service-blueprint-confirm-kicker,
.service-blueprint-confirm-section > span,
.service-blueprint-confirm-grid span {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-blueprint-confirm-head strong {
  overflow: hidden;
  font-size: 18px;
  font-weight: 600;
  line-height: 26px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-blueprint-confirm-status {
  flex: none;
  padding: 3px 8px;
  border-radius: 4px;
  background: var(--td-warning-color-1);
  color: var(--td-warning-color-7);
  font-size: 12px;
  line-height: 18px;
}

.service-blueprint-confirm-copy {
  margin: 14px 0 18px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 21px;
}

.service-blueprint-confirm-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-blueprint-confirm-grid > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.service-blueprint-confirm-grid strong {
  overflow: hidden;
  font-size: 13px;
  font-weight: 500;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-blueprint-confirm-section {
  margin-top: 18px;
}

.service-blueprint-confirm-section > span {
  display: block;
  margin-bottom: 8px;
}

.service-blueprint-confirm-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.service-blueprint-confirm-tags em {
  padding: 3px 8px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 4px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-style: normal;
  line-height: 18px;
}

@keyframes service-hub-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1180px) {
  .service-hub-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 920px) {
  .service-hub-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .service-hub-header-art {
    display: none;
  }

  .service-hub-space-head {
    flex-wrap: wrap;
    padding: 8px 0;
  }

  .service-hub-space-actions {
    width: 100%;
    margin-left: 0;
  }

  .service-hub-workspace-view > .service-hub-space-head + .service-hub-workbench {
    margin-top: 10px;
  }
}

@media (max-width: 680px) {
  .service-hub-list-view,
  .service-hub-create-view,
  .service-hub-workspace-view {
    padding: 16px;
  }

  .service-hub-grid {
    grid-template-columns: 1fr;
  }

  .service-hub-space-template {
    display: none;
  }

  .service-hub-space-actions {
    justify-content: flex-start;
    overflow-x: auto;
    padding-bottom: 2px;
  }

  .service-hub-head-tool {
    flex: none;
  }

  .service-hub-chat-prompts {
    margin-top: 48px;
  }

  .service-hub-chat-prompts > strong {
    font-size: 18px;
    line-height: 26px;
  }

  .service-hub-section-head,
  .service-hub-section-tools,
  .service-hub-form-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .service-hub-search,
  .service-hub-sort {
    width: 100%;
  }

  .service-hub-form-footer > div {
    justify-content: flex-end;
  }

  .service-blueprint-confirm-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
