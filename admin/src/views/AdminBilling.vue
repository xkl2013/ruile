<template>
  <section class="billing-admin">
    <div class="billing-admin__toolbar">
      <div>
        <h2>订阅与计费</h2>
        <p>管理套餐及其价格，并核对用户或企业的积分主体与模型用量账本。模型配置和模型价格统一在「模型」页面维护。</p>
      </div>
      <t-button variant="outline" :loading="loading" @click="loadAll">
        <template #icon><t-icon name="refresh" /></template>
        刷新
      </t-button>
    </div>

    <t-alert
      v-if="errorMessage"
      theme="error"
      :message="errorMessage"
      class="billing-admin__alert"
    />

    <t-tabs v-model="activeTab" class="billing-admin__tabs">
      <t-tab-panel value="plans" label="套餐">
        <div class="billing-admin__tab-actions">
          <t-button theme="primary" @click="openPlanDialog()">
            <template #icon><t-icon name="add" /></template>
            新增套餐
          </t-button>
        </div>
        <div class="billing-admin__table-wrap">
          <table class="billing-admin__table">
            <thead>
              <tr>
                <th>套餐</th>
                <th>版本</th>
                <th>空间类型</th>
                <th>包含存储</th>
                <th>周期积分</th>
                <th>价格</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="plan in plans" :key="plan.id">
                <td><strong>{{ plan.name }}</strong><span>{{ plan.code }}</span></td>
                <td>{{ plan.edition }}</td>
                <td>{{ planSpaceTypeLabel(plan.space_type) }}</td>
                <td>{{ formatBytes(plan.included_storage_bytes) }}</td>
                <td>{{ formatPoints(plan.included_point_micros) }}</td>
                <td>
                  <div v-if="pricesByPlanID.get(plan.id)?.length" class="billing-admin__plan-prices">
                    <div v-for="price in pricesByPlanID.get(plan.id)" :key="price.id" class="billing-admin__plan-price">
                      <strong>{{ formatMoney(price.amount_minor, price.currency) }}</strong>
                      <span>
                        {{ billingIntervalLabel(price.billing_interval) }} · {{ price.code }}
                        <t-tag v-if="price.is_default" theme="success" variant="light" size="small">默认</t-tag>
                      </span>
                    </div>
                  </div>
                  <span v-else>暂未设置</span>
                </td>
                <td>
                  <t-tag :theme="plan.status === 'active' ? 'success' : 'default'" variant="light">
                    {{ plan.status === 'active' ? '启用' : '停用' }}
                  </t-tag>
                </td>
                <td>
                  <div class="billing-admin__row-actions">
                    <t-button variant="text" size="small" @click="openPlanDialog(plan)">
                      编辑套餐
                    </t-button>
                    <t-button variant="text" size="small" @click="openPriceDialog(plan)">
                      新增价格
                    </t-button>
                  </div>
                </td>
              </tr>
              <tr v-if="!loading && plans.length === 0">
                <td colspan="8" class="billing-admin__empty">暂无套餐</td>
              </tr>
            </tbody>
          </table>
        </div>
      </t-tab-panel>

      <t-tab-panel value="accounts" label="计费主体">
        <div class="billing-admin__table-wrap">
          <table class="billing-admin__table">
            <thead>
              <tr>
                <th>计费主体</th>
                <th>主体类型</th>
                <th>积分余额</th>
                <th>关联空间</th>
                <th>账户创建时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="account in accounts" :key="account.id">
                <td>
                  <strong>{{ account.subject_name || account.tenant_name }}</strong>
                  <span v-if="account.subject_contact">{{ account.subject_contact }}</span>
                  <span v-else>#{{ account.subject_id || account.tenant_id }}</span>
                </td>
                <td>{{ billingSubjectTypeLabel(account.subject_type) }}</td>
                <td><strong class="billing-admin__points">{{ formatPoints(account.balance_point_micros) }}</strong></td>
                <td>{{ account.tenant_count || 1 }} 个空间</td>
                <td>{{ formatDate(account.created_at) }}</td>
              </tr>
              <tr v-if="!loading && accounts.length === 0">
                <td colspan="5" class="billing-admin__empty">暂无积分账户</td>
              </tr>
            </tbody>
          </table>
        </div>
      </t-tab-panel>

      <t-tab-panel value="purchase-items" label="商品">
        <div class="billing-admin__tab-actions">
          <t-button theme="primary" @click="openPurchaseItemDialog()">
            <template #icon><t-icon name="add" /></template>
            新增商品
          </t-button>
        </div>
        <div class="billing-admin__table-wrap">
          <table class="billing-admin__table billing-admin__table--wide">
            <thead>
              <tr>
                <th>商品</th>
                <th>类型</th>
                <th>适用空间</th>
                <th>价格</th>
                <th>权益</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in purchaseItems" :key="item.id">
                <td><strong>{{ item.name }}</strong><span>{{ item.code }}</span></td>
                <td>{{ purchaseItemTypeLabel(item.item_type) }}</td>
                <td>{{ editionScopeLabel(item.edition_scope) }}</td>
                <td><strong>{{ formatMoney(item.amount_cents, item.currency) }}</strong></td>
                <td>
                  <strong v-if="item.item_type === 'topup'">{{ formatPoints(item.credit_point_micros) }}</strong>
                  <strong v-else>{{ formatBytes(item.storage_quota_bytes) }}</strong>
                  <span>长期有效</span>
                </td>
                <td>
                  <t-tag :theme="item.status === 'active' ? 'success' : 'default'" variant="light">
                    {{ item.status === 'active' ? '启用' : '停用' }}
                  </t-tag>
                </td>
                <td>
                  <t-button variant="text" size="small" @click="openPurchaseItemDialog(item)">
                    编辑
                  </t-button>
                </td>
              </tr>
              <tr v-if="!loading && purchaseItems.length === 0">
                <td colspan="7" class="billing-admin__empty">暂无商品</td>
              </tr>
            </tbody>
          </table>
        </div>
      </t-tab-panel>

      <t-tab-panel value="manual-operations" label="人工开通">
        <div class="billing-admin__offline-note">
          <span>在线支付当前未配置。此处仅用于后台人工开通合同、调整积分或存储，所有操作都会生成已支付的人工订单。</span>
          <t-button theme="primary" @click="openManualOrderDialog">
            <template #icon><t-icon name="add" /></template>
            新建人工操作
          </t-button>
        </div>
      </t-tab-panel>

      <t-tab-panel value="payment-orders" label="订单">
        <div class="billing-admin__table-wrap">
          <table class="billing-admin__table billing-admin__table--wide">
            <thead>
              <tr>
                <th>订单号</th>
                <th>工作空间</th>
                <th>类型</th>
                <th>权益</th>
                <th>金额</th>
                <th>状态</th>
                <th>完成时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="order in paymentOrders" :key="order.id">
                <td><strong>{{ order.order_no }}</strong><span>{{ order.provider }} · {{ order.payment_method }}</span></td>
                <td><strong>{{ order.tenant_name }}</strong><span>#{{ order.tenant_id }}</span></td>
                <td>{{ orderTypeLabel(order.order_type) }}</td>
                <td>
                  <strong v-if="order.order_type === 'topup'">{{ formatPoints(order.credit_point_micros) }}</strong>
                  <strong v-else-if="order.order_type === 'storage_addon'">{{ formatBytes(order.storage_quota_bytes) }}</strong>
                  <strong v-else>{{ order.plan_name || '-' }}</strong>
                  <span v-if="order.billing_interval">{{ order.billing_interval }}</span>
                </td>
                <td>{{ formatMoney(order.amount_cents, order.currency) }}</td>
                <td>
                  <t-tag :theme="order.status === 'paid' ? 'success' : 'default'" variant="light">
                    {{ order.status === 'paid' ? '已完成' : order.status }}
                  </t-tag>
                </td>
                <td>{{ formatDate(order.paid_at || order.created_at) }}</td>
              </tr>
              <tr v-if="!loading && paymentOrders.length === 0">
                <td colspan="7" class="billing-admin__empty">暂无订单</td>
              </tr>
            </tbody>
          </table>
        </div>
      </t-tab-panel>

      <t-tab-panel value="service-prices" label="MCP定价">
        <div class="billing-admin__tab-actions">
          <t-button theme="primary" @click="openServicePriceDialog">
            <template #icon><t-icon name="add" /></template>
            新增MCP价格版本
          </t-button>
        </div>
        <div class="billing-admin__table-wrap">
          <table class="billing-admin__table">
            <thead>
              <tr>
                <th>MCP工具</th>
                <th>计费模式</th>
                <th>价格</th>
                <th>服务倍率</th>
                <th>版本</th>
                <th>状态</th>
                <th>生效时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="price in servicePrices" :key="price.id">
                <td><strong>{{ price.service_name || price.service_code }}</strong><span>{{ price.service_code }}</span></td>
                <td>{{ price.pricing_mode === 'unit' ? `按${price.unit_name || '单位'}` : '按次' }}</td>
                <td>{{ formatServicePrice(price) }}</td>
                <td>{{ formatMultiplier(price.service_multiplier_ppm) }}</td>
                <td>v{{ price.version }}</td>
                <td>
                  <t-tag :theme="price.status === 'active' ? 'success' : 'default'" variant="light">
                    {{ price.status }}
                  </t-tag>
                </td>
                <td>{{ formatDate(price.effective_at) }}</td>
              </tr>
              <tr v-if="!loading && servicePrices.length === 0">
                <td colspan="7" class="billing-admin__empty">暂无服务价格</td>
              </tr>
            </tbody>
          </table>
        </div>
      </t-tab-panel>

      <t-tab-panel value="usage-ledgers" label="用量账本">
        <div class="billing-admin__table-wrap">
          <table class="billing-admin__table billing-admin__table--wide">
            <thead>
              <tr>
                <th>工作空间</th>
                <th>服务 / 模型</th>
                <th>Token</th>
                <th>计费积分</th>
                <th>价格版本</th>
                <th>状态</th>
                <th>计费时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="ledger in usageLedgers" :key="ledger.id">
                <td><strong>{{ ledger.tenant_name }}</strong><span>#{{ ledger.tenant_id }}</span></td>
                <td><strong>{{ ledger.service_code || '-' }}</strong><span>{{ ledger.model_key || ledger.provider || '-' }}</span></td>
                <td>
                  <strong>{{ ledger.input_tokens }} / {{ ledger.output_tokens }}</strong>
                  <span>输入 / 输出，缓存 {{ ledger.cached_tokens }}</span>
                </td>
                <td>{{ formatPoints(ledger.billed_point_micros) }}</td>
                <td>{{ ledger.pricing_version ? `v${ledger.pricing_version}` : '-' }}</td>
                <td>
                  <t-tag :theme="ledgerStatusTheme(ledger.status)" variant="light">
                    {{ ledger.status }}
                  </t-tag>
                  <span v-if="ledger.failure_code">{{ ledger.failure_code }}</span>
                </td>
                <td>{{ formatDate(ledger.billing_at) }}</td>
              </tr>
              <tr v-if="!loading && usageLedgers.length === 0">
                <td colspan="7" class="billing-admin__empty">暂无模型用量账本</td>
              </tr>
            </tbody>
          </table>
        </div>
      </t-tab-panel>

      <t-tab-panel value="storage-transactions" label="存储流水">
        <div class="billing-admin__table-wrap">
          <table class="billing-admin__table billing-admin__table--wide">
            <thead>
              <tr>
                <th>工作空间</th>
                <th>操作</th>
                <th>变动</th>
                <th>变动后用量</th>
                <th>操作者</th>
                <th>引用号</th>
                <th>时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in storageTransactions" :key="row.id">
                <td><strong>{{ row.tenant_name }}</strong><span>#{{ row.tenant_id }}</span></td>
                <td>{{ row.operation }}</td>
                <td :class="{ 'billing-admin__storage-release': row.amount_bytes < 0 }">
                  {{ row.amount_bytes > 0 ? '+' : '' }}{{ formatBytes(row.amount_bytes) }}
                </td>
                <td>{{ formatBytes(row.storage_used_after_bytes) }}</td>
                <td>{{ row.actor_user_id || '-' }}</td>
                <td><span>{{ row.ref_no }}</span></td>
                <td>{{ formatDate(row.created_at) }}</td>
              </tr>
              <tr v-if="!loading && storageTransactions.length === 0">
                <td colspan="7" class="billing-admin__empty">暂无存储流水</td>
              </tr>
            </tbody>
          </table>
        </div>
      </t-tab-panel>
    </t-tabs>

    <t-dialog
      v-model:visible="planDialogVisible"
      :header="planEditingID ? '编辑套餐' : '新增套餐'"
      width="720px"
      :confirm-btn="{ content: planEditingID ? '保存套餐' : '创建套餐', loading: savingPlan }"
      :cancel-btn="{ content: '取消', disabled: savingPlan }"
      @confirm="savePlan"
    >
      <t-form label-align="top">
        <t-alert
          v-if="!planEditingID"
          theme="info"
          message="套餐创建后，可在套餐列表中继续新增价格版本。已有订阅会保留原套餐快照。"
          class="billing-admin__dialog-alert"
        />
        <div class="billing-admin__form-grid">
          <t-form-item label="套餐名称" required>
            <t-input v-model="planForm.name" placeholder="例如 企业团队版" />
          </t-form-item>
          <t-form-item label="套餐代码" required>
            <t-input v-model="planForm.code" :disabled="Boolean(planEditingID)" placeholder="例如 enterprise_team" />
          </t-form-item>
          <t-form-item label="版本类型" required>
            <t-select v-model="planForm.edition" :disabled="Boolean(planEditingID)" @change="syncPlanSpaceType">
              <t-option value="personal" label="个人版" />
              <t-option value="enterprise" label="企业版" />
              <t-option value="legacy" label="历史兼容版" />
            </t-select>
          </t-form-item>
          <t-form-item label="空间类型" required>
            <t-select v-model="planForm.spaceType" disabled>
              <t-option value="personal" label="个人空间" />
              <t-option value="organization" label="企业空间" />
              <t-option value="legacy" label="历史空间" />
            </t-select>
          </t-form-item>
          <t-form-item label="包含存储（GB）" required>
            <t-input-number v-model="planForm.storageGB" :min="0" :decimal-places="2" />
          </t-form-item>
          <t-form-item label="周期积分" required>
            <t-input-number v-model="planForm.periodPoints" :min="0" :decimal-places="0" />
          </t-form-item>
          <t-form-item label="积分计费倍率" required>
            <t-input-number v-model="planForm.multiplier" :min="0.000001" :decimal-places="6" />
          </t-form-item>
          <t-form-item label="状态" required>
            <t-select v-model="planForm.status">
              <t-option value="active" label="启用" />
              <t-option value="disabled" label="停用" />
            </t-select>
          </t-form-item>
        </div>
        <div class="billing-admin__switch-row">
          <span>
            <strong>对外展示</strong>
            <small>关闭后不可用于新的公开订阅，但不影响已有订阅。</small>
          </span>
          <t-switch v-model="planForm.isPublic" />
        </div>
        <t-form-item label="套餐说明">
          <t-textarea v-model="planForm.description" :maxlength="512" placeholder="说明套餐适用范围和权益" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="priceDialogVisible"
      header="新增套餐价格版本"
      width="620px"
      :confirm-btn="{ content: '创建价格版本', loading: savingPrice }"
      :cancel-btn="{ content: '取消', disabled: savingPrice }"
      @confirm="savePriceVersion"
    >
      <t-form label-align="top">
        <t-alert
          theme="info"
          message="价格版本创建后不可覆盖修改；如需调价，请创建新版本并设置为默认。"
          class="billing-admin__dialog-alert"
        />
        <div class="billing-admin__form-grid">
          <t-form-item label="关联套餐" required>
            <t-select v-model="priceForm.planID" disabled>
              <t-option v-for="plan in plans" :key="plan.id" :value="plan.id" :label="plan.name" />
            </t-select>
          </t-form-item>
          <t-form-item label="价格代码" required>
            <t-input v-model="priceForm.code" placeholder="例如 enterprise_team_cny_v2" />
          </t-form-item>
          <t-form-item label="计费周期" required>
            <t-select v-model="priceForm.billingInterval">
              <t-option value="none" label="一次性 / 无周期" />
              <t-option value="trial" label="试用" />
              <t-option value="month" label="月付" />
              <t-option value="year" label="年付" />
              <t-option value="manual" label="人工合同" />
            </t-select>
          </t-form-item>
          <t-form-item label="价格（元）" required>
            <t-input-number v-model="priceForm.amountCNY" :min="0" :decimal-places="2" />
          </t-form-item>
          <t-form-item label="状态" required>
            <t-select v-model="priceForm.status">
              <t-option value="active" label="启用" />
              <t-option value="draft" label="草稿" />
              <t-option value="disabled" label="停用" />
            </t-select>
          </t-form-item>
        </div>
        <div class="billing-admin__switch-row">
          <span>
            <strong>设为默认价格</strong>
            <small>同一套餐只能有一个默认价格，设为默认会取消旧默认价格。</small>
          </span>
          <t-switch v-model="priceForm.isDefault" />
        </div>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="servicePriceDialogVisible"
      header="新增MCP工具价格版本"
      width="620px"
      :confirm-btn="{ content: '保存价格', loading: savingServicePrice }"
      :cancel-btn="{ content: '取消', disabled: savingServicePrice }"
      @confirm="saveServicePrice"
    >
      <t-form label-align="top">
        <div class="billing-admin__form-grid">
          <t-form-item label="服务代码">
            <t-input model-value="mcp.tool_call" disabled />
          </t-form-item>
          <t-form-item label="计费模式">
            <t-input model-value="按次" disabled />
          </t-form-item>
          <t-form-item label="价格（元 / 次）" required>
            <t-input-number v-model="servicePriceForm.priceCNY" :min="0" :decimal-places="6" />
          </t-form-item>
          <t-form-item label="服务倍率">
            <t-input-number v-model="servicePriceForm.multiplier" :min="0.000001" :decimal-places="6" />
          </t-form-item>
        </div>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="purchaseItemDialogVisible"
      :header="purchaseItemEditingID ? '编辑商品' : '新增商品'"
      width="680px"
      :confirm-btn="{ content: purchaseItemEditingID ? '保存商品' : '创建商品', loading: savingPurchaseItem }"
      :cancel-btn="{ content: '取消', disabled: savingPurchaseItem }"
      @confirm="savePurchaseItem"
    >
      <t-form label-align="top">
        <div class="billing-admin__form-grid">
          <t-form-item label="商品名称" required>
            <t-input v-model="purchaseItemForm.name" placeholder="例如 10,000 积分" />
          </t-form-item>
          <t-form-item label="商品代码" required>
            <t-input v-model="purchaseItemForm.code" placeholder="例如 credit_10000_points" />
          </t-form-item>
          <t-form-item label="商品类型" required>
            <t-select v-model="purchaseItemForm.itemType">
              <t-option value="topup" label="积分包" />
              <t-option value="storage_addon" label="存储包" />
            </t-select>
          </t-form-item>
          <t-form-item label="适用空间" required>
            <t-select v-model="purchaseItemForm.editionScope">
              <t-option value="all" label="个人和企业" />
              <t-option value="personal" label="个人" />
              <t-option value="enterprise" label="企业" />
            </t-select>
          </t-form-item>
          <t-form-item label="价格（元）" required>
            <t-input-number v-model="purchaseItemForm.priceCNY" :min="0" :decimal-places="2" />
          </t-form-item>
          <t-form-item v-if="purchaseItemForm.itemType === 'topup'" label="积分数量" required>
            <t-input-number v-model="purchaseItemForm.creditPoints" :min="1" :decimal-places="0" />
          </t-form-item>
          <t-form-item v-else label="存储容量（GB）" required>
            <t-input-number v-model="purchaseItemForm.storageGB" :min="0.01" :decimal-places="2" />
          </t-form-item>
          <t-form-item label="状态" required>
            <t-select v-model="purchaseItemForm.status">
              <t-option value="active" label="启用" />
              <t-option value="disabled" label="停用" />
            </t-select>
          </t-form-item>
          <t-form-item label="排序">
            <t-input-number v-model="purchaseItemForm.sortOrder" :min="0" :decimal-places="0" />
          </t-form-item>
        </div>
        <t-form-item label="商品说明">
          <t-textarea v-model="purchaseItemForm.description" :maxlength="512" placeholder="说明该商品适用的权益和范围" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="manualOrderDialogVisible"
      header="新建人工操作"
      width="680px"
      :confirm-btn="{ content: '确认并生效', loading: savingManualOrder }"
      :cancel-btn="{ content: '取消', disabled: savingManualOrder }"
      @confirm="saveManualOrder"
    >
      <t-form label-align="top">
        <t-alert theme="warning" message="操作会立即生效并生成不可修改的人工订单，请确认目标工作空间和权益数值。" class="billing-admin__dialog-alert" />
        <div class="billing-admin__form-grid">
          <t-form-item label="目标工作空间" required>
            <t-select v-model="manualOrderForm.tenantID" filterable placeholder="选择工作空间">
              <t-option
                v-for="account in accounts"
                :key="account.tenant_id"
                :value="String(account.tenant_id)"
                :label="`${account.tenant_name} (#${account.tenant_id})`"
              />
            </t-select>
          </t-form-item>
          <t-form-item label="操作类型" required>
            <t-select v-model="manualOrderForm.orderType">
              <t-option value="manual_contract" label="合同订阅" />
              <t-option value="topup" label="积分调额" />
              <t-option value="storage_addon" label="存储调整" />
            </t-select>
          </t-form-item>
          <template v-if="manualOrderForm.orderType === 'manual_contract'">
            <t-form-item label="套餐" required>
              <t-select v-model="manualOrderForm.planID" placeholder="选择套餐">
                <t-option v-for="plan in contractPlans" :key="plan.id" :value="plan.id" :label="plan.name" />
              </t-select>
            </t-form-item>
            <t-form-item label="合同周期" required>
              <t-select v-model="manualOrderForm.billingInterval">
                <t-option value="trial" label="试用" />
                <t-option value="month" label="月付合同" />
                <t-option value="year" label="年付合同" />
                <t-option value="manual" label="自定义合同" />
              </t-select>
            </t-form-item>
            <t-form-item label="有效天数" required>
              <t-input-number v-model="manualOrderForm.periodDays" :min="1" :max="3660" :decimal-places="0" />
            </t-form-item>
          </template>
          <template v-else-if="manualOrderForm.orderType === 'topup'">
            <t-form-item label="积分调整" required>
              <t-input-number v-model="manualOrderForm.creditPoints" :decimal-places="0" />
            </t-form-item>
          </template>
          <template v-else>
            <t-form-item label="存储调整（GB）" required>
              <t-input-number v-model="manualOrderForm.storageGB" :decimal-places="2" />
            </t-form-item>
          </template>
          <t-form-item label="记录金额（元）">
            <t-input-number v-model="manualOrderForm.amountCNY" :min="0" :decimal-places="2" />
          </t-form-item>
        </div>
        <t-form-item label="操作说明" required>
          <t-textarea v-model="manualOrderForm.description" :maxlength="512" placeholder="例如：2026 年度合同续费，线下已确认款项" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  createBillingPlan,
  createBillingManualOrder,
  createBillingPriceVersion,
  createBillingPurchaseItem,
  createBillingServicePriceVersion,
  listBillingCreditAccounts,
  listBillingPaymentOrders,
  listBillingPlans,
  listBillingPrices,
  listBillingPurchaseItems,
  listBillingServicePrices,
  listBillingStorageTransactions,
  listBillingUsageLedgers,
  type BillingCreditAccountItem,
  type BillingPaymentOrderItem,
  type BillingPlanItem,
  type BillingPlanInput,
  type BillingPriceItem,
  type BillingPriceInput,
  type BillingPurchaseItem,
  type BillingServicePriceItem,
  type BillingStorageTransactionItem,
  type BillingUsageLedgerItem,
  updateBillingPlan,
  updateBillingPurchaseItem,
} from '@/api/system'

