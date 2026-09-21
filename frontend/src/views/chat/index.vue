<template>
    <div class="chat" :class="{
        'is-embedded': embeddedMode,
        'is-sidebar-collapsed': uiStore.sidebarCollapsed,
        'has-references-panel': referencesDrawerVisible,
        'has-expert-artifact-panel': activeExpertArtifact,
    }">
        <ChatHeader v-if="!embeddedMode" :session="currentSession" :has-references-panel="referencesDrawerVisible" />
        <div ref="scrollContainer" class="chat_scroll_box" @scroll="handleScroll">
            <div class="msg_list" :class="{ 'is-embedded': embeddedMode }">
                <!-- 消息列表骨架屏 -->
                <div v-if="historyLoading && messagesList.length === 0" class="msg-skeleton-list">
                    <div class="msg-skeleton msg-skeleton-user">
                        <t-skeleton animation="gradient" :row-col="[{ width: '45%', height: '36px', type: 'rect' }]" />
                    </div>
                    <div class="msg-skeleton msg-skeleton-bot">
                        <t-skeleton animation="gradient"
                            :row-col="[{ width: '80%', height: '16px' }, { width: '100%', height: '16px' }, { width: '60%', height: '16px' }]" />
                    </div>
                    <div class="msg-skeleton msg-skeleton-user">
                        <t-skeleton animation="gradient" :row-col="[{ width: '35%', height: '36px', type: 'rect' }]" />
                    </div>
                    <div class="msg-skeleton msg-skeleton-bot">
                        <t-skeleton animation="gradient"
                            :row-col="[{ width: '70%', height: '16px' }, { width: '90%', height: '16px' }]" />
                    </div>
                </div>
                <!-- 推荐问题卡片 - 仅在新会话（无消息）时展示 -->
                <div v-if="!embeddedMode && messagesList.length === 0 && !loading" class="suggested-questions-container"
                    :class="{ 'has-questions': suggestedQuestions.length > 0 || suggestedQuestionsLoading }">
                    <!-- 骨架屏占位 -->
                    <div v-if="suggestedQuestionsLoading && suggestedQuestions.length === 0"
                        class="suggested-questions-inner">
                        <div class="suggested-questions-title"><t-skeleton animation="gradient"
                                :row-col="[{ width: '120px', height: '14px' }]" /></div>
                        <div class="suggested-questions-grid">
                            <div v-for="n in 6" :key="'sq-skel-' + n" class="suggested-question-card sq-card-skeleton">
                                <t-skeleton animation="gradient"
                                    :row-col="[{ width: '100%', height: '14px', type: 'rect' }]" />
                            </div>
                        </div>
                    </div>
                    <transition v-else appear name="sq-fade">
                        <div v-if="suggestedQuestions.length > 0" class="suggested-questions-inner">
                            <div class="suggested-questions-title-row">
                                <p class="suggested-questions-caption">
                                    <span class="suggested-questions-title">{{ t('chat.suggestedQuestions') }}</span>
                                    <button type="button" class="suggested-questions-refresh"
                                        :disabled="suggestedQuestionsLoading"
                                        :title="t('chat.refreshSuggestedQuestions')"
                                        :aria-label="t('chat.refreshSuggestedQuestions')"
                                        @click="fetchSuggestedQuestions">
                                        <t-icon :name="suggestedQuestionsLoading ? 'loading' : 'refresh'"
                                            :class="{ 'sq-refresh-spin': suggestedQuestionsLoading }" />
                                    </button>
                                </p>
                            </div>
                            <div class="suggested-questions-grid">
                                <div v-for="(item, index) in suggestedQuestions" :key="item.question"
                                    class="suggested-question-card"
                                    @click="handleSuggestedQuestionClick(item.question)">
                                    <span class="suggested-question-text">{{ item.question }}</span>
                                    <span v-if="item.source === 'faq'" class="suggested-question-badge faq">FAQ</span>
                                </div>
                            </div>
                        </div>
                    </transition>
                </div>
                <div
                    v-if="embeddedMode && messagesList.length === 0 && !historyLoading && !loading && $slots['empty-suggestions']"
                    class="chat-empty-suggestions"
                >
                    <slot name="empty-suggestions" />
                </div>
                <!--
                  关键：必须用 session.id 作为 key，不能用 v-for 的索引。
                  向上滚动加载历史时会插入一批消息（push/unshift）到列表，
                  若用索引作 key 会让所有已渲染消息的 key 漂移，触发整个列表的销毁重建
                  （botmsg / AgentStreamDisplay 全部重新挂载、markdown 重新渲染），
                  这是历史加载时白屏 + layout shift 蔓延到 session 列表的根因。
                  仅对极少数尚未拿到 id 的本地占位消息 fallback 到 role+created_at+index。
                -->
                <div v-for="(session, index) in messagesList"
                    :key="session.id || `${session.role}-${session.created_at}-${index}`" class="msg-item-wrapper">

                    <div v-if="session.role == 'user'">
                        <usermsg :content="session.content" :mentioned_items="session.mentioned_items"
                            :images="session.images" :attachments="session.attachments" :embeddedMode="embeddedMode"
                            :session-id="session_id">
                        </usermsg>
                    </div>
                    <div v-if="session.role == 'assistant' && session.isPublishedExpertRun">
                        <PublishedExpertRunMessage :message="session"
                            @open-artifact="openExpertArtifact"
                            @submit-answers="(answers) => submitPublishedExpertAnswers(session, answers)"
                            @regenerate="(action) => regeneratePublishedExpertRun(session, action)"
                            @ask-follow-up="(action) => askPublishedExpertFollowUp(session, action)"
                            @cancel="cancelPublishedExpertRun(session)"
                            @open-diff="openExpertRunDiff(session)" />
                    </div>
                    <div v-else-if="session.role == 'assistant' && session.isPublishedExpertFollowUp">
                        <PublishedExpertFollowUpMessage :message="session" />
                    </div>
                    <div v-else-if="session.role == 'assistant' && session.isPublishedExpertRouteChoice">
                        <PublishedExpertRouteChoiceMessage
                            :message="session"
                            @select="(candidate) => confirmPublishedExpertRoute(session, candidate)"
                            @cancel="cancelPublishedExpertRoute(session)" />
                    </div>
                    <div v-else-if="session.role == 'assistant' && shouldRenderAssistantMessage(session)">
                        <botmsg :content="session.content" :session="session" :session-id="session_id"
                            :user-query="getUserQuery(index)" @scroll-bottom="scrollToBottom"
                            :isFirstEnter="isFirstEnter" :embeddedMode="embeddedMode"
                            :follow-up-loading="Boolean(session.suggestionLoading && !session.suggestionSet?.questions?.length)"
                            @render-complete-change="(ready) => handleAnswerRenderComplete(session, ready)">
                        </botmsg>
                        <FollowUpSuggestions v-if="session.answerFullyRendered && !session.suggestionsDismissed"
                            :suggestion-set="session.suggestionSet"
                            :loading="session.suggestionLoading"
                            :allow-regenerate="session.suggestionSet?.allow_regenerate"
                            @select="(item) => handleFollowUpSelect(session, item)"
                            @regenerate="loadFollowUpSuggestions(session, true, true)"
                            @impression="(set) => recordSuggestionEvent(session, set, 'impression')"
                            @dismiss="(set) => dismissSuggestions(session, set)" />
                    </div>
                </div>
                <div v-if="showGlobalTypingIndicator" class="chat-global-wait" role="status"
                    :aria-label="t('chat.thinkingAlt')">
                    <span class="chat-global-wait__spinner" aria-hidden="true"></span>
                </div>
            </div>
        </div>
        <transition name="scroll-btn-fade">
            <div v-show="userHasScrolledUp" class="scroll-to-bottom-btn" @click="onClickScrollToBottom">
                <t-icon name="chevron-down" size="20px" />
            </div>
        </transition>
        <div class="input-container" :class="{ 'is-embedded': embeddedMode }">
            <InputField ref="inputFieldRef"
                @send-msg="(query, modelId, mentionedItems, imageFiles, attachmentFiles, responseTier) => sendMsg(query, modelId, mentionedItems, imageFiles, attachmentFiles, responseTier)"
                @send-published-expert="handlePublishedExpertSend"
                @stop-generation="handleStopGeneration" :isReplying="isReplying" :sessionId="session_id"
                :assistantMessageId="currentAssistantMessageId" :agent-id="agentId"
                :placeholder="embeddedInputPlaceholder" :embeddedMode="embeddedMode"></InputField>
        </div>
    </div>
    <ChatReferencesDrawer />
    <ChatAttachmentPreviewDrawer />
    <ExpertArtifactPanel v-if="activeExpertArtifact" :artifact="activeExpertArtifact" :run-id="activeExpertArtifactRunId"
        @close="activeExpertArtifact = null" />
    <ExpertRunDiffPanel v-if="activeExpertRunDiff" :diff="activeExpertRunDiff" :loading="expertRunDiffLoading"
        @close="activeExpertRunDiff = null" />
