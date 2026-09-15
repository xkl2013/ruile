<template>
  <section class="admin-editions">
    <div class="admin-editions__intro">
      <t-alert theme="info" variant="light">
        本页面向平台运营者展示系统支持的产品版本、能力边界和升级路径。个人版与企业版共用同一套账号和知识库底层数据。
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
    code: 'personal',
    name: '个人版',
    space: '个人工作区',
    boundary: '知识库默认私有，仅账号本人可见和管理。',
    admin: '不开放成员管理、团队空间、企业存储和企业审计。',
    upgrade: '可新建企业工作区；个人知识库不自动迁移或共享。',
    icon: 'education',
  },
  {
    code: 'enterprise',
    name: '企业版',
    space: '企业工作区',
    boundary: '企业知识库归属于工作区，成员按角色和授权协作。',
    admin: '支持成员、团队空间、企业存储、默认配置和审计治理。',
    upgrade: '可继续扩展成员、配额和企业级发布能力。',
    icon: 'home',
  },
]

</script>
