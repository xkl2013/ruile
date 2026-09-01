<template>
  <section class="admin-editions">
    <div class="admin-editions__intro">
      <t-alert theme="info" variant="light">
        本页面向软件开发者和平台运营者，用来定义客户空间的产品版本、能力开关和升级路径。园长和园所成员不直接使用这个 admin 工程。
      </t-alert>
    </div>

    <div class="edition-grid">
      <article v-for="edition in editions" :key="edition.code" class="edition-card">
        <header>
          <t-icon :name="edition.icon" />
          <div>
            <h2>{{ edition.name }}</h2>
            <p>{{ edition.space }}</p>
          </div>
        </header>
        <dl>
          <div>
            <dt>资源边界</dt>
            <dd>{{ edition.boundary }}</dd>
          </div>
          <div>
            <dt>运营关注</dt>
            <dd>{{ edition.admin }}</dd>
          </div>
          <div>
            <dt>升级路径</dt>
            <dd>{{ edition.upgrade }}</dd>
          </div>
        </dl>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
const editions = [
  {
    code: 'principal',
    name: '园长版',
    space: '园长个人空间',
    boundary: '保留个人私有资料，面向幼儿园负责人。',
    admin: '关注园长身份、园所线索、招生场景和升级到园所版。',
    upgrade: '推荐创建园所空间并选择性迁移资产。',
    icon: 'education',
  },
  {
    code: 'school',
    name: '园所版',
    space: '园所团队空间',
    boundary: '一个幼儿园一个空间，拥有成员、角色、配额和审计。',
    admin: '关注成员、权限、知识库、智能体、渠道、审计和续费。',
    upgrade: '可加入集团组织，接收总部共享资产。',
    icon: 'home',
  },
  {
    code: 'group',
    name: '集团版',
    space: '多园所组织',
    boundary: '组织成员是一个个园所空间，不直接混用所有用户。',
    admin: '关注总部模板、跨园所共享、组织成员和多空间统计。',
    upgrade: '可扩展总部、区域、园所多层治理。',
    icon: 'organization',
  },
]

</script>