const activeTab = ref('plans')
const loading = ref(false)
const errorMessage = ref('')
const plans = ref<BillingPlanItem[]>([])
const prices = ref<BillingPriceItem[]>([])
const accounts = ref<BillingCreditAccountItem[]>([])
const purchaseItems = ref<BillingPurchaseItem[]>([])
const paymentOrders = ref<BillingPaymentOrderItem[]>([])
const usageLedgers = ref<BillingUsageLedgerItem[]>([])
const servicePrices = ref<BillingServicePriceItem[]>([])
const storageTransactions = ref<BillingStorageTransactionItem[]>([])
const planDialogVisible = ref(false)
const planEditingID = ref('')
const savingPlan = ref(false)
const priceDialogVisible = ref(false)
const savingPrice = ref(false)
const servicePriceDialogVisible = ref(false)
const savingServicePrice = ref(false)
const planForm = reactive({
  code: '',
  name: '',
  description: '',
  edition: 'personal' as 'personal' | 'enterprise' | 'legacy',
  spaceType: 'personal' as 'personal' | 'organization' | 'legacy',
  status: 'active' as 'active' | 'disabled',
  isPublic: true,
  storageGB: 1,
  periodPoints: 0,
  multiplier: 1,
})
const priceForm = reactive({
  planID: '',
  code: '',
  billingInterval: 'month' as 'none' | 'trial' | 'month' | 'year' | 'manual',
  amountCNY: 0,
  status: 'draft' as 'active' | 'draft' | 'disabled',
  isDefault: false,
})
const purchaseItemDialogVisible = ref(false)
const purchaseItemEditingID = ref('')
const savingPurchaseItem = ref(false)
const manualOrderDialogVisible = ref(false)
const savingManualOrder = ref(false)
const servicePriceForm = reactive({
  serviceCode: '',
  serviceName: '',
  pricingMode: 'call' as 'call' | 'unit',
  unitName: '',
  priceCNY: 0,
  multiplier: 1,
})
const purchaseItemForm = reactive({
  code: '',
  name: '',
  itemType: 'topup' as 'topup' | 'storage_addon',
  editionScope: 'all' as 'all' | 'personal' | 'enterprise',
  description: '',
  priceCNY: 0,
  creditPoints: 1000,
  storageGB: 20,
  status: 'active' as 'active' | 'disabled',
  sortOrder: 0,
})
const manualOrderForm = reactive({
  tenantID: '',
  orderType: 'manual_contract' as 'topup' | 'storage_addon' | 'manual_contract',
  planID: '',
  billingInterval: 'year' as 'trial' | 'month' | 'year' | 'manual',
  periodDays: 366,
  creditPoints: 0,
  storageGB: 0,
  amountCNY: 0,
  description: '',
})