</template>
<script setup>
import { storeToRefs } from 'pinia';
import { ref, onMounted, onBeforeMount, onUnmounted, nextTick, watch, reactive, computed } from 'vue';
import { useRoute, onBeforeRouteLeave, onBeforeRouteUpdate } from 'vue-router';
import InputField from '../../components/Input-field.vue';
import botmsg from './components/botmsg.vue';
import usermsg from './components/usermsg.vue';
import PublishedExpertRunMessage from './components/PublishedExpertRunMessage.vue';
import PublishedExpertFollowUpMessage from './components/PublishedExpertFollowUpMessage.vue';
import PublishedExpertRouteChoiceMessage from './components/PublishedExpertRouteChoiceMessage.vue';
import ExpertArtifactPanel from './components/ExpertArtifactPanel.vue';
import ExpertRunDiffPanel from './components/ExpertRunDiffPanel.vue';
import { getMessageList, getSession } from "@/api/chat/index";
import { getSuggestedQuestions } from "@/api/agent/index";
import { deleteTemporaryAttachment, uploadTemporaryAttachment } from '@/api/chat/temporary-attachments';
import { useStream } from '../../api/chat/streame'
import { useMenuStore } from '@/stores/menu';
import { useSettingsStore } from '@/stores/settings';
import { MessagePlugin } from 'tdesign-vue-next';
import { useI18n } from 'vue-i18n';
import { useUIStore } from '@/stores/ui';
import { useChatStreamHandler } from '@/composables/useChatStreamHandler';
import { useStickyBottomOnResize } from '@/composables/useStickyBottomOnResize';
import { clearCitationChunkCache } from '@/utils/citationChunkCache';
import ChatReferencesDrawer from '@/components/ChatReferencesDrawer.vue';
import ChatAttachmentPreviewDrawer from '@/components/ChatAttachmentPreviewDrawer.vue';
import FollowUpSuggestions from '@/components/chat/FollowUpSuggestions.vue';
import ChatHeader from '@/components/ChatHeader.vue';
import {
    notifySessionMutation,
    SESSION_MUTATION_EVENT,
} from '@/components/sessionMutations';
import {
    ensureMessageSuggestions,
    getMessageSuggestions,
    recordMessageSuggestionEvent,
} from '@/api/message-suggestion';
import { provideChatReferencesDrawer } from '@/composables/useChatReferencesDrawer';
import { provideChatAttachmentPreviewDrawer } from '@/composables/useChatAttachmentPreviewDrawer';
import {
    cancelAgentRun,
    getAgentRun,
    getAgentRunDiff,
    listAgentRunSteps,
    regenerateAgentRun,
    streamAgentRunEvents,
    submitAgentRunAnswers,
} from '@/api/agent-run';
import {
    routePublishedExpert,
    runPublishedExpert,
    runPublishedExpertAuto,
    runPublishedExpertFollowUp,
} from '@/api/expert-package';

const referencesDrawer = provideChatReferencesDrawer();
provideChatAttachmentPreviewDrawer();
const { visible: referencesDrawerVisible } = referencesDrawer;

const props = defineProps({
    session_id: { type: String, default: '' },
    agentId: { type: String, default: '' },
    kbIds: { type: Array, default: () => [] },
    quotedContext: { type: String, default: '' },
    embeddedInputPlaceholder: { type: String, default: '' },
    embeddedMode: { type: Boolean, default: false },
});
const emit = defineEmits(['message-state-change', 'user-message-send']);

const usemenuStore = useMenuStore();
const useSettingsStoreInstance = useSettingsStore();

// Whether the active chat session is using the Agent pipeline (not quick-answer).
const isAgentStreamSession = () => {
    if (props.embeddedMode) {
        return !!(props.agentId && props.agentId !== 'builtin-quick-answer');
    }
    return useSettingsStoreInstance.isAgentStreamMode;
};

const uiStore = useUIStore();
const { t } = useI18n();
const {
    firstQuery,
    firstMentionedItems,
    firstModelId,
    firstImageFiles,
    firstAttachmentFiles,
    firstPublishedExpert,
    firstPublishedExpertRouteMode,
} = storeToRefs(usemenuStore);
const { onChunk, error, startStream, stopStream, lastStreamRequest } = useStream();
/** Snapshot of the in-flight HTTP request for attaching to the next assistant message. */
const pendingStreamDebug = ref(null);

const buildStreamDebugPayload = () => {
    const meta = lastStreamRequest.value;
    if (!meta) return null;
    return {
        requestId: meta.requestId,
        url: meta.url,
        method: meta.method,
        body: meta.body,
        sentAt: meta.sentAt,
        sessionId: session_id.value,
    };
};

const attachStreamDebugToMessage = (message) => {
    if (!message) return;
    const payload = pendingStreamDebug.value || buildStreamDebugPayload();
    if (!payload) return;
    if (payload.requestId && !message.request_id) {
        message.request_id = payload.requestId;
    }
    message.debugRequest = payload;
};
const route = useRoute();
const session_id = ref(props.session_id || route.params.chatid);
const currentSession = ref(null);
const isPublishedExpertSessionId = (sessionId = session_id.value) =>
    /^expert-\d+-[a-z0-9]+$/i.test(String(sessionId || '').trim());

// 拉 session 详情，并按其 last_request_state 把输入栏状态恢复到当时的发起态。
// 嵌入式（embeddedMode）由宿主页面注入 agent/KB，所以跳过整套恢复逻辑，
// 避免污染宿主的 settings store。
const loadSessionAndHydrate = async (sid) => {
    if (!sid || props.embeddedMode || isPublishedExpertSessionId(sid)) return;
    try {
        const sessionRes = await getSession(sid);
        if (sessionRes?.data && sid === session_id.value) {
            currentSession.value = sessionRes.data;
            const lastState = sessionRes.data.last_request_state;
            if (lastState) {
                // 先把当前的"全局默认"快照下来，再用 session 状态覆盖；
                // 离开会话时会从快照还原，避免本会话的状态污染新建对话。
                useSettingsStoreInstance.snapshotAsDefaultsIfNeeded();
                useSettingsStoreInstance.applyLastRequestState(lastState);
            }
        }
    } catch (error) {
        console.error('Failed to load session data:', error);
    }
};
const inputFieldRef = ref();
const created_at = ref('');
const limit = ref(20);
const messagesList = reactive([]);
const emitMessageStateChange = () => {
    emit('message-state-change', {
        sessionId: session_id.value,
        messageCount: messagesList.length,
        hasMessages: messagesList.length > 0,
    });
};
const isReplying = ref(false);
const currentAssistantMessageId = ref(''); // 当前正在生成的 assistant message ID
const activeExpertArtifact = ref(null);
const activeExpertArtifactRunId = ref('');
const activeExpertRunDiff = ref(null);
const expertRunDiffLoading = ref(false);
// True only while attaching to an in-flight *IM-originated* reply via continue-stream.
// Such replies are generated on the IM side and never stream through this server, so
// continue-stream always fails even though the answer is coming — recover by polling
// instead of erroring. Web/api replies are left on the original error path.
const isAttachingImStream = ref(false);
let recoverPollTimer = null;
// True while polling to recover an in-flight IM reply we couldn't stream. Drives
// the same "generating" typing indicator the normal reply path shows, so the wait
// isn't a silent gap. IM-only: false everywhere else, so other flows are unchanged.
const isImRecovering = ref(false);
const scrollLock = ref(false);
const isFirstEnter = ref(true);
const loading = ref(false);
const historyLoading = ref(true);
const historyLoadingMore = ref(false);
const hasMoreHistory = ref(true);
let fullContent = ref('')
const scrollContainer = ref(null)
const userHasScrolledUp = ref(false)
const SCROLL_BOTTOM_THRESHOLD = 80

const isNearBottom = () => {
    if (!scrollContainer.value) return true;
    const { scrollTop, scrollHeight, clientHeight } = scrollContainer.value;
    return scrollHeight - scrollTop - clientHeight < SCROLL_BOTTOM_THRESHOLD;
}

// ===== 推荐问题 =====
const suggestedQuestions = ref([]);
const suggestedQuestionsLoading = ref(false);
let suggestedQuestionsFetchId = 0; // 用于取消过时的请求
let suggestedDebounceTimer = null;
let pendingSuggestionAttribution = null;
let pendingSuggestionKnowledgeBaseIds = [];

const cancelSuggestedQuestionsFetch = () => {
    suggestedQuestionsFetchId++;
    suggestedQuestionsLoading.value = false;
    suggestedQuestions.value = [];
    if (suggestedDebounceTimer) {
        clearTimeout(suggestedDebounceTimer);
        suggestedDebounceTimer = null;
    }
};

const fetchSuggestedQuestionsIfNeeded = async () => {
    if (props.embeddedMode) return;
    // 初始历史尚未拉完时不能判断是否有消息，避免有历史的会话误请求推荐问法
    if (historyLoading.value || messagesList.length > 0) {
        if (messagesList.length > 0) {
            cancelSuggestedQuestionsFetch();
        }
        return;
    }
    await fetchSuggestedQuestions();
};

const fetchSuggestedQuestions = async () => {
    if (historyLoading.value || messagesList.length > 0) {
        return;
    }
    const fetchId = ++suggestedQuestionsFetchId;
    suggestedQuestionsLoading.value = true;
    // 加载期间保留旧数据，不清空，避免布局抖动
    try {
        const agentId = useSettingsStoreInstance.selectedAgentId;
        if (!agentId) return;
        const res = await getSuggestedQuestions(agentId, useSettingsStoreInstance.getSuggestedQuestionsParams(6));
        if (fetchId === suggestedQuestionsFetchId) {
            suggestedQuestions.value = res?.data?.questions || [];
        }
    } catch (err) {
        console.warn('[SuggestedQuestions] Failed to fetch:', err);
        if (fetchId === suggestedQuestionsFetchId) {
            suggestedQuestions.value = [];
        }
    } finally {
        if (fetchId === suggestedQuestionsFetchId) {
            suggestedQuestionsLoading.value = false;
        }
    }
};

const handleSuggestedQuestionClick = (question) => {
    if (inputFieldRef.value?.triggerSend) {
        inputFieldRef.value.triggerSend(question);
    } else {
        sendMsg(question);
    }
};

const resolveAssistantMessageId = (message) => message?.id || message?.assistant_message_id;

const handleAnswerRenderComplete = (message, ready) => {
    message.answerFullyRendered = Boolean(ready);
};

