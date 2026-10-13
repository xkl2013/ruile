<template>
    <div ref="dialogueWrapRef" class="dialogue-wrap" data-page="xiaorui-home">
        <div class="dialogue-answers">
            <section class="xiaorui-notice">
                <div class="xiaorui-notice-icon" aria-hidden="true">
                    <t-icon name="notification" />
                </div>
                <div class="xiaorui-notice-copy">
                    <strong>{{ $t('createChat.home.noticeTitle') }}</strong>
                    <span>{{ $t('createChat.home.noticeDescription') }}</span>
                </div>
                <button type="button" class="xiaorui-notice-action" @click="focusComposer">
                    {{ $t('createChat.home.noticeAction') }}
                    <t-icon name="arrow-right" />
                </button>
            </section>

            <section class="xiaorui-hero">
                <div class="xiaorui-mark-wrap">
                    <div class="xiaorui-mark" aria-label="小睿">
                        <span class="xiaorui-avatar">
                            <img :src="xiaoruiAvatar" alt="" />
                        </span>
                    </div>
                    <div class="xiaorui-date" aria-label="今天日期">
                        <span>{{ dateMonth }}</span>
                        <strong>{{ dateDay }}</strong>
                    </div>
                </div>
                <h1><span>Hi</span>{{ $t('createChat.home.greeting') }}</h1>
                <p>{{ $t('createChat.home.tagline') }}</p>
            </section>

            <section class="xiaorui-experts" aria-labelledby="xiaorui-experts-title">
                <h2 id="xiaorui-experts-title" class="sr-only">{{ $t('createChat.home.expertsTitle') }}</h2>
                <div v-if="featuredAgents.length > 0" class="xiaorui-expert-track">
                    <button v-for="agent in featuredAgents" :key="agent.id" type="button" class="xiaorui-expert-card"
                        :class="{ 'is-selected': agent.id === selectedAgentId }" @click="selectExpert(agent)">
                        <div class="xiaorui-expert-head">
                            <AgentAvatar :name="agent.name" :avatar="agent.avatar" size="small" />
                            <span class="xiaorui-expert-name">
                                <strong>{{ agent.name }}</strong>
                                <small>{{ expertRole(agent) }}</small>
                            </span>
                            <t-icon name="arrow-right" class="xiaorui-expert-arrow" aria-hidden="true" />
                        </div>
                        <p>{{ agent.description || $t('createChat.home.agentDescriptionFallback') }}</p>
                        <span class="xiaorui-expert-tags">
                            <span v-for="tag in agentTags(agent)" :key="tag">{{ tag }}</span>
                        </span>
                    </button>
                </div>
                <div v-else class="xiaorui-expert-empty">
                    <t-icon name="usergroup" />
                    <span>{{ $t('createChat.home.noExperts') }}</span>
                </div>
            </section>

            <div class="xiaorui-composer">
                <InputField ref="inputFieldRef" @send-msg="sendMsg"></InputField>
            </div>
        </div>
    </div>

    <ContextualGuide tour="chat" :when="showChatContextualGuide" />

</template>
<script setup lang="ts">
import { ref, computed, nextTick, onMounted } from 'vue';
import { storeToRefs } from 'pinia';
import ContextualGuide from '@/components/ContextualGuide.vue';
import InputField from '@/components/Input-field.vue';
import AgentAvatar from '@/components/AgentAvatar.vue';
import xiaoruiAvatar from '@/assets/img/shizi.gif';
import { createSessions } from "@/api/chat/index";
import { useMenuStore } from '@/stores/menu';
import { useSettingsStore } from '@/stores/settings';
import { useChatResourcesStore } from '@/stores/chatResources';
import { isBuiltinAgent, BUILTIN_QUICK_ANSWER_ID, BUILTIN_SMART_REASONING_ID, type CustomAgent } from '@/api/agent';
import { useRoute, useRouter } from 'vue-router';
import { MessagePlugin } from 'tdesign-vue-next';
import { useI18n } from 'vue-i18n';

const router = useRouter();
const route = useRoute();
const usemenuStore = useMenuStore();
const settingsStore = useSettingsStore();
const chatResources = useChatResourcesStore();
const { t } = useI18n();
const { agents, disabledOwnAgentIds, publishedExperts } = storeToRefs(chatResources);