const pricesByPlanID = computed(() => {
  const rows = new Map<string, BillingPriceItem[]>()
  for (const price of prices.value) {
    const planPrices = rows.get(price.plan_id) || []
    planPrices.push(price)
    rows.set(price.plan_id, planPrices)
  }
  return rows
})

const contractPlans = computed(() => {
  const selected = accounts.value.find((account) => String(account.tenant_id) === manualOrderForm.tenantID)
  if (!selected) return plans.value
  const expectedSpaceType = selected.space_type === 'organization' ? 'organization' : 'personal'
  return plans.value.filter((plan) => plan.space_type === expectedSpaceType)
})

function formatBytes(value: number) {
  const original = Number(value) || 0
  const bytes = Math.abs(original)
  if (bytes <= 0) return '按空间配置'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const formatted = `${(bytes / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`
  return original < 0 ? `-${formatted}` : formatted
}

function formatPoints(pointMicros: number) {
  return `${new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 2 }).format((Number(pointMicros) || 0) / 1_000_000)} 积分`
}

function formatMoney(amountMinor: number, currency: string) {
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: currency || 'CNY',
  }).format((Number(amountMinor) || 0) / 100)
}

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function ledgerStatusTheme(status: string): 'success' | 'warning' | 'danger' | 'default' {
  if (status === 'settled') return 'success'
  if (status === 'reconciliation' || status === 'unpriced') return 'danger'
  if (status === 'observed') return 'warning'
  return 'default'
}