const loadFollowUpSuggestions = async (message, ensure = false, regenerate = false) => {
    const messageId = resolveAssistantMessageId(message);
    const targetSessionId = session_id.value;
    if (!messageId || !targetSessionId || message.suggestionsDismissed) return;
    message.suggestionLoading = true;
    try {
        let response = ensure
            ? await ensureMessageSuggestions(targetSessionId, messageId, regenerate)
            : await getMessageSuggestions(targetSessionId, messageId);
        let set = response?.data;
        for (let attempt = 0; set?.status === 'generating' && attempt < 120; attempt++) {
            await new Promise((resolve) => setTimeout(resolve, 1000));
            if (session_id.value !== targetSessionId || message.suggestionsDismissed) return;
            response = await getMessageSuggestions(targetSessionId, messageId);
            set = response?.data;
        }
        message.suggestionSet = set?.status === 'ready' ? set : null;
    } catch (error) {
        if (ensure) console.warn('[FollowUpSuggestions] Failed to generate:', error);
        message.suggestionSet = null;
    } finally {
        message.suggestionLoading = false;
    }
};

const recordSuggestionEvent = (message, set, eventType, questionId = '') => {
    if (!set?.id) return;
    void recordMessageSuggestionEvent(session_id.value, set.id, eventType, questionId).catch(() => undefined);
};

const handleFollowUpSelect = (message, item) => {
    recordSuggestionEvent(message, message.suggestionSet, 'click', item.id);
    pendingSuggestionAttribution = {
        suggestion_set_id: message.suggestionSet.id,
        question_id: item.id,
    };
    // Knowledge-backed follow-ups are generated from a specific KB. Keep that
    // authorized retrieval anchor for the immediate next request; model-backed
    // suggestions intentionally do not inherit transient @file/@tag/MCP/Skill scope.
    pendingSuggestionKnowledgeBaseIds = [...new Set(item.knowledge_base_ids || [])];
    if (inputFieldRef.value?.triggerSend) inputFieldRef.value.triggerSend(item.text);
    else sendMsg(item.text);
};

const dismissSuggestions = (message, set) => {
    message.suggestionsDismissed = true;
    recordSuggestionEvent(message, set, 'dismiss');
};

// 防抖包装，切换知识库/文件时300ms内不重复请求
const debouncedFetchSuggestions = () => {
    if (historyLoading.value || messagesList.length > 0) return;
    if (suggestedDebounceTimer) clearTimeout(suggestedDebounceTimer);
    suggestedDebounceTimer = setTimeout(() => { fetchSuggestedQuestionsIfNeeded(); }, 300);
};

// 监听 Agent / 知识库 / 文件 / 标签 / MCP / Skill @mention，重新获取推荐问题
watch(
    () => ({
        agentId: useSettingsStoreInstance.selectedAgentId,
        kbs: useSettingsStoreInstance.settings.selectedKnowledgeBases,
        files: useSettingsStoreInstance.settings.selectedFiles,
        tags: useSettingsStoreInstance.settings.selectedTags,
        mcps: useSettingsStoreInstance.settings.selectedMCPServices,
        skills: useSettingsStoreInstance.settings.selectedSkills,
    }),
    debouncedFetchSuggestions,
    { deep: true },
);

function fileToBase64(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(reader.result);
        reader.onerror = reject;
        reader.readAsDataURL(file);
    });
}

const getUserQuery = (index) => {
    if (index <= 0) {
        return '';
    }
    const previous = messagesList[index - 1];
    if (previous && previous.role === 'user') {
        return previous.content || '';
    }
    return '';
};

watch([() => route.params], async (newvalue) => {
    isFirstEnter.value = true;
    if (newvalue[0].chatid) {
        if (!firstQuery.value) {
            scrollLock.value = false;
        }
        messagesList.splice(0);
        session_id.value = newvalue[0].chatid;
        currentSession.value = null;
        clearCitationChunkCache();

        // 切换会话时，重置状态
        historyLoading.value = true;
        historyLoadingMore.value = false;
        hasMoreHistory.value = true;
        created_at.value = '';
        loading.value = false;
        isReplying.value = false;
        currentAssistantMessageId.value = '';
        userHasScrolledUp.value = false;

        // 跨会话切换：先把旧会话覆盖前的全局默认还原，再让新会话重新拍快照
        // 并应用自己的 last_request_state（在 loadSessionAndHydrate 内部完成）。
        useSettingsStoreInstance.restoreDefaultsIfSnapshotted();

        await loadSessionAndHydrate(session_id.value);
        let data = {
            session_id: session_id.value,
            created_at: '',
            limit: limit.value
        }
        await getmsgList(data);
        await restorePublishedExpertRuns();
    }
});
const scrollToBottom = (force = false) => {
    if (!force && userHasScrolledUp.value) return;
    nextTick(() => {
        if (scrollContainer.value) {
            scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight;
        }
    })
}
const onClickScrollToBottom = () => {
    userHasScrolledUp.value = false;
    scrollToBottom(true);
}

// Images and other rich Markdown content can grow after the SSE chunk that
// introduced them. Follow those delayed height changes while the user remains
// at the live edge; preserve position when they intentionally scroll upward.
useStickyBottomOnResize(scrollContainer, userHasScrolledUp, scrollToBottom);

const debounce = (fn, delay) => {
    let timer
    return (...args) => {
        clearTimeout(timer)
        timer = setTimeout(() => fn(...args), delay)
    }
}
const onChatScrollTop = () => {
    if (scrollLock.value || historyLoadingMore.value || !hasMoreHistory.value) return;
    if (!scrollContainer.value) return;
    const { scrollTop, scrollHeight } = scrollContainer.value;
    isFirstEnter.value = false
    if (scrollTop <= 0) {
        let data = {
            session_id: session_id.value,
            created_at: created_at.value,
            limit: limit.value
        }
        getmsgList(data, true, scrollHeight);
    }
}
const debouncedScrollTop = debounce(onChatScrollTop, 500);
let lastScrollTop = 0;
const handleScroll = () => {
    const el = scrollContainer.value;
    if (el) {
        const currentTop = el.scrollTop;
        // Only an actual upward scroll detaches from the live edge. Content that
        // grows after a chunk (images, diagrams) keeps scrollTop fixed and would
        // otherwise fire a stale scroll event that falsely marks the user as
        // scrolled up, killing the auto-follow during streaming.
        if (currentTop < lastScrollTop - 1) {
            userHasScrolledUp.value = !isNearBottom();
        } else if (isNearBottom()) {
            userHasScrolledUp.value = false;
        }
        lastScrollTop = currentTop;
    }
    debouncedScrollTop();
};

const fetchMessageList = (data) => getMessageList(data);

const {
    findLastMessage,
    shouldRenderAssistantMessage,
    shouldShowGlobalTypingIndicator,
    handleMsgList,
    processStreamChunk,
    prepareForNewOutgoingMessage,
    markInFlightAssistantStopped,
} = useChatStreamHandler({
    messagesList,
    loading,
    isReplying,
    currentAssistantMessageId,
    fullContent,
    isAgentStreamSession,
    scrollToBottom,
    onError: (msg) => MessagePlugin.error(msg),
    preserveIncompleteStreamReactive: true,
    isFirstEnter,
    scrollContainer,
    debug: import.meta.env.DEV,
    onAfterMsgList: async () => {
        for (const message of messagesList) {
            if (message.role === 'assistant' && message.is_completed && message.suggestionSet === undefined) {
                void loadFollowUpSuggestions(message, false);
            }
        }
        const lastMessage = messagesList[messagesList.length - 1];
        if (lastMessage && !lastMessage.is_completed) {
            isReplying.value = true;
            if (lastMessage.role === 'assistant') {
                currentAssistantMessageId.value = lastMessage.id;
                console.log('[Continue Stream] Set assistant message ID:', lastMessage.id);
            }
            // Only IM-originated replies (channel === 'im') get the quiet poll-to-recover
            // path: their answer is generated on the IM side and never streams through
            // this server, so continue-stream always 404s even though the reply *is*
            // coming. Web/api replies keep the original behaviour (a real failure to
            // resume the stream still surfaces as an error) — we don't touch them.
            isAttachingImStream.value = lastMessage.channel === 'im';
            await startStream({
                session_id: session_id.value,
                query: lastMessage.id,
                method: 'GET',
                url: '/api/v1/sessions/continue-stream',
            });
            // On success the stream resumed normally; on failure the error watcher
            // already took over (quiet recovery for IM), so only clear the flag here.
            if (!error.value) isAttachingImStream.value = false;
        }
    },
    onAgentQuery: (data, existingMessage) => {
        pendingStreamDebug.value = buildStreamDebugPayload();
        if (existingMessage) attachStreamDebugToMessage(existingMessage);
    },
    onMessageCreated: (message) => attachStreamDebugToMessage(message),
    onMessageUpdated: (message, payload) => {
        attachStreamDebugToMessage(message);
        if (payload?.is_completed) pendingStreamDebug.value = null;
    },
    onAgentAnswerDone: (message) => {
        attachStreamDebugToMessage(message);
        pendingStreamDebug.value = null;
    },
    onAgentChunkBound: (message) => {
        attachStreamDebugToMessage(message);
        pendingStreamDebug.value = null;
    },
    onTurnComplete: (message) => {
        void loadFollowUpSuggestions(message, true);
    },
});

const showGlobalTypingIndicator = computed(() =>
    shouldShowGlobalTypingIndicator(messagesList, loading.value, isImRecovering.value),
);