const today = new Date();
const dateMonth = `${today.getMonth() + 1}${t('createChat.home.monthSuffix')}`;
const dateDay = `${today.getDate()}`;
const dialogueWrapRef = ref<HTMLElement | null>(null);
const selectedAgentId = computed(() => settingsStore.selectedAgentId);

const isExpertAgent = (agent: CustomAgent) =>
    agent.config?.tags?.includes('expert') === true || (!isBuiltinAgent(agent.id) && !!agent.description);

const publishedExpertForAgent = (agent: CustomAgent) => publishedExperts.value.find((expert) =>
    expert.agent_id === agent.id
    || expert.display_name === agent.name
    || expert.package_display_name === agent.name,
);

const agentWithPublishedAvatar = (agent: CustomAgent): CustomAgent => {
    if (agent.avatar?.trim()) return agent;
    const publishedAvatar = publishedExpertForAgent(agent)?.avatar?.trim();
    return publishedAvatar ? { ...agent, avatar: publishedAvatar } : agent;
};

const featuredAgents = computed(() => {
    const disabled = new Set(disabledOwnAgentIds.value);
    const available = agents.value.filter((agent) => !disabled.has(agent.id));
    const experts = available.filter(isExpertAgent);
    const fallback = available.filter((agent) =>
        agent.id !== BUILTIN_QUICK_ANSWER_ID && agent.id !== BUILTIN_SMART_REASONING_ID,
    );
    const candidates = experts.length > 0 ? experts : fallback;
    const uniqueAgents = new Map<string, CustomAgent>();

    candidates.forEach((agent) => {
        const candidate = agentWithPublishedAvatar(agent);
        const identity = `${candidate.name.trim()}\u0000${(candidate.description || '').trim()}`;
        const existing = uniqueAgents.get(identity);
        if (!existing || (!existing.avatar?.trim() && candidate.avatar?.trim())) {
            uniqueAgents.set(identity, candidate);
        }
    });

    return [...uniqueAgents.values()].slice(0, 8);
});

const expertRole = (agent: CustomAgent) => {
    const tag = agent.config?.tags?.find((item) => item !== 'expert');
    if (tag) return tag;
    return agent.config?.agent_mode === 'smart-reasoning'
        ? t('createChat.home.smartReasoning')
        : t('createChat.home.expertRole');
};

const agentTags = (agent: CustomAgent) => {
    const tags = [...new Set([
        ...(agent.config?.tags || []),
        ...(publishedExpertForAgent(agent)?.tags || []),
    ])].filter((item) => item !== 'expert').slice(0, 3);
    if (tags.length > 0) return tags;
    return [agent.config?.agent_mode === 'smart-reasoning'
        ? t('createChat.home.smartReasoning')
        : t('createChat.home.knowledgeAnswer')];
};

const focusComposer = () => {
    inputFieldRef.value?.focusInput?.();
};

const selectExpert = async (agent: CustomAgent) => {
    settingsStore.selectAgent(agent.id);
    settingsStore.toggleAgent(agent.config?.agent_mode === 'smart-reasoning');

    try {
        const sessionTitle = agent.name?.trim() || t('createChat.newSessionTitle');
        const res = await createSessions({ title: sessionTitle });
        const sessionId = res?.data?.id;
        if (!sessionId) {
            console.error('[createChat] Failed to create empty session for agent');
            MessagePlugin.error(t('createChat.messages.createFailed'));
            return;
        }

        const now = new Date().toISOString();
        usemenuStore.updataMenuChildren({
            title: sessionTitle,
            path: `chat/${sessionId}`,
            id: sessionId,
            isMore: false,
            isNoTitle: false,
            created_at: now,
            updated_at: now,
        });
        usemenuStore.changeIsFirstSession(false);
        usemenuStore.changeFirstQuery('', [], '', [], []);
        usemenuStore.changeFirstPublishedExpert(null);
        usemenuStore.changeFirstPublishedExpertRouteMode('manual');

        await router.push({
            path: `/platform/chat/${sessionId}`,
            query: { agent_id: agent.id },
        });
    } catch (error) {
        console.error('[createChat] Create empty session for agent error:', error);
        MessagePlugin.error(t('createChat.messages.createError'));
    }
};

const showChatContextualGuide = computed(() => {
    return route.name === 'globalCreatChat' || route.name === 'kbCreatChat';
});