function formatServicePrice(price: BillingServicePriceItem) {
  const nano = price.pricing_mode === 'unit' ? price.nanousd_per_unit : price.nanousd_per_call
  const unit = price.pricing_mode === 'unit' ? price.unit_name || '单位' : '次'
  return `${(Number(nano || 0) / 1_000_000_000).toFixed(6)} 元 / ${unit}`
}

function formatMultiplier(value: number) {
  return `${(Number(value || 0) / 1_000_000).toFixed(2)}x`
}

function purchaseItemTypeLabel(itemType: string) {
  return itemType === 'storage_addon' ? '存储包' : '积分包'
}

function editionScopeLabel(scope: string) {
  if (scope === 'personal') return '个人'
  if (scope === 'enterprise') return '企业'
  return '个人和企业'
}

function orderTypeLabel(orderType: string) {
  if (orderType === 'topup') return '积分调额'
  if (orderType === 'storage_addon') return '存储调整'
  if (orderType === 'manual_contract') return '合同订阅'
  return '套餐订阅'
}

function planSpaceTypeLabel(spaceType: string) {
  if (spaceType === 'personal') return '个人空间'
  if (spaceType === 'organization') return '企业空间'
  return '历史空间'
}

function billingIntervalLabel(interval: string) {
  if (interval === 'none') return '无周期'
  if (interval === 'trial') return '试用'
  if (interval === 'month') return '月付'
  if (interval === 'year') return '年付'
  if (interval === 'manual') return '人工合同'
  return interval
}