const getmsgList = (data, isScrollType = false, scrollHeight) => {
    if (isPublishedExpertSessionId(data?.session_id)) {
        historyLoading.value = false;
        historyLoadingMore.value = false;
        hasMoreHistory.value = false;
        emitMessageStateChange();
        return Promise.resolve();
    }
    if (isScrollType) {
        if (historyLoadingMore.value || !hasMoreHistory.value) return;
        historyLoadingMore.value = true;
    }
    return fetchMessageList(data).then(async (res) => {
        const batch = res?.data;
        if (!batch?.length) {
            if (isScrollType) {
                hasMoreHistory.value = false;
            }
            return;
        }
        if (!isScrollType) {
            cancelSuggestedQuestionsFetch();
        }
        const nextCursor = batch[0].created_at;
        if (isScrollType && created_at.value && nextCursor === created_at.value) {
            hasMoreHistory.value = false;
            return;
        }
        if (batch.length < limit.value) {
            hasMoreHistory.value = false;
        }
        created_at.value = nextCursor;
        await handleMsgList(batch, isScrollType, scrollHeight);
        emitMessageStateChange();
    }).catch((err) => {
        console.error('Failed to load messages:', err);
        if (isScrollType) {
            hasMoreHistory.value = false;
        }
    }).finally(() => {
        historyLoading.value = false;
        historyLoadingMore.value = false;
        if (!isScrollType && messagesList.length === 0) {
            fetchSuggestedQuestionsIfNeeded();
        }
        if (!isScrollType) {
            emitMessageStateChange();
        }
    })
}

// 发送消息
// 处理停止生成事件 - 立即清除 loading 状态
const handleStopGeneration = () => {
    console.log('[Stop Generation] Immediately clearing loading state');
    stopStream();
    loading.value = false;
    isReplying.value = false;
    // 标记当前 assistant 为已结束，避免下一条 query 复用该消息行
    markInFlightAssistantStopped(currentAssistantMessageId.value);
    // 保留 currentAssistantMessageId，Input-field 仍需用它调用 stop API
};

const openExpertArtifact = (artifact, runId = '') => {
    activeExpertArtifact.value = artifact;
    activeExpertArtifactRunId.value = runId;
};

const publishedExpertStorageKey = () => {
    const currentSessionId = String(session_id.value || '').trim();
    return currentSessionId ? `ruile:published-expert-runs:${currentSessionId}` : '';
};

const readStoredPublishedExpertRuns = () => {
    const key = publishedExpertStorageKey();
    if (!key) return [];
    try {
        const raw = localStorage.getItem(key);
        const parsed = raw ? JSON.parse(raw) : [];
        return Array.isArray(parsed) ? parsed : [];
    } catch (error) {
        console.warn('[AgentRun] failed to read persisted expert runs:', error);
        return [];
    }
};

const storedPublishedExpertEvent = (event) => {
    if (!event || typeof event !== 'object') return null;
    const safe = {
        type: event.type,
        id: event.id,
        runId: event.runId,
        sequence: event.sequence,
        status: event.status,
        phase: event.phase,
        message: event.message,
        delta: event.delta,
        stepId: event.stepId,
        stepType: event.stepType,
        label: event.label,
        createdAt: event.createdAt,
        startedAt: event.startedAt,
        finishedAt: event.finishedAt,
        durationMs: event.durationMs,
        totalDurationMs: event.totalDurationMs,
        queueDurationMs: event.queueDurationMs,
        title: event.title,
        messageId: event.messageId,
        parentMessageId: event.parentMessageId,
        toolCallId: event.toolCallId,
        toolCallName: event.toolCallName,
        activityType: event.activityType,
        content: event.content,
        output: event.output,
        result: event.result,
        error: event.error,
        errorCode: event.errorCode,
        answers: event.answers,
        interaction: event.interaction,
        quality: event.quality,
        modelId: event.modelId,
        attempt: event.attempt,
        queuedAt: event.queuedAt,
        role: event.role,
        replace: event.replace,
        retrying: event.retrying,
    };
    return Object.fromEntries(Object.entries(safe).filter(([, value]) => value !== undefined));
};

const persistPublishedExpertRun = (message, prompt = '') => {
    const key = publishedExpertStorageKey();
    const runId = message?.run?.id;
    if (!key || !runId) return;
    const stored = readStoredPublishedExpertRuns().filter((item) => item?.runId !== runId);
    const run = message.run || {};
    const runSnapshot = {
        id: run.id,
        status: run.status,
        phase: run.phase,
        interaction: run.interaction,
        error_code: run.error_code,
        error_message: run.error_message,
        last_event_sequence: message.last_event_sequence || run.last_event_sequence || 0,
    };
    stored.push({
        runId,
        prompt: prompt || message.prompt || '',
        message_kind: message.isPublishedExpertFollowUp ? 'follow_up' : 'run',
        expert: message.expert,
        run: runSnapshot,
        events: (message.events || []).slice(-200).map(storedPublishedExpertEvent).filter(Boolean),
        submittedAnswers: message.submittedAnswers || null,
        submittedAnswerSummary: message.submittedAnswerSummary || '',
        updatedAt: Date.now(),
    });
    try {
        localStorage.setItem(key, JSON.stringify(stored.slice(-20)));
    } catch (error) {
        console.warn('[AgentRun] failed to persist expert run:', error);
    }
};

const restorePublishedExpertRuns = async () => {
    const storedRuns = readStoredPublishedExpertRuns();
    if (!storedRuns.length) return;
    for (const stored of storedRuns) {
        if (!stored?.runId || messagesList.some((message) => message?.run?.id === stored.runId)) {
            continue;
        }
        const userMessage = {
            id: `${stored.runId}-restored-user`,
            content: stored.prompt || '继续查看专家运行结果',
            role: 'user',
            channel: 'web',
            expert_name: stored.expert?.display_name,
            isRestoredPublishedExpertMessage: true,
        };
        const assistantMessage = reactive({
            id: `${stored.runId}-restored-assistant`,
            role: 'assistant',
            isPublishedExpertRun: stored.message_kind !== 'follow_up',
            isPublishedExpertFollowUp: stored.message_kind === 'follow_up',
            is_completed: false,
            expert: stored.expert || {},
            prompt: stored.prompt || '',
            events: Array.isArray(stored.events) ? stored.events : [],
            steps: [],
            submittedAnswers: stored.submittedAnswers || null,
            submittedAnswerSummary: stored.submittedAnswerSummary || '',
            last_event_sequence: 0,
            run: stored.run || { id: stored.runId, status: 'queued' },
        });
        messagesList.push(userMessage, assistantMessage);
        try {
            const response = await getAgentRun(stored.runId);
            if (!response?.success || !response?.data) {
                throw new Error(response?.message || '专家运行状态读取失败');
            }
            assistantMessage.run = response.data;
            await waitForPublishedExpertRun(assistantMessage, stored.runId);
            assistantMessage.is_completed = true;
        } catch (error) {
            assistantMessage.run = {
                ...(assistantMessage.run || {}),
                status: 'failed',
                error_message: error?.message || '专家运行恢复失败',
            };
            assistantMessage.is_completed = true;
        }
        persistPublishedExpertRun(assistantMessage, stored.prompt || '');
    }
    emitMessageStateChange();
    await nextTick();
    scrollToBottom(true);
};

const summarizeExpertAnswers = (interaction, answers) => {
    const questions = Array.isArray(interaction?.questions) ? interaction.questions : [];
    return questions
        .map((question) => {
            const value = answers?.[question.id];
            if (value === undefined || value === null || value === '') return '';
            const text = Array.isArray(value) ? value.join('、') : String(value);
            return `${question.label || question.id}：${text}`;
        })
        .filter(Boolean)
        .join('；');
};

const submitPublishedExpertAnswers = async (message, answers) => {
    const runId = message?.run?.id;
    if (!runId || message.submittingAnswers) return;

    const answerSummary = summarizeExpertAnswers(message.run?.interaction, answers);
    const messageIndex = messagesList.indexOf(message);
    const answerMessage = {
        id: `published-expert-answer-${Date.now()}`,
        content: answerSummary,
        role: 'user',
        channel: 'web',
        isPublishedExpertAnswer: true,
    };
    const resumeMessage = reactive({
        id: `published-expert-resume-${Date.now()}`,
        role: 'assistant',
        isPublishedExpertRun: true,
        is_completed: false,
        expert: message.expert,
        run: {
            status: 'queued',
        },
    });

    message.submittedAnswers = answers;
    message.submittedAnswerSummary = answerSummary;
    persistPublishedExpertRun(message, message.prompt || '');
    messagesList.splice(messageIndex + 1, 0, answerMessage, resumeMessage);
    emitMessageStateChange();
    await nextTick();
    scrollToBottom(true);
    message.submittingAnswers = true;
    try {
        const response = await submitAgentRunAnswers(runId, answers);
        if (!response?.success || !response?.data) {
            throw new Error(response?.message || '补充信息提交失败');
        }
        resumeMessage.run = response.data;
        resumeMessage.prompt = message.prompt || '';
        resumeMessage.events = message.events || [];
        resumeMessage.steps = message.steps || [];
        resumeMessage.last_event_sequence = message.last_event_sequence || 0;
        await waitForPublishedExpertRun(resumeMessage, runId);
        resumeMessage.is_completed = true;
        persistPublishedExpertRun(resumeMessage, message.prompt || '');
    } catch (error) {
        const currentIndex = messagesList.indexOf(answerMessage);
        if (currentIndex >= 0) messagesList.splice(currentIndex, 2);
        message.submittedAnswers = null;
        message.submittedAnswerSummary = '';
        persistPublishedExpertRun(message, message.prompt || '');
        MessagePlugin.error(error?.message || '补充信息提交失败');
    } finally {
        message.submittingAnswers = false;
        await nextTick();
        scrollToBottom(true);
    }
};