onMounted(() => {
    nextTick(() => {
        dialogueWrapRef.value?.scrollTo({ top: 0, left: 0 });
    });
});

const inputFieldRef = ref();

const sendMsg = (value: string, modelId: string, mentionedItems: any[], imageFiles: any[] = [], attachmentFiles: any[] = []) => {
    createNewSession(value, modelId, mentionedItems, imageFiles, attachmentFiles);
}

async function createNewSession(
    value: string,
    modelId: string,
    mentionedItems: any[] = [],
    imageFiles: any[] = [],
    attachmentFiles: any[] = [],
    publishedExpert: any | null = null,
    publishedExpertRouteMode: 'auto' | 'manual' = 'manual',
) {
    // Published expert runs are tracked by AgentRun and rendered locally in
    // this first slice, so they do not depend on normal session creation.
    if (publishedExpert || publishedExpertRouteMode === 'auto') {
        const temporarySessionId = `expert-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
        await navigateToSession(
            temporarySessionId,
            value,
            modelId,
            mentionedItems,
            imageFiles,
            attachmentFiles,
            publishedExpert,
            publishedExpertRouteMode,
        );
        return;
    }

    const selectedKbs = settingsStore.settings.selectedKnowledgeBases || [];
    const selectedFiles = settingsStore.settings.selectedFiles || [];

    // 构建 session 数据，包含 Agent 配置
    const sessionData: any = {};

    // 添加 Agent 配置（知识库信息在 agent_config 中）
    sessionData.agent_config = {
        enabled: true,
        max_iterations: settingsStore.agentConfig.maxIterations,
        temperature: settingsStore.agentConfig.temperature,
        knowledge_bases: selectedKbs,  // 所有选中的知识库
        knowledge_ids: selectedFiles,  // 所有选中的普通知识/文件
        allowed_tools: settingsStore.agentConfig.allowedTools
    };

    try {
        const res = await createSessions(sessionData);
        if (res.data && res.data.id) {
            await navigateToSession(res.data.id, value, modelId, mentionedItems, imageFiles, attachmentFiles);
        } else {
            console.error('[createChat] Failed to create session');
            MessagePlugin.error(t('createChat.messages.createFailed'));
        }
    } catch (error) {
        console.error('[createChat] Create session error:', error);
        MessagePlugin.error(t('createChat.messages.createError'));
    }
}

const navigateToSession = async (
    sessionId: string,
    value: string,
    modelId: string,
    mentionedItems: any[],
    imageFiles: any[] = [],
    attachmentFiles: any[] = [],
    publishedExpert: any | null = null,
    publishedExpertRouteMode: 'auto' | 'manual' = 'manual',
) => {
    const now = new Date().toISOString();
    let obj = {
        title: t('createChat.newSessionTitle'),
        path: `chat/${sessionId}`,
        id: sessionId,
        isMore: false,
        isNoTitle: true,
        created_at: now,
        updated_at: now
    };
    usemenuStore.updataMenuChildren(obj);
    usemenuStore.changeIsFirstSession(true);
    usemenuStore.changeFirstQuery(value, mentionedItems, modelId, imageFiles, attachmentFiles);
    usemenuStore.changeFirstPublishedExpert(publishedExpert);
    usemenuStore.changeFirstPublishedExpertRouteMode(publishedExpertRouteMode);
    router.push(`/platform/chat/${sessionId}`);
}

</script>
<style lang="less" scoped>
.dialogue-wrap {
    flex: 1;
    display: flex;
    justify-content: center;
    min-height: 0;
    overflow: hidden;
    scrollbar-width: none;
    -ms-overflow-style: none;
    position: relative;
    background: var(--td-bg-color-container, #fff);

    &::-webkit-scrollbar {
        display: none;
        width: 0;
        height: 0;
    }
}

.dialogue-answers {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    width: 100%;
    max-width: 960px;
    height: 100%;
    min-height: 0;
    box-sizing: border-box;
    padding: 16px 24px 20px;

    :deep(.answers-input) {
        position: static;
        transform: none;
        width: 100%;
    }
}

.xiaorui-notice {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    max-width: 768px;
    min-height: 52px;
    margin: 0 auto;
    padding: 10px 12px 10px 10px;
    border: 1px solid #c6ecd5;
    border-radius: 9px;
    background: #f0fbf4;
}

.xiaorui-notice-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    flex: 0 0 32px;
    border: 1px solid #d6f0df;
    border-radius: 7px;
    background: #fff;
    color: var(--td-brand-color, #07c05f);
}

.xiaorui-notice-copy {
    display: flex;
    flex-direction: column;
    min-width: 0;
    flex: 1;
    line-height: 1.5;

    strong {
        color: var(--td-text-color-primary, rgba(0, 0, 0, .9));
        font-size: 13px;
        font-weight: 600;
    }

    span {
        margin-top: 1px;
        color: var(--td-text-color-secondary, rgba(0, 0, 0, .6));
        font-size: 11.5px;
    }
}

.xiaorui-notice-action {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    flex: 0 0 auto;
    height: 28px;
    padding: 0 10px;
    border: 1px solid #bfe7ce;
    border-radius: 6px;
    background: #fff;
    color: var(--td-brand-color, #07c05f);
    font-size: 12px;
    font-weight: 600;
    transition: background .15s, border-color .15s;

    &:hover {
        border-color: var(--td-brand-color, #07c05f);
        background: #e8f8ed;
    }
}

.xiaorui-hero {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    margin: 18px auto 12px;
}

.xiaorui-mark-wrap {
    position: relative;
    display: inline-flex;
    padding-right: 52px;
}

.xiaorui-mark {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 108px;
    height: 108px;
}

.xiaorui-avatar {
    position: relative;
    z-index: 1;
    display: block;
    box-sizing: border-box;
    width: 100px;
    height: 100px;
    overflow: hidden;
    border: 2px solid #d4d0f7;
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 2px 6px rgba(0, 0, 0, .1);

    img {
        display: block;
        width: 100%;
        height: 100%;
        object-fit: cover;
    }
}

.xiaorui-date {
    position: absolute;
    top: 0;
    right: 0;
    display: flex;
    flex-direction: column;
    width: 44px;
    overflow: hidden;
    border: 1px solid var(--td-component-stroke, #e7e7e7);
    border-radius: 8px;
    background: #fff;
    box-shadow: 0 2px 8px rgba(0, 0, 0, .06);
    text-align: center;

    span {
        padding: 2px 0;
        background: var(--td-brand-color, #07c05f);
        color: #fff;
        font-size: 10px;
        font-weight: 600;
        line-height: 1.35;
    }

    strong {
        padding: 3px 0 4px;
        color: var(--td-text-color-primary, rgba(0, 0, 0, .9));
        font-size: 16px;
        font-weight: 600;
        line-height: 1.25;
    }
}

.xiaorui-hero h1 {
    margin: 20px 0 0;
    color: var(--td-text-color-primary, rgba(0, 0, 0, .9));
    font-size: 22px;
    font-weight: 600;
    line-height: 1.4;

    span {
        margin-right: 5px;
        color: var(--td-brand-color, #07c05f);
    }
}

.xiaorui-hero p {
    max-width: 540px;
    margin: 8px 0 0;
    color: var(--td-text-color-secondary, rgba(0, 0, 0, .6));
    font-size: 13px;
    line-height: 1.75;
}

.xiaorui-experts {
    width: 100%;
    max-width: 768px;
    margin: 0 auto;
}

.xiaorui-expert-track {
    display: flex;
    gap: 12px;
    width: 100%;
    overflow-x: auto;
    padding: 2px 1px 6px;
    scrollbar-width: none;

    &::-webkit-scrollbar {
        display: none;
    }
}

.xiaorui-expert-card {
    display: flex;
    flex-direction: column;
    width: 224px;
    min-width: 224px;
    min-height: 146px;
    padding: 12px;
    border: 1px solid var(--td-component-stroke, #e7e7e7);
    border-radius: 9px;
    background: #fff;
    color: inherit;
    text-align: left;
    transition: border-color .15s, box-shadow .15s, transform .15s;

    &:hover,
    &.is-selected {
        border-color: #9bdfb2;
        box-shadow: 0 4px 12px rgba(14, 129, 61, .1);
        transform: translateY(-1px);
    }

    &.is-selected {
        background: #fbfffc;
    }
}

.xiaorui-expert-head {
    display: flex;
    align-items: center;
    gap: 9px;
    min-width: 0;
}

.xiaorui-expert-name {
    display: flex;
    flex-direction: column;
    min-width: 0;
    flex: 1;

    strong {
        overflow: hidden;
        color: var(--td-text-color-primary, rgba(0, 0, 0, .9));
        font-size: 13px;
        font-weight: 600;
        line-height: 1.35;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    small {
        margin-top: 1px;
        overflow: hidden;
        color: var(--td-text-color-placeholder, rgba(0, 0, 0, .4));
        font-size: 10.5px;
        line-height: 1.35;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
}

.xiaorui-expert-arrow {
    flex: 0 0 auto;
    color: var(--td-text-color-placeholder, rgba(0, 0, 0, .4));
    font-size: 13px;
}

.xiaorui-expert-card p {
    display: -webkit-box;
    min-height: 38px;
    margin: 12px 0 10px;
    overflow: hidden;
    color: var(--td-text-color-secondary, rgba(0, 0, 0, .6));
    font-size: 11.5px;
    line-height: 1.65;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
}

.xiaorui-expert-tags {
    display: flex;
    gap: 5px;
    margin-top: auto;
    padding-top: 9px;
    border-top: 1px solid var(--td-bg-color-secondarycontainer, #f3f3f3);
    overflow: hidden;

    span {
        max-width: 96px;
        overflow: hidden;
        padding: 2px 7px;
        border-radius: 999px;
        background: #e9f8ed;
        color: #159b45;
        font-size: 10px;
        line-height: 1.45;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
}

.xiaorui-expert-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    min-height: 48px;
    border: 1px dashed var(--td-component-stroke, #dcdcdc);
    border-radius: 9px;
    color: var(--td-text-color-placeholder, rgba(0, 0, 0, .4));
    font-size: 12px;
}

.xiaorui-composer {
    width: 100%;
    max-width: 768px;
    margin: auto auto 100px;

    :deep(.rich-input-container) {
        max-width: none;
        border-radius: 12px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, .04), 0 7px 18px rgba(0, 0, 0, .04);
    }

    :deep(.t-textarea__inner) {
        min-height: 96px !important;
        max-height: 160px !important;
        padding: 14px 16px 52px;
        font-size: 14px;
    }

    :deep(.control-bar) {
        bottom: 10px;
        left: 12px;
        right: 12px;
    }
}

.xiaorui-notice-action :deep(.t-icon),
.xiaorui-expert-arrow {
    font-size: 13px;
}

.sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
}

@media (max-width: 720px) {
    .dialogue-answers {
        padding: 14px 14px 18px;
    }

    .xiaorui-notice {
        align-items: flex-start;
    }

    .xiaorui-notice-action {
        align-self: center;
        padding: 0 8px;
    }

    .xiaorui-hero {
        margin-top: 16px;
    }

    .xiaorui-expert-track {
        margin-right: -14px;
        padding-right: 14px;
    }

    .xiaorui-composer {
        margin-top: auto;
    }
}

@media (max-height: 760px) {
    .dialogue-answers {
        padding-top: 10px;
        padding-bottom: 12px;
    }

    .xiaorui-notice {
        min-height: 48px;
    }

    .xiaorui-hero {
        margin-top: 10px;
        margin-bottom: 8px;
    }

    .xiaorui-hero h1 {
        margin-top: 12px;
    }

    .xiaorui-hero p {
        margin-top: 4px;
        line-height: 1.5;
    }

    .xiaorui-expert-card {
        min-height: 124px;
        padding: 10px;
    }

    .xiaorui-expert-card p {
        margin-top: 8px;
        margin-bottom: 6px;
    }

    .xiaorui-expert-tags {
        padding-top: 6px;
    }

    .xiaorui-expert-empty {
        min-height: 44px;
    }

    .xiaorui-composer {
        margin-top: auto;

        :deep(.t-textarea__inner) {
            min-height: 82px !important;
            padding-top: 10px;
            padding-bottom: 46px;
        }
    }
}

@media (max-width: 430px) {
    .xiaorui-notice-copy span {
        max-width: 170px;
    }

    .xiaorui-notice-action {
        font-size: 11px;
    }
}
</style>
<style lang="less">
.del-menu-popup {
    z-index: 99 !important;

    .t-popup__content {
        width: 100px;
        height: 40px;
        line-height: 30px;
        padding-left: 14px;
        cursor: pointer;
        margin-top: 4px !important;

    }
}
</style>