function billingSubjectTypeLabel(subjectType: string) {
  if (subjectType === 'user') return '个人用户'
  if (subjectType === 'enterprise') return '企业'
  return '历史主体'
}

function syncPlanSpaceType() {
  planForm.spaceType = planForm.edition === 'personal'
    ? 'personal'
    : planForm.edition === 'enterprise'
      ? 'organization'
      : 'legacy'
}

function openPlanDialog(plan?: BillingPlanItem) {
  planEditingID.value = plan?.id || ''
  Object.assign(planForm, {
    code: plan?.code || '',
    name: plan?.name || '',
    description: plan?.description || '',
    edition: plan?.edition === 'enterprise' || plan?.edition === 'legacy' ? plan.edition : 'personal',
    spaceType: plan?.space_type === 'organization' || plan?.space_type === 'legacy' ? plan.space_type : 'personal',
    status: plan?.status === 'disabled' ? 'disabled' : 'active',
    isPublic: plan?.is_public ?? true,
    storageGB: (Number(plan?.included_storage_bytes) || 0) / (1024 ** 3),
    periodPoints: (Number(plan?.included_point_micros) || 0) / 1_000_000,
    multiplier: (Number(plan?.billing_multiplier_ppm) || 1_000_000) / 1_000_000,
  })
  planDialogVisible.value = true
}