const regeneratePublishedExpertRun = async (message, action) => {
    const parentRunId = String(message?.run?.id || '').trim();
    const feedback = String(action?.feedback || '').trim();
    const label = String(action?.label || '继续完善').trim();
    const actionKey = String(action?.key || 'regenerate').trim();
    if (!parentRunId || !feedback || message.regeneratingAction) return;

    message.regeneratingAction = actionKey;
    const actionMessage = {
        id: `published-expert-follow-up-${Date.now()}`,
        content: label,
        role: 'user',
        channel: 'web',
        expert_name: message.expert?.display_name,
        isPublishedExpertFollowUp: true,
    };
    const childMessage = reactive({
        id: `published-expert-child-${Date.now()}`,
        role: 'assistant',
        isPublishedExpertRun: true,
        is_completed: false,
        expert: message.expert || {},
        prompt: label,
        feedback,
        parentRunId,
        events: [],
        last_event_sequence: 0,
        run: {
            status: 'queued',
            parent_run_id: parentRunId,
        },
    });

    messagesList.push(actionMessage, childMessage);
    emitMessageStateChange();
    await nextTick();
    scrollToBottom(true);

    try {
        const response = await regenerateAgentRun(parentRunId, feedback);
        if (!response?.success || !response?.data?.id) {
            throw new Error(response?.message || '新版本任务创建失败');
        }
        childMessage.run = response.data;
        persistPublishedExpertRun(childMessage, label);
        await waitForPublishedExpertRun(childMessage, response.data.id);
        childMessage.is_completed = true;
        persistPublishedExpertRun(childMessage, label);
    } catch (error) {
        childMessage.run = {
            ...(childMessage.run || {}),
            status: 'failed',
            error_message: error?.message || '新版本生成失败',
        };
        childMessage.is_completed = true;
        persistPublishedExpertRun(childMessage, label);
        MessagePlugin.error(childMessage.run.error_message);
    } finally {
        message.regeneratingAction = '';
        emitMessageStateChange();
        await nextTick();
        scrollToBottom(true);
    }
};

const askPublishedExpertFollowUp = async (message, action) => {
    const parentRunId = String(message?.run?.id || '').trim();
    const prompt = String(action?.prompt || '').trim();
    const label = String(action?.label || '解释上一版结果').trim();
    const displayText = String(action?.displayText || label).trim();
    const actionKey = String(action?.key || 'explanation').trim();
    if (!parentRunId || !prompt || message.followingUpAction) return;

    message.followingUpAction = actionKey;
    const actionMessage = {
        id: `published-expert-explanation-${Date.now()}`,
        content: displayText,
        role: 'user',
        channel: 'web',
        expert_name: message.expert?.display_name,
        isPublishedExpertFollowUp: true,
    };
    const answerMessage = reactive({
        id: `published-expert-explanation-answer-${Date.now()}`,
        role: 'assistant',
        isPublishedExpertFollowUp: true,
        is_completed: false,
        expert: message.expert || {},
        prompt: label,
        parentRunId,
        events: [],
        last_event_sequence: 0,
        run: {
            status: 'queued',
            parent_run_id: parentRunId,
        },
    });

    messagesList.push(actionMessage, answerMessage);
    emitMessageStateChange();
    await nextTick();
    scrollToBottom(true);

    try {
        const response = await runPublishedExpertFollowUp(parentRunId, {
            prompt,
            mode: 'explanation',
        });
        if (!response?.success || !response?.data?.id) {
            throw new Error(response?.message || '专家追问创建失败');
        }
        answerMessage.run = response.data;
        persistPublishedExpertRun(answerMessage, label);
        await waitForPublishedExpertRun(answerMessage, response.data.id);
        answerMessage.is_completed = true;
        persistPublishedExpertRun(answerMessage, label);
    } catch (error) {
        answerMessage.run = {
            ...(answerMessage.run || {}),
            status: 'failed',
            error_message: error?.message || '专家追问失败',
        };
        answerMessage.is_completed = true;
        persistPublishedExpertRun(answerMessage, label);
        MessagePlugin.error(answerMessage.run.error_message);
    } finally {
        message.followingUpAction = '';
        emitMessageStateChange();
        await nextTick();
        scrollToBottom(true);
    }
};

const cancelPublishedExpertRun = async (message) => {
    const runId = String(message?.run?.id || '').trim();
    if (!runId || message.cancelling) return;
    message.cancelling = true;
    try {
        const response = await cancelAgentRun(runId);
        if (!response?.success || !response?.data) {
            throw new Error(response?.message || '取消任务失败');
        }
        message.run = response.data;
        message.is_completed = true;
        persistPublishedExpertRun(message, message.prompt || '');
    } catch (error) {
        MessagePlugin.error(error?.message || '取消任务失败');
    } finally {
        message.cancelling = false;
        emitMessageStateChange();
    }
};

const openExpertRunDiff = async (message) => {
    const runId = String(message?.run?.id || '').trim();
    if (!runId) return;
    activeExpertRunDiff.value = { before: '', after: '', changed: false };
    expertRunDiffLoading.value = true;
    try {
        const response = await getAgentRunDiff(runId);
        if (!response?.success || !response?.data) {
            throw new Error(response?.message || '版本差异读取失败');
        }
        activeExpertRunDiff.value = response.data;
    } catch (error) {
        activeExpertRunDiff.value = null;
        MessagePlugin.error(error?.message || '版本差异读取失败');
    } finally {
        expertRunDiffLoading.value = false;
    }
};

const publishedExpertStepRequests = new Map();
const hydratePublishedExpertSteps = async (message, runId) => {
    const id = String(runId || '').trim();
    if (!id || !message) return;
    if (publishedExpertStepRequests.has(id)) {
        await publishedExpertStepRequests.get(id);
        return;
    }
    const request = listAgentRunSteps(id)
        .then((response) => {
            if (response?.success && Array.isArray(response.data)) {
                message.steps = response.data;
            }
        })
        .catch((error) => {
            console.warn('[AgentRun] failed to load step input/output:', error);
        })
        .finally(() => {
            publishedExpertStepRequests.delete(id);
        });
    publishedExpertStepRequests.set(id, request);
    await request;
};

const waitForPublishedExpertRun = async (message, runId) => {
    const applyEvent = (event) => {
        if (!event || event.runId !== runId) return;
        const sequence = Number(event.sequence || 0);
        const events = Array.isArray(message.events) ? message.events : [];
        const existingIndex = sequence > 0
            ? events.findIndex((item) => Number(item?.sequence || 0) === sequence)
            : events.findIndex((item) => item?.id && item.id === event.id);
        if (existingIndex >= 0) {
            const nextEvents = [...events];
            nextEvents[existingIndex] = { ...nextEvents[existingIndex], ...event };
            message.events = nextEvents;
        } else {
            message.events = [...events, event]
                .sort((left, right) => Number(left?.sequence || 0) - Number(right?.sequence || 0))
                .slice(-200);
        }
        message.run = {
            ...(message.run || {}),
            ...(event.status ? { status: event.status } : {}),
            ...(event.phase ? { phase: event.phase } : {}),
            ...(event.interaction ? { interaction: event.interaction } : {}),
            ...(event.type === 'RUN_STARTED' && event.startedAt ? { started_at: event.startedAt } : {}),
            ...(['RUN_FINISHED', 'RUN_ERROR', 'RUN_CANCELLED'].includes(event.type) && event.createdAt
                ? { finished_at: event.createdAt }
                : {}),
            last_event_sequence: Math.max(
                Number(message.run?.last_event_sequence || 0),
                Number(message.last_event_sequence || 0),
                sequence,
            ),
        };
        message.last_event_sequence = message.run.last_event_sequence;
        if (event.type === 'TEXT_MESSAGE_CONTENT' && event.delta) {
            message.liveSummary = `${message.liveSummary || ''}${event.delta}`;
        }
        if (event.type === 'AGENT_STEP_FINISHED' || event.type === 'AGENT_STEP_ERROR') {
            void hydratePublishedExpertSteps(message, runId);
        }
        persistPublishedExpertRun(message, message.prompt || '');
    };

    try {
        await streamAgentRunEvents(runId, {
            afterSequence: Number(message.run?.last_event_sequence || 0),
            onEvent: applyEvent,
        });
    } catch (error) {
        // Keep the old polling path as a compatibility fallback while older
        // deployments are still starting up without the event migration.
        console.warn('[AgentRun] event stream unavailable, falling back to polling:', error);
    }

    const deadline = Date.now() + 5 * 60 * 1000;
    while (Date.now() < deadline) {
        const response = await getAgentRun(runId);
        const run = response?.data;
        if (!response?.success || !run) {
            throw new Error(response?.message || '专家运行状态读取失败');
        }
        message.run = run;
        message.last_event_sequence = Math.max(
            Number(message.last_event_sequence || 0),
            Number(message.run?.last_event_sequence || 0),
        );
        persistPublishedExpertRun(message, message.prompt || '');
        if (run.status === 'succeeded' || run.status === 'waiting_input'
            || run.status === 'failed' || run.status === 'cancelled') {
            await hydratePublishedExpertSteps(message, runId);
            return run;
        }
        await new Promise((resolve) => window.setTimeout(resolve, 1200));
    }
    throw new Error('专家运行超时，请稍后重试');
};

const routedExpertFromRun = (run, fallbackExpert) => {
    const decision = run?.input?.routing_decision;
    const candidate = Array.isArray(decision?.candidates) ? decision.candidates[0] : null;
    if (!candidate) return fallbackExpert;
    return {
        package_id: candidate.package_id,
        package_version_id: candidate.package_version_id,
        package_display_name: candidate.package_display_name,
        definition_id: candidate.definition_id,
        agent_id: candidate.agent_id,
        version: candidate.version,
        display_name: candidate.display_name,
        description: candidate.description,
        domain: candidate.domain,
        route_mode: 'auto',
    };
};

const isAutoRouteFallbackError = (error) => {
    const message = String(error?.message || '').toLowerCase();
    return error?.status === 400
        && message.includes('no published expert matched');
};

