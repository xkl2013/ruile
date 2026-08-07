const { saveSettings, getSettings } = require("../../utils/config");
const { createKnowledgeFromURL, listKnowledgeBases, listTenants } = require("../../utils/request");

function normalizeKnowledgeBases(response) {
  if (Array.isArray(response?.data)) {
    return response.data;
  }
  if (Array.isArray(response?.data?.list)) {
    return response.data.list;
  }
  if (Array.isArray(response?.knowledge_bases)) {
    return response.knowledge_bases;
  }
  return [];
}

function normalizeTenants(response) {
  if (Array.isArray(response?.data?.items)) {
    return response.data.items;
  }
  if (Array.isArray(response?.data)) {
    return response.data;
  }
  if (Array.isArray(response?.tenants)) {
    return response.tenants;
  }
  return [];
}

function formatBytes(bytes) {
  const value = Number(bytes || 0);
  if (!Number.isFinite(value) || value <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = value;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  const precision = size >= 10 || unitIndex === 0 ? 0 : 1;
  return `${size.toFixed(precision)} ${units[unitIndex]}`;
}

function buildStorageUsage(tenant) {
  if (!tenant) return null;

  const usage = tenant.storage_usage || {};
  const quotaBytes = Number(usage.quota_bytes ?? tenant.storage_quota ?? 0);
  const usedBytes = Number(usage.used_bytes ?? tenant.storage_used ?? 0);
  const warningThreshold = Number(usage.warning_threshold_percent ?? 80);
  const unlimited = Boolean(usage.unlimited || quotaBytes <= 0);

  if (unlimited) {
    return {
      status: "unlimited",
      statusClass: "storage-unlimited",
      usedText: formatBytes(usedBytes),
      quotaText: "Unlimited",
      remainingText: "Unlimited",
      percentText: "Unlimited",
      progressPercent: 0,
      notice: ""
    };
  }

  const remainingBytes = Math.max(0, Number(usage.remaining_bytes ?? quotaBytes - usedBytes));
  const rawPercent = Number(usage.usage_percent ?? (quotaBytes > 0 ? (usedBytes / quotaBytes) * 100 : 0));
  const usagePercent = Number.isFinite(rawPercent) ? rawPercent : 0;
  const status =
    usage.status || (usedBytes >= quotaBytes ? "exceeded" : usagePercent >= warningThreshold ? "warning" : "ok");
  const requiresIncrease = Boolean(usage.requires_quota_increase || status === "warning" || status === "exceeded");

  let notice = "";
  if (status === "exceeded") {
    notice = "Storage is full. Ask an administrator to increase the quota before importing more content.";
  } else if (requiresIncrease) {
    notice = `Storage is close to the quota (${warningThreshold.toFixed(0)}%). Ask an administrator to increase the quota soon.`;
  }

  return {
    status,
    statusClass: `storage-${status}`,
    usedText: formatBytes(usedBytes),
    quotaText: formatBytes(quotaBytes),
    remainingText: formatBytes(remainingBytes),
    percentText: `${usagePercent.toFixed(1)}%`,
    progressPercent: Math.max(0, Math.min(100, Math.round(usagePercent))),
    notice
  };
}

Page({
  data: {
    importing: false,
    knowledgeBases: [],
    knowledgeBaseNames: [],
    loading: false,
    needsSettings: false,
    selectedIndex: 0,
    selectedKnowledgeBaseId: "",
    selectedKnowledgeBaseName: "",
    statusMessage: "",
    storageBlocked: false,
    storageLoading: false,
    storageUsage: null,
    url: ""
  },

  onShow() {
    const settings = getSettings();
    const needsSettings = !settings.baseUrl || !settings.apiKey;
    if (settings.selectedKnowledgeBaseId) {
      this.setData({ selectedKnowledgeBaseId: settings.selectedKnowledgeBaseId });
    }
    this.setData({ needsSettings });
    if (needsSettings) {
      this.setData({ storageBlocked: false, storageUsage: null });
      return;
    }
    this.loadKnowledgeBases();
  },

  onUrlInput(event) {
    this.setData({ url: event.detail.value });
  },

  onKnowledgeBaseChange(event) {
    const selectedIndex = Number(event.detail.value);
    this.selectKnowledgeBase(selectedIndex);
  },

  onKnowledgeBaseTap(event) {
    this.selectKnowledgeBase(Number(event.currentTarget.dataset.index));
  },

  selectKnowledgeBase(selectedIndex) {
    const selected = this.data.knowledgeBases[selectedIndex];
    if (!selected) return;

    saveSettings({ selectedKnowledgeBaseId: selected.id });
    this.setData({
      selectedIndex,
      selectedKnowledgeBaseId: selected.id,
      selectedKnowledgeBaseName: selected.name
    });
  },

  openSettings() {
    wx.switchTab({ url: "/pages/settings/settings" });
  },

  async loadKnowledgeBases() {
    const settings = getSettings();
    if (!settings.baseUrl || !settings.apiKey) {
      this.setData({ needsSettings: true });
      return;
    }

    this.setData({ loading: true, statusMessage: "" });
    try {
      const response = await listKnowledgeBases();
      const knowledgeBases = normalizeKnowledgeBases(response);
      const knowledgeBaseNames = knowledgeBases.map((item) => item.name || item.id);
      const settings = getSettings();
      const selectedIndex = Math.max(
        0,
        knowledgeBases.findIndex((item) => item.id === settings.selectedKnowledgeBaseId)
      );
      const selected = knowledgeBases[selectedIndex];
      this.setData({
        knowledgeBases,
        knowledgeBaseNames,
        selectedIndex,
        selectedKnowledgeBaseId: selected?.id || "",
        selectedKnowledgeBaseName: selected?.name || "",
        statusMessage: knowledgeBases.length
          ? `Loaded ${knowledgeBases.length} knowledge bases.`
          : "No knowledge bases found."
      });
      if (selected?.id) {
        saveSettings({ selectedKnowledgeBaseId: selected.id });
      }
      if (typeof this.loadTenantStorage === "function") {
        this.loadTenantStorage();
      }
    } catch (error) {
      wx.showModal({
        title: "Load failed",
        content: error.message,
        showCancel: false
      });
    } finally {
      this.setData({ loading: false });
    }
  },

  async loadTenantStorage() {
    this.setData({ storageLoading: true });
    try {
      const response = await listTenants();
      const tenant = normalizeTenants(response)[0];
      const storageUsage = buildStorageUsage(tenant);
      this.setData({
        storageUsage,
        storageBlocked: storageUsage?.status === "exceeded"
      });
    } catch (error) {
      this.setData({
        storageBlocked: false,
        storageUsage: null
      });
    } finally {
      this.setData({ storageLoading: false });
    }
  },

  async importURL() {
    this.setData({ importing: true });
    try {
      await createKnowledgeFromURL(this.data.selectedKnowledgeBaseId, this.data.url.trim(), false);
      this.setData({ url: "" });
      wx.showToast({ title: "Imported", icon: "success" });
    } catch (error) {
      wx.showModal({
        title: "Import failed",
        content: /storage quota/i.test(error.message)
          ? `${error.message}. Ask an administrator to increase the storage quota.`
          : error.message,
        showCancel: false
      });
    } finally {
      this.setData({ importing: false });
      if (typeof this.loadTenantStorage === "function") {
        this.loadTenantStorage();
      }
    }
  }
});