function openPriceDialog(plan: BillingPlanItem) {
  const versions = pricesByPlanID.value.get(plan.id) || []
  const nextVersion = versions.length + 1
  const interval = plan.edition === 'personal' && plan.code === 'personal_free' ? 'none' : 'month'
  Object.assign(priceForm, {
    planID: plan.id,
    code: `${plan.code}_cny_v${nextVersion}`,
    billingInterval: interval,
    amountCNY: 0,
    status: 'draft',
    isDefault: versions.length === 0,
  })
  priceDialogVisible.value = true
}

async function savePlan() {
  if (!planForm.name.trim() || !planForm.code.trim()) {
    MessagePlugin.warning('请填写套餐名称和套餐代码')
    return
  }
  if (planForm.storageGB < 0 || planForm.periodPoints < 0 || planForm.multiplier <= 0) {
    MessagePlugin.warning('存储、积分和倍率必须是有效的非负数')
    return
  }
  const payload: BillingPlanInput = {
    code: planForm.code.trim(),
    name: planForm.name.trim(),
    description: planForm.description.trim(),
    edition: planForm.edition,
    space_type: planForm.spaceType,
    status: planForm.status,
    is_public: planForm.isPublic,
    included_storage_bytes: Math.round(Number(planForm.storageGB) * 1024 ** 3),
    included_point_micros: Math.round(Number(planForm.periodPoints) * 1_000_000),
    billing_multiplier_ppm: Math.round(Number(planForm.multiplier) * 1_000_000),
  }
  savingPlan.value = true
  try {
    if (planEditingID.value) {
      await updateBillingPlan(planEditingID.value, payload)
      MessagePlugin.success('套餐已更新')
    } else {
      await createBillingPlan(payload)
      MessagePlugin.success('套餐已创建')
    }
    planDialogVisible.value = false
    await loadAll()
  } catch (error) {
    MessagePlugin.error(error instanceof Error ? error.message : '套餐保存失败')
  } finally {
    savingPlan.value = false
  }
}