const appendPublishedExpertRun = async (value, routeMode, displayExpert, runData, appendUserMessage = true) => {
    const localMessageId = `published-expert-${Date.now()}`;
    if (appendUserMessage) {
        messagesList.push({
            id: `${localMessageId}-user`,
            content: value,
            role: 'user',
            channel: 'web',
            expert_name: displayExpert.display_name,
        });
    }
    const assistantMessage = reactive({
        id: `${localMessageId}-assistant`,
        role: 'assistant',
        isPublishedExpertRun: true,
        is_completed: false,
        expert: routedExpertFromRun(runData, displayExpert),
        routeMode,
        prompt: value,
        events: [],
        last_event_sequence: 0,
        run: runData,
    });
    messagesList.push(assistantMessage);
    emitMessageStateChange();
    scrollToBottom(true);

    try {
        persistPublishedExpertRun(assistantMessage, value);
        const run = await waitForPublishedExpertRun(assistantMessage, runData.id);
        assistantMessage.run = run;
        assistantMessage.expert = routedExpertFromRun(run, assistantMessage.expert);
        assistantMessage.is_completed = true;
        persistPublishedExpertRun(assistantMessage, value);
    } catch (error) {
        assistantMessage.run = {
            ...(assistantMessage.run || {}),
            status: 'failed',
            error_message: error?.message || '专家运行失败',
        };
        assistantMessage.is_completed = true;
        persistPublishedExpertRun(assistantMessage, value);
        MessagePlugin.error(assistantMessage.run.error_message);
    } finally {
        isReplying.value = false;
        loading.value = false;
        currentAssistantMessageId.value = '';
        await nextTick();
        scrollToBottom(true);
    }
};

const appendPublishedExpertRouteChoice = (value, modelId, decision) => {
    const localMessageId = `published-expert-route-${Date.now()}`;
    const selectedCandidate = Array.isArray(decision?.candidates)
        ? decision.candidates.find((candidate) => candidate.definition_id === decision.selected_expert_id)
        : null;
    messagesList.push({
        id: `${localMessageId}-user`,
        content: value,
        role: 'user',
        channel: 'web',
        expert_name: '自动匹配专家',
    });
    messagesList.push(reactive({
        id: `${localMessageId}-assistant`,
        role: 'assistant',
        isPublishedExpertRouteChoice: true,
        is_completed: true,
        prompt: value,
        modelId: modelId || '',
        routeDecision: decision,
        selectedCandidate,
        confirming: false,
        confirmed: false,
        cancelled: false,
    }));
    emitMessageStateChange();
    isReplying.value = false;
    loading.value = false;
    currentAssistantMessageId.value = '';
    nextTick(() => scrollToBottom(true));
};

const confirmPublishedExpertRoute = async (message, candidate) => {
    if (!message || !candidate || message.confirming || message.confirmed || message.cancelled) return;
    message.confirming = true;
    isReplying.value = true;
    loading.value = true;
    const originalDecision = message.routeDecision || {};
    const candidates = Array.isArray(originalDecision.candidates)
        ? [candidate, ...originalDecision.candidates.filter((item) => item.definition_id !== candidate.definition_id)]
        : [candidate];
    const routingDecision = {
        ...originalDecision,
        route_mode: 'auto',
        selected_expert_id: candidate.definition_id,
        selected_version: candidate.version,
        candidates: candidates.slice(0, 3),
        requires_confirmation: false,
    };
    try {
        const response = await runPublishedExpertAuto({
            prompt: message.prompt,
            model_id: message.modelId || undefined,
            routing_decision: routingDecision,
            user_confirmed: true,
        });
        if (!response?.success || !response?.data?.id) {
            throw new Error(response?.message || '专家运行创建失败');
        }
        message.confirming = false;
        message.confirmed = true;
        message.selectedCandidate = candidate;
        await appendPublishedExpertRun(
            message.prompt,
            'auto',
            { ...candidate, route_mode: 'auto' },
            response.data,
            false,
        );
    } catch (error) {
        message.confirming = false;
        isReplying.value = false;
        loading.value = false;
        MessagePlugin.error(error?.message || '专家路由确认失败');
    }
};

const cancelPublishedExpertRoute = (message) => {
    if (!message || message.confirming || message.confirmed) return;
    message.cancelled = true;
    message.is_completed = true;
    isReplying.value = false;
    loading.value = false;
    currentAssistantMessageId.value = '';
    emitMessageStateChange();
};

const handlePublishedExpertSend = async (value, modelId, expert, routeMode = 'manual') => {
    stopStream();
    prepareForNewOutgoingMessage();
    isReplying.value = true;
    loading.value = true;

    const displayExpert = expert || { display_name: '自动匹配专家', route_mode: 'auto' };
    try {
        if (routeMode === 'auto') {
            const routeResponse = await routePublishedExpert(value, modelId || undefined);
            if (!routeResponse?.success || !routeResponse?.data) {
                throw new Error(routeResponse?.message || '专家路由失败');
            }
            const decision = routeResponse.data;
            if (decision.requires_confirmation || Number(decision.confidence || 0) < 0.85) {
                appendPublishedExpertRouteChoice(value, modelId, decision);
                return;
            }
            const response = await runPublishedExpertAuto({
                prompt: value,
                model_id: modelId || undefined,
                routing_decision: decision,
                user_confirmed: true,
            });
            if (!response?.success || !response?.data?.id) {
                throw new Error(response?.message || '专家运行创建失败');
            }
            await appendPublishedExpertRun(value, routeMode, displayExpert, response.data);
            return;
        }
        const response = await runPublishedExpert(expert.package_id, expert.definition_id, {
                prompt: value,
                model_id: modelId || undefined,
                route_mode: 'manual',
            });
        if (!response?.success || !response?.data?.id) {
            throw new Error(response?.message || '专家运行创建失败');
        }
        await appendPublishedExpertRun(value, routeMode, displayExpert, response.data);
    } catch (error) {
        isReplying.value = false;
        loading.value = false;
        if (routeMode === 'auto' && isAutoRouteFallbackError(error)) {
            // No relevant published expert: transparently continue with the
            // existing model chat instead of surfacing a routing error.
            sendMsg(value, modelId);
            return;
        }
        MessagePlugin.error(error?.message || '专家运行创建失败');
        isReplying.value = false;
        loading.value = false;
        currentAssistantMessageId.value = '';
    }
};

const sendMsg = async (value, modelId = '', mentionedItems = [], imageFiles = [], attachmentFiles = [], responseTier = useSettingsStoreInstance.settings.responseTier) => {
    stopStream();
    prepareForNewOutgoingMessage();
    isReplying.value = true;
    loading.value = true;
    const selectedAgentId = props.embeddedMode ? props.agentId : (useSettingsStoreInstance.selectedAgentId || '');

    // Images are unified with the attachment pipeline: on the authenticated web
    // client they upload as temporary documents (understood in the background by
    // the VLM) and are sent as attachment_ids. The inline base64 `images`
    // payload is kept only for the embedded/public API path. A base64 fallback
    // is used per-image if the async upload fails.
    let imageAttachments = [];
    let userImages = [];
    const imageAttachmentIds = [];
    if (imageFiles && imageFiles.length > 0) {
        for (const file of imageFiles) {
            let dataURI;
            try {
                dataURI = await fileToBase64(file);
            } catch (e) {
                console.error('[Image] Failed to read images:', e);
                loading.value = false;
                isReplying.value = false;
                return;
            }
            userImages.push({ url: dataURI });
            if (props.embeddedMode) {
                imageAttachments.push({ data: dataURI });
                continue;
            }
            try {
                const upload = await uploadTemporaryAttachment(session_id.value, file, selectedAgentId, 'auto');
                imageAttachmentIds.push(upload.data.id);
            } catch (e) {
                console.error('[Image] Temporary image upload failed, falling back to inline:', e);
                imageAttachments.push({ data: dataURI });
            }
        }
    }

    // The create-chat page cannot upload before its session exists. Once it
    // navigates here, move those local files through the same asynchronous
    // upload/parse flow before starting the first stream.
    const localAttachments = (attachmentFiles || []).filter(attachment => !attachment.documentId);
    if (!props.embeddedMode && localAttachments.length > 0) {
        try {
            // Only upload to obtain a document ID; parsing continues in the
            // background and is awaited by the backend (shown on the timeline).
            await Promise.all(localAttachments.map(async (attachment) => {
                attachment.status = 'uploading';
                const upload = await uploadTemporaryAttachment(
                    session_id.value, attachment.file, selectedAgentId, 'auto'
                );
                attachment.documentId = upload.data.id;
                attachment.status = upload.data.status;
            }));
        } catch (error) {
            console.error('[Attachment] Temporary document upload failed:', error);
            await Promise.all(localAttachments
                .filter(attachment => attachment.documentId)
                .map(attachment => deleteTemporaryAttachment(session_id.value, attachment.documentId).catch(() => undefined)));
            MessagePlugin.error(error?.message || t('chat.attachmentParseFailed'));
            loading.value = false;
            isReplying.value = false;
            return;
        }
    }

    // Send any successfully uploaded attachment (parsing may still be running);
    // the backend waits for readiness and reports progress on the timeline.
    const attachmentIds = (attachmentFiles || [])
        .filter(attachment => attachment.documentId && attachment.status !== 'failed')
        .map(attachment => attachment.documentId);
    attachmentIds.push(...imageAttachmentIds);
	// Embedded public routes do not expose the authenticated session upload API;
	// keep their existing inline payload for compatibility.
    const legacyAttachmentFiles = props.embeddedMode
        ? (attachmentFiles || []).filter(attachment => !attachment.documentId)
        : [];
    let attachmentUploads = [];
    if (legacyAttachmentFiles.length > 0) {
        try {
            for (const attachment of legacyAttachmentFiles) {
                const reader = new FileReader();
                const base64Promise = new Promise((resolve, reject) => {
                    reader.onload = () => {
                        const result = reader.result;
                        // Extract base64 content (remove data:...;base64, prefix)
                        const base64 = result.split(',')[1];
                        resolve(base64);
                    };
                    reader.onerror = reject;
                    reader.readAsDataURL(attachment.file);
                });
                const base64Data = await base64Promise;
                attachmentUploads.push({
                    data: base64Data,
                    file_name: attachment.name,
                    file_size: attachment.size
                });
            }
        } catch (e) {
            console.error('[Attachment] Failed to read attachments:', e);
            loading.value = false;
            isReplying.value = false;
            return;
        }
    }

    // 将@提及的知识库和文件信息存入用户消息
    messagesList.push({ content: value, role: 'user', mentioned_items: mentionedItems, images: userImages, attachments: attachmentFiles.map(a => ({ id: a.documentId, file_name: a.name, file_size: a.size, file_type: '.' + a.name.split('.').pop()?.toLowerCase() })), channel: 'web' });
    emit('user-message-send', {
        sessionId: session_id.value,
        query: value,
    });
    emitMessageStateChange();
    userHasScrolledUp.value = false;
    scrollToBottom(true);

    // Get agent mode status from settings store (prefer selectedAgentId for builtins)
    const agentEnabled = props.embeddedMode
        ? (props.agentId && props.agentId !== 'builtin-quick-answer')
        : useSettingsStoreInstance.isAgentStreamMode;

    // Get web search status from settings store
    const webSearchEnabled = props.embeddedMode ? false : useSettingsStoreInstance.isWebSearchEnabled;

    // Get knowledge_base_ids from settings store (selected by user via KnowledgeBaseSelector)
    // Merge @mentioned KB/file IDs so retrieval uses the same targets user @mentioned (including shared KBs)
    const sidebarKbIds = props.embeddedMode ? props.kbIds : (useSettingsStoreInstance.settings.selectedKnowledgeBases || []);
    const sidebarFileIds = props.embeddedMode ? [] : (useSettingsStoreInstance.settings.selectedFiles || []);
    const kbIdSet = new Set(sidebarKbIds);
    const fileIdSet = new Set(sidebarFileIds);
    for (const kbId of pendingSuggestionKnowledgeBaseIds) {
        if (kbId) kbIdSet.add(kbId);
    }
    for (const item of mentionedItems || []) {
        if (!item?.id) continue;
        if (item.type === 'kb' && !kbIdSet.has(item.id)) {
            kbIdSet.add(item.id);
        } else if (item.type === 'file' && !fileIdSet.has(item.id)) {
            fileIdSet.add(item.id);
        }
    }
    const kbIds = [...kbIdSet];
    const knowledgeIds = [...fileIdSet];
    const tagIds = [...new Set((mentionedItems || []).filter(item => item.type === 'tag' && item.id).map(item => item.id))];
    const mcpServiceIds = [...new Set((mentionedItems || []).filter(item => item.type === 'mcp' && item.id).map(item => item.id))];
    const skillNames = [...new Set((mentionedItems || []).filter(item => item.type === 'skill' && item.id).map(item => item.skill_name || item.id))];

    const endpoint = agentEnabled ? '/api/v1/agent-chat' : '/api/v1/knowledge-chat';

    const requestMcpServiceIds = agentEnabled ? mcpServiceIds : [];
    const requestSkillNames = agentEnabled ? skillNames : [];

    const suggestionAttribution = pendingSuggestionAttribution;
    pendingSuggestionAttribution = null;
    pendingSuggestionKnowledgeBaseIds = [];
    const modelOverride = props.embeddedMode ? '' : modelId;
    await startStream({
        session_id: session_id.value,
        knowledge_base_ids: kbIds,
        knowledge_ids: knowledgeIds,
        agent_enabled: agentEnabled,
        agent_id: selectedAgentId,
        web_search_enabled: webSearchEnabled,
        response_tier: props.embeddedMode ? undefined : responseTier,
        summary_model_id: modelOverride,
        mcp_service_ids: requestMcpServiceIds,
        skill_names: requestSkillNames,
        tag_ids: tagIds,
        mentioned_items: mentionedItems,
        images: imageAttachments.length > 0 ? imageAttachments : undefined,
        attachment_uploads: attachmentUploads.length > 0 ? attachmentUploads : undefined,
        attachment_ids: attachmentIds.length > 0 ? attachmentIds : undefined,
        query: value,
        suggestion_attribution: suggestionAttribution || undefined,
        quoted_context: props.quotedContext || undefined,
        method: 'POST',
        url: endpoint,
    });
}

// Quietly recover an in-flight IM reply we couldn't attach to (it's generated on
// the IM side, so it never streamed through this server). Poll until it completes,
// then reload the thread so it renders via the normal path. Bounded so an IM reply
// that genuinely died (e.g. the bot crashed) doesn't spin forever — on timeout we
// surface the original error so the failure isn't hidden.
const RECOVER_POLL_INTERVAL = 2500;
const RECOVER_POLL_MAX_ATTEMPTS = 48; // ~2 min
const recoverIncompleteMessage = () => {
    const targetSession = session_id.value;
    const targetMessageId = currentAssistantMessageId.value;
    if (recoverPollTimer) { clearTimeout(recoverPollTimer); recoverPollTimer = null; }
    if (!targetMessageId) { isReplying.value = false; isImRecovering.value = false; return; }
    isImRecovering.value = true; // show the "generating" indicator while we poll
    let attempts = 0;
    const poll = async () => {
        recoverPollTimer = null;
        if (session_id.value !== targetSession) { isReplying.value = false; isImRecovering.value = false; return; } // navigated away
        attempts++;
        try {
            const res = await getMessageList({ session_id: targetSession, limit: limit.value, created_at: '' });
            const target = (res?.data || []).find((m) => m.id === targetMessageId);
            if (target && target.is_completed) {
                created_at.value = '';
                messagesList.splice(0);
                getmsgList({ session_id: targetSession, limit: limit.value, created_at: '' });
                isReplying.value = false;
                isImRecovering.value = false;
                currentAssistantMessageId.value = '';
                return;
            }
        } catch (e) {
            console.warn('[Continue Stream] recovery poll failed:', e);
        }
        if (attempts >= RECOVER_POLL_MAX_ATTEMPTS) {
            // The IM reply never completed — don't hide it; surface the standard
            // stream-failure message (reuses the existing i18n key, no raw HTTP code).
            MessagePlugin.error(t('error.streamFailed'));
            isReplying.value = false;
            isImRecovering.value = false;
            currentAssistantMessageId.value = '';
            return;
        }
        recoverPollTimer = setTimeout(poll, RECOVER_POLL_INTERVAL);
    };
    recoverPollTimer = setTimeout(poll, RECOVER_POLL_INTERVAL);
};

// Watch for stream errors and show message
watch(error, (newError) => {
    if (!newError) return;
    // A failed attach to an in-flight IM reply isn't a real error — the answer is
    // produced on the IM side and never streams here. Recover quietly by polling to
    // completion instead of flashing a "stream failed" toast. Web/api replies fall
    // through to the normal error toast below, unchanged.
    if (isAttachingImStream.value) {
        isAttachingImStream.value = false;
        recoverIncompleteMessage();
        return;
    }
    MessagePlugin.error(newError);
    isReplying.value = false;
    loading.value = false;
    // 清空当前 assistant message ID
    currentAssistantMessageId.value = '';
});

onChunk((data) => {
    if (data.response_type === 'session_title') {
        const title = data.content || data.data?.title;
        if (title && data.data?.session_id) {
            console.log('[Session Title Update]', {
                session_id: data.data.session_id,
                title: title,
            });
            usemenuStore.updatasessionTitle(data.data.session_id, title);
            usemenuStore.changeIsFirstSession(false);
            notifySessionMutation({
                sessionId: data.data.session_id,
                patch: { title },
            });
        }
        return;
    }
    processStreamChunk(data);
});

const handleSessionMutation = (event) => {
    const detail = event.detail;
    if (detail?.sessionId !== session_id.value) return;

    if (detail.patch) {
        currentSession.value = {
            ...(currentSession.value || { id: session_id.value }),
            ...detail.patch,
        };
    }
    if (detail.messagesCleared) {
        messagesList.splice(0);
        created_at.value = '';
        hasMoreHistory.value = true;
        historyLoadingMore.value = false;
        const key = publishedExpertStorageKey();
        if (key) localStorage.removeItem(key);
        fetchSuggestedQuestionsIfNeeded();
    }
};

onBeforeMount(async () => {
    // 若从智能体列表点击共享智能体进入，URL 带 agent_id 与 source_tenant_id，同步到 store
    if (!props.embeddedMode) {
        const agentIdFromQuery = props.agentId || (route.query.agent_id && String(route.query.agent_id));
        const sourceTenantIdFromQuery = route.query.source_tenant_id && String(route.query.source_tenant_id);
        if (agentIdFromQuery && sourceTenantIdFromQuery) {
            useSettingsStoreInstance.selectAgent(agentIdFromQuery, sourceTenantIdFromQuery);
        } else if (agentIdFromQuery) {
            useSettingsStoreInstance.selectAgent(agentIdFromQuery, null);
        }

        if (props.kbIds && props.kbIds.length > 0) {
            useSettingsStoreInstance.selectKnowledgeBases(props.kbIds);
        }
    }

    // 必须在 Input-field onMounted 之前完成：按 session.last_request_state 恢复输入栏
    await loadSessionAndHydrate(session_id.value);
});