async function savePriceVersion() {
  if (!priceForm.planID || !priceForm.code.trim()) {
    MessagePlugin.warning('请选择套餐并填写价格代码')
    return
  }
  if (priceForm.amountCNY < 0) {
    MessagePlugin.warning('价格不能小于 0')
    return
  }
  const payload: BillingPriceInput = {
    plan_id: priceForm.planID,
    code: priceForm.code.trim(),
    currency: 'CNY',
    billing_interval: priceForm.billingInterval,
    amount_minor: Math.round(Number(priceForm.amountCNY) * 100),
    status: priceForm.status,
    is_default: priceForm.isDefault,
  }
  savingPrice.value = true
  try {
    await createBillingPriceVersion(payload)
    priceDialogVisible.value = false
    MessagePlugin.success('价格版本已创建')
    await loadAll()
  } catch (error) {
    MessagePlugin.error(error instanceof Error ? error.message : '价格版本创建失败')
  } finally {
    savingPrice.value = false
  }
}

function openServicePriceDialog() {
  Object.assign(servicePriceForm, {
    serviceCode: 'mcp.tool_call',
    serviceName: 'MCP工具调用',
    pricingMode: 'call',
    unitName: '次',
    priceCNY: 0,
    multiplier: 1,
  })
  servicePriceDialogVisible.value = true
}

function openPurchaseItemDialog(item?: BillingPurchaseItem) {
  purchaseItemEditingID.value = item?.id || ''
  Object.assign(purchaseItemForm, {
    code: item?.code || '',
    name: item?.name || '',
    itemType: item?.item_type || 'topup',
    editionScope: item?.edition_scope || 'all',
    description: item?.description || '',
    priceCNY: (Number(item?.amount_cents) || 0) / 100,
    creditPoints: (Number(item?.credit_point_micros) || 0) / 1_000_000 || 1000,
    storageGB: (Number(item?.storage_quota_bytes) || 0) / (1024 ** 3) || 20,
    status: item?.status || 'active',
    sortOrder: Number(item?.sort_order) || 0,
  })
  purchaseItemDialogVisible.value = true
}

function openManualOrderDialog() {
  Object.assign(manualOrderForm, {
    tenantID: '',
    orderType: 'manual_contract',
    planID: '',
    billingInterval: 'year',
    periodDays: 366,
    creditPoints: 0,
    storageGB: 0,
    amountCNY: 0,
    description: '',
  })
  manualOrderDialogVisible.value = true
}

async function saveServicePrice() {
  savingServicePrice.value = true
  try {
    const nanoPrice = Math.round(Math.max(0, Number(servicePriceForm.priceCNY) || 0) * 1_000_000_000)
    await createBillingServicePriceVersion({
      service_code: 'mcp.tool_call',
      service_name: 'MCP工具调用',
      pricing_mode: 'call',
      nanousd_per_call: nanoPrice,
      nanousd_per_unit: 0,
      unit_name: '次',
      service_multiplier_ppm: Math.round(Math.max(0.000001, Number(servicePriceForm.multiplier) || 1) * 1_000_000),
      status: 'active',
    })
    servicePriceDialogVisible.value = false
    MessagePlugin.success('服务价格版本已创建')
    await loadAll()
  } catch (error) {
    MessagePlugin.error(error instanceof Error ? error.message : '服务价格保存失败')
  } finally {
    savingServicePrice.value = false
  }
}

async function savePurchaseItem() {
  if (!purchaseItemForm.name.trim() || !purchaseItemForm.code.trim()) {
    MessagePlugin.warning('请填写商品名称和商品代码')
    return
  }
  const creditPoints = Math.round(Number(purchaseItemForm.creditPoints) || 0)
  const storageBytes = Math.round((Number(purchaseItemForm.storageGB) || 0) * 1024 ** 3)
  if (purchaseItemForm.itemType === 'topup' && creditPoints <= 0) {
    MessagePlugin.warning('积分包必须填写大于 0 的积分数量')
    return
  }
  if (purchaseItemForm.itemType === 'storage_addon' && storageBytes <= 0) {
    MessagePlugin.warning('存储包必须填写大于 0 的存储容量')
    return
  }
  savingPurchaseItem.value = true
  try {
    const payload = {
      code: purchaseItemForm.code.trim(),
      name: purchaseItemForm.name.trim(),
      item_type: purchaseItemForm.itemType,
      edition_scope: purchaseItemForm.editionScope,
      description: purchaseItemForm.description.trim(),
      currency: 'CNY' as const,
      amount_cents: Math.round(Math.max(0, Number(purchaseItemForm.priceCNY) || 0) * 100),
      credit_point_micros: purchaseItemForm.itemType === 'topup' ? creditPoints * 1_000_000 : 0,
      storage_quota_bytes: purchaseItemForm.itemType === 'storage_addon' ? storageBytes : 0,
      duration_days: 0,
      status: purchaseItemForm.status,
      sort_order: Math.max(0, Math.round(Number(purchaseItemForm.sortOrder) || 0)),
    }
    if (purchaseItemEditingID.value) {
      await updateBillingPurchaseItem(purchaseItemEditingID.value, payload)
      MessagePlugin.success('商品已更新')
    } else {
      await createBillingPurchaseItem(payload)
      MessagePlugin.success('商品已创建')
    }
    purchaseItemDialogVisible.value = false
    await loadAll()
  } catch (error) {
    MessagePlugin.error(error instanceof Error ? error.message : '商品保存失败')
  } finally {
    savingPurchaseItem.value = false
  }
}