onMounted(async () => {
    window.addEventListener(SESSION_MUTATION_EVENT, handleSessionMutation);
    messagesList.splice(0);

    // 初始化状态：加载历史消息时不应显示loading
    loading.value = false;
    isReplying.value = false;

    if (firstQuery.value) {
        scrollLock.value = true;
        historyLoading.value = false;
        if (firstModelId.value) {
            useSettingsStoreInstance.updateConversationModels({
                summaryModelId: firstModelId.value,
                selectedChatModelId: firstModelId.value,
                rerankModelId: '',
            });
        }
        if (firstPublishedExpert.value || firstPublishedExpertRouteMode.value === 'auto') {
            handlePublishedExpertSend(
                firstQuery.value,
                firstModelId.value || '',
                firstPublishedExpert.value,
                firstPublishedExpertRouteMode.value,
            );
        } else {
            sendMsg(firstQuery.value, firstModelId.value || '', firstMentionedItems.value || [], firstImageFiles.value || [], firstAttachmentFiles.value || []);
        }
        usemenuStore.changeFirstQuery('', [], '', [], []);
        usemenuStore.changeFirstPublishedExpert(null);
        usemenuStore.changeFirstPublishedExpertRouteMode('manual');
    } else {
        scrollLock.value = false;
        hasMoreHistory.value = true;
        historyLoadingMore.value = false;
        let data = {
            session_id: session_id.value,
            created_at: '',
            limit: limit.value
        }
        await getmsgList(data)
        await restorePublishedExpertRuns()
    }
})
const clearData = () => {
    stopStream();
    referencesDrawer.close();
    isReplying.value = false;
    fullContent.value = '';
    // Stop any IM-reply recovery poll for the session we're leaving/switching.
    if (recoverPollTimer) { clearTimeout(recoverPollTimer); recoverPollTimer = null; }
    isImRecovering.value = false;
}
onUnmounted(() => {
    window.removeEventListener(SESSION_MUTATION_EVENT, handleSessionMutation);
    if (recoverPollTimer) { clearTimeout(recoverPollTimer); recoverPollTimer = null; }
});
onBeforeRouteLeave((to, from, next) => {
    clearData()
    // 离开聊天会话 → 还原"用户全局默认"，避免旧会话的请求态泄漏到新建对话。
    useSettingsStoreInstance.restoreDefaultsIfSnapshotted();
    next()
})
onBeforeRouteUpdate((to, from, next) => {
    clearData()
    // 仅"会话 → 会话"会落到这里；跨会话覆盖的还原放到 route.params 的 watch 里，
    // 因为新会话的 getSession 也在那边触发，便于保证 restore→snapshot→apply 顺序。
    next()
})

defineExpose({
    triggerSend(question) {
        inputFieldRef.value?.triggerSend(question);
    },
})
</script>
<style lang="less" scoped>
.chat {
    font-size: 20px;
    // 右侧不留 padding，滚动条贴到内容区最右缘
    padding: 0 0 20px 20px;
    box-sizing: border-box;
    flex: 1;
    // The parent .platform-route-outlet is a flex column with min-height:0
    // and overflow:hidden — we also need min-height:0 here so that our
    // own flex:1 child (.chat_scroll_box) can shrink below its content
    // height and scroll instead of pushing .input-container out of view.
    min-height: 0;
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    max-width: calc(100vw - 260px);
    min-width: 400px;

    &.is-sidebar-collapsed {
        max-width: calc(100vw - 60px);
    }

    &.is-embedded {
        max-width: 100%;
        min-width: 100%;
        padding: 0;
        overflow-x: hidden;
    }

    &:not(.is-embedded) {
        @media (min-width: 960px) {
            transition: padding-right 0.3s cubic-bezier(0.22, 0.61, 0.36, 1);
        }
    }

    &.has-references-panel:not(.is-embedded) {
        @media (min-width: 960px) {
            padding-right: 420px;
            box-sizing: border-box;

            .chat_scroll_box {
                padding-top: 0;
            }
        }
    }

    &.has-expert-artifact-panel:not(.is-embedded) {
        @media (min-width: 960px) {
            padding-right: min(42vw, 640px);
            box-sizing: border-box;
        }
    }

    &.is-embedded :deep(.answers-input) {
        position: relative;
        transform: translateX(0);
        width: 100%;
        left: 0;
        bottom: auto;
        display: flex;
        justify-content: center;
    }

    &.is-embedded :deep(.control-bar) {
        justify-content: flex-end;
    }

    &:not(.is-embedded) :deep(.answers-input) {
        position: static;
        transform: translateX(0);

        .t-textarea__inner {
            width: 100% !important;
        }
    }

    &.is-embedded :deep(.answers-input) .t-textarea__inner {
        width: 100% !important;
        min-height: 48px !important;
        padding: 10px 14px 48px 14px;
    }
}

.chat_scroll_box {
    flex: 1;
    // Without min-height: 0, a flex-column child defaults to min-height: auto
    // and expands to fit all inner content. When there are many messages,
    // that pushes .input-container out of the viewport. Clamping min-height
    // to 0 lets overflow-y: auto take effect so the messages scroll inside
    // this box instead of stretching it.
    min-height: 0;
    width: 100%;
    padding-top: 8px;
    box-sizing: border-box;
    overflow-y: auto;
    // 使用系统原生滚动条（macOS 滚动时自动显示 overlay 滚动条，类似 ChatGPT）
    scrollbar-width: auto;
    scrollbar-color: auto;
}

// 深色模式下 theme.css 对 * 做了 webkit 滚动条着色，这里恢复为系统默认
:global(:root[theme-mode="dark"]) .chat_scroll_box {
    &::-webkit-scrollbar-thumb {
        background-color: initial !important;
    }

    &::-webkit-scrollbar-thumb:hover {
        background-color: initial !important;
    }

    &::-webkit-scrollbar-track {
        background-color: initial !important;
    }
}

.scroll-to-bottom-btn {
    position: absolute;
    left: 50%;
    transform: translateX(-50%);
    bottom: 140px;
    z-index: 10;
    width: 36px;
    height: 36px;
    border-radius: 50%;
    background: var(--td-bg-color-container);
    border: 1px solid var(--td-component-stroke);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    color: var(--td-text-color-secondary);
    transition: all 0.2s ease;

    &:hover {
        background: var(--td-bg-color-container-hover);
        color: var(--td-text-color-primary);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    }

    &:active {
        transform: translateX(-50%) scale(0.92);
    }
}

.scroll-btn-fade-enter-active,
.scroll-btn-fade-leave-active {
    transition: opacity 0.2s ease, transform 0.2s ease;
}

.scroll-btn-fade-enter-from,
.scroll-btn-fade-leave-to {
    opacity: 0;
    transform: translateX(-50%) translateY(8px);
}

@keyframes contentFadeIn {
    from {
        opacity: 0;
        transform: translateY(6px);
    }

    to {
        opacity: 1;
        transform: translateY(0);
    }
}

.msg-skeleton-list {
    display: flex;
    flex-direction: column;
    gap: 20px;
    max-width: 960px;
    padding: 16px 0;
    animation: contentFadeIn 0.3s ease-out;
}

.msg-skeleton-user {
    display: flex;
    justify-content: flex-end;
}

.msg-skeleton-bot {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-left: 4px;
}

.input-container {
    min-height: 115px;
    flex-shrink: 0;
    margin: 0 auto;
    width: 100%;
    max-width: 960px;
    box-sizing: border-box;
    position: relative;

    &.is-embedded {
        max-width: 100%;
        width: 100%;
        margin: 0;
        padding: 12px 16px 16px;
        min-height: auto;
        box-sizing: border-box;
        overflow-x: hidden;
    }
}

.msg_list {
    display: flex;
    flex-direction: column;
    gap: 16px;
    max-width: 960px;
    flex: 1;
    margin: 0 auto;
    width: 100%;

    /*
      给每条消息加 layout/style containment：
      - 一条消息的内部布局变化不再让浏览器去 invalidate 整个文档，
        这是修掉"hover 到 session 列表也变白"那个问题的关键。
      - 不要再用 content-visibility: auto / contain-intrinsic-size：
        agent 消息真实高度差异巨大（几百 ~ 数千 px），估的占位高度会让消息进入视口时
        反复发生"占位 -> 真实高度"的大幅 layout shift + 首次 paint 滞后，
        反而在向上滚动时制造"未画完"的白屏闪烁。
        当前 handleMsgList 全流程 ~50ms，根本无需跳过渲染，老老实实正常渲染最稳。
      - 不开 contain: paint：AgentStreamDisplay 里有 tooltip / popover 等会溢出的浮层，
        paint containment 会把它们裁掉。
    */
    .msg-item-wrapper {
        contain: layout style;
    }

    .botanswer_laoding_gif {
        width: 24px;
        height: 18px;
        margin-left: 16px;
    }

    .chat-global-wait {
        display: flex;
        align-items: center;
        min-height: 28px;
        padding-left: 4px;
    }

    .chat-global-wait__spinner {
        width: 12px;
        height: 12px;
        box-sizing: border-box;
        border: 1.5px solid var(--td-component-stroke);
        border-top-color: var(--td-text-color-secondary);
        border-radius: 50%;
        animation: chatGlobalWaitSpin 0.8s linear infinite;
    }

    .chat-empty-suggestions {
        width: 100%;
        margin-top: auto;
    }
}

@keyframes chatGlobalWaitSpin {
    to {
        transform: rotate(360deg);
    }
}

@media (prefers-reduced-motion: reduce) {
    .chat-global-wait__spinner {
        animation: none;
    }
}

@import '../../components/css/suggested-questions.less';

.suggested-questions-container {
    transition: min-height 0.3s @suggested-ease;
}

.suggested-questions-inner {
    animation: contentFadeIn 0.3s ease-out;
}

.sq-fade-enter-active,
.sq-fade-leave-active {
    transition: opacity 0.25s @suggested-ease;
}

.sq-fade-enter-from,
.sq-fade-leave-to {
    opacity: 0;
}
</style>