async function saveManualOrder() {
  const tenantID = Number(manualOrderForm.tenantID)
  if (!Number.isInteger(tenantID) || tenantID <= 0) {
    MessagePlugin.warning('请选择目标工作空间')
    return
  }
  if (!manualOrderForm.description.trim()) {
    MessagePlugin.warning('请填写操作说明')
    return
  }
  const payload = {
    tenant_id: tenantID,
    order_type: manualOrderForm.orderType,
    amount_cents: Math.round(Math.max(0, Number(manualOrderForm.amountCNY) || 0) * 100),
    description: manualOrderForm.description.trim(),
  }
  if (manualOrderForm.orderType === 'manual_contract') {
    if (!manualOrderForm.planID) {
      MessagePlugin.warning('请选择套餐')
      return
    }
    if (!Number.isInteger(Number(manualOrderForm.periodDays)) || Number(manualOrderForm.periodDays) <= 0) {
      MessagePlugin.warning('合同有效天数必须大于 0')
      return
    }
    Object.assign(payload, {
      plan_id: manualOrderForm.planID,
      billing_interval: manualOrderForm.billingInterval,
      period_days: Math.round(Number(manualOrderForm.periodDays)),
    })
  } else if (manualOrderForm.orderType === 'topup') {
    const credits = Math.round(Number(manualOrderForm.creditPoints) || 0)
    if (credits === 0) {
      MessagePlugin.warning('积分调整不能为 0')
      return
    }
    Object.assign(payload, { credit_point_micros: credits * 1_000_000 })
  } else {
    const storageBytes = Math.round((Number(manualOrderForm.storageGB) || 0) * 1024 ** 3)
    if (storageBytes === 0) {
      MessagePlugin.warning('存储调整不能为 0')
      return
    }
    Object.assign(payload, { storage_quota_bytes: storageBytes })
  }
  savingManualOrder.value = true
  try {
    await createBillingManualOrder(payload)
    manualOrderDialogVisible.value = false
    activeTab.value = 'payment-orders'
    MessagePlugin.success('人工操作已生效，并已生成订单')
    await loadAll()
  } catch (error) {
    MessagePlugin.error(error instanceof Error ? error.message : '人工操作失败')
  } finally {
    savingManualOrder.value = false
  }
}

async function loadAll() {
  loading.value = true
  errorMessage.value = ''
  try {
    const [planRows, priceRows, accountRows, itemRows, orderRows, ledgerRows, serviceRows, storageRows] = await Promise.all([
      listBillingPlans(),
      listBillingPrices(),
      listBillingCreditAccounts(),
      listBillingPurchaseItems(),
      listBillingPaymentOrders(),
      listBillingUsageLedgers(),
      listBillingServicePrices(),
      listBillingStorageTransactions(),
    ])
    plans.value = planRows || []
    prices.value = priceRows || []
    accounts.value = accountRows || []
    purchaseItems.value = itemRows || []
    paymentOrders.value = orderRows || []
    usageLedgers.value = ledgerRows || []
    servicePrices.value = serviceRows || []
    storageTransactions.value = storageRows || []
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '计费数据加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadAll()
})
</script>

<style scoped>
.billing-admin {
  margin: 28px 30px 36px;
  border: 1px solid var(--admin-border);
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.billing-admin__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 20px;
  border-bottom: 1px solid var(--admin-border);
}

.billing-admin__toolbar h2,
.billing-admin__toolbar p {
  margin: 0;
}

.billing-admin__toolbar h2 {
  color: var(--admin-text);
  font-size: 18px;
}

.billing-admin__toolbar p {
  margin-top: 5px;
  color: var(--admin-text-muted);
  font-size: 13px;
}

.billing-admin__alert {
  margin: 16px 20px 0;
}

.billing-admin__tabs {
  padding: 0 20px 20px;
}

.billing-admin__table-wrap {
  overflow-x: auto;
  padding-top: 12px;
}

.billing-admin__tab-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 14px;
}

.billing-admin__row-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}

.billing-admin__switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin: 2px 0 18px;
  padding: 12px 14px;
  border: 1px solid var(--admin-border);
  background: var(--admin-surface-soft);
}

.billing-admin__switch-row strong,
.billing-admin__switch-row small {
  display: block;
}

.billing-admin__switch-row small {
  margin-top: 3px;
  color: var(--admin-text-muted);
  font-size: 12px;
}

.billing-admin__offline-note {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 14px;
  padding: 14px 16px;
  border-left: 3px solid var(--td-warning-color);
  background: var(--td-warning-color-1);
  color: var(--admin-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.billing-admin__dialog-alert {
  margin-bottom: 14px;
}

.billing-admin__form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 4px 16px;
}

.billing-admin__storage-release {
  color: var(--td-success-color) !important;
}

.billing-admin__table {
  width: 100%;
  min-width: 820px;
  border-collapse: collapse;
}

.billing-admin__table--wide {
  min-width: 1080px;
}

.billing-admin__table th,
.billing-admin__table td {
  padding: 13px 12px;
  border-bottom: 1px solid var(--admin-border);
  color: var(--admin-text-secondary);
  font-size: 13px;
  text-align: left;
  vertical-align: middle;
}

.billing-admin__table th {
  color: var(--admin-text-muted);
  font-weight: 500;
  background: var(--admin-surface-soft);
}

.billing-admin__table td strong,
.billing-admin__table td span {
  display: block;
}

.billing-admin__table td span {
  margin-top: 3px;
  color: var(--admin-text-muted);
  font-size: 12px;
}

.billing-admin__points {
  color: var(--td-success-color);
}

.billing-admin__plan-prices {
  display: grid;
  gap: 6px;
}

.billing-admin__plan-price {
  display: grid;
  gap: 2px;
}

.billing-admin__plan-price span {
  display: flex !important;
  align-items: center;
  gap: 4px;
}

.billing-admin__empty {
  height: 120px;
  color: var(--admin-text-muted);
  text-align: center !important;
}

@media (max-width: 720px) {
  .billing-admin {
    margin: 16px;
  }

  .billing-admin__toolbar {
    align-items: flex-start;
  }

  .billing-admin__offline-note {
    align-items: flex-start;
    flex-direction: column;
  }

  .billing-admin__form-grid {
    grid-template-columns: 1fr;
  }

}
</style>
