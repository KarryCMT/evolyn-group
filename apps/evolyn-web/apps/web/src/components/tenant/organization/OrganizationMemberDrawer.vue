<script setup lang="ts">
import { RiCloseLargeFill, RiGroup2Fill } from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import type { OrganizationMember } from './organization.types';

const props = defineProps<{
  modelValue: boolean;
  member: OrganizationMember | null;
  roleNames: string[];
}>();
const emit = defineEmits<{
  'update:modelValue': [visible: boolean];
  save: [member: OrganizationMember];
}>();

const activeTab = shallowRef<'basic' | 'more'>('basic');
const draft = shallowRef<OrganizationMember | null>(null);

const displayName = computed(() => props.member?.name ?? '成员');
const displayInitial = computed(() => displayName.value.trim().slice(0, 1) || '成');
const statusLabel = computed(() => {
  const status = props.member?.status;
  if (status === 'disabled') return '已停用';
  if (status === 'resigned') return '已离职';
  return '已启用';
});
watch(
  () => props.member,
  (member) => {
    draft.value = member ? { ...member } : null;
  },
  { immediate: true },
);

function close() {
  emit('update:modelValue', false);
}
function save() {
  if (draft.value) emit('save', draft.value);
  close();
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="props.modelValue"
      class="organization-member-drawer"
      role="dialog"
      aria-modal="true"
      aria-label="编辑成员"
    >
      <div class="organization-member-drawer__overlay" @click="close" />
      <aside class="organization-member-drawer__panel">
        <header class="organization-member-drawer__header">
          <span class="organization-member-drawer__avatar">{{ displayInitial }}</span>
          <div>
            <h2>{{ displayName }}</h2>
            <div class="organization-member-drawer__tags">
              <el-tag>已加入</el-tag><el-tag type="success">{{ statusLabel }}</el-tag>
            </div>
          </div>
          <button
            class="organization-member-drawer__close"
            type="button"
            aria-label="关闭编辑成员"
            @click="close"
          >
            <RiCloseLargeFill />
          </button>
        </header>
        <div class="organization-member-drawer__tabs" role="tablist">
          <button
            :class="{ 'organization-member-drawer__tab--active': activeTab === 'basic' }"
            type="button"
            @click="activeTab = 'basic'"
          >
            基础字段</button
          ><button
            :class="{ 'organization-member-drawer__tab--active': activeTab === 'more' }"
            type="button"
            @click="activeTab = 'more'"
          >
            更多字段
          </button>
        </div>
        <el-scrollbar class="organization-member-drawer__content">
          <form v-if="draft" class="organization-member-drawer__form" @submit.prevent="save">
            <template v-if="activeTab === 'basic'">
              <div class="organization-member-drawer__field-grid">
                <label>姓名<span>*</span><input v-model="draft.name" /></label
                ><label>别名<input v-model="draft.alias" placeholder="请输入" /></label>
                <label
                  >编号<span>*</span><input :value="draft.employeeNo" disabled /><small
                    >不支持修改此编号</small
                  ></label
                ><label>性别<input v-model="draft.gender" placeholder="请输入" /></label>
              </div>
              <section class="organization-member-drawer__field-section">
                <label>手机</label>
                <div class="organization-member-drawer__phone">
                  <span>+86</span><input :value="draft.phone.replace('+86-', '')" disabled />
                </div>
                <small>手机已验证，你无法修改。如需修改请联系成员在个人设置页面重新绑定。</small>
              </section>
              <section class="organization-member-drawer__field-section">
                <label>邮箱</label><input v-model="draft.email" />
              </section>
              <section class="organization-member-drawer__field-section">
                <label>工号</label><input v-model="draft.employeeNo" placeholder="请输入" />
              </section>
              <section class="organization-member-drawer__field-section">
                <label>部门</label>
                <div class="organization-member-drawer__selection">
                  <RiGroup2Fill /><span>{{ draft.department }}</span>
                </div>
              </section>
              <section class="organization-member-drawer__field-section">
                <label>角色</label>
                <div class="organization-member-drawer__selection">
                  <RiGroup2Fill /><span>{{ props.roleNames.join('、') || '未分配角色' }}</span>
                </div>
              </section>
            </template>
            <template v-else
              ><section class="organization-member-drawer__field-section">
                <label>成员备注</label><textarea placeholder="请输入成员备注" />
              </section>
              <section class="organization-member-drawer__field-section">
                <label>入职日期</label><input placeholder="请选择日期" /></section
            ></template>
          </form>
        </el-scrollbar>
        <footer class="organization-member-drawer__footer">
          <el-button type="primary" @click="save">保存</el-button><el-button>更多</el-button>
        </footer>
      </aside>
    </div>
  </Teleport>
</template>

<style scoped lang="scss">
.organization-member-drawer {
  position: fixed;
  z-index: 3000;
  inset: 0;
  display: flex;
  justify-content: flex-end;
}
.organization-member-drawer__overlay {
  position: absolute;
  inset: 0;
  background: rgb(0 0 0 / 50%);
}
.organization-member-drawer__panel {
  position: relative;
  display: flex;
  width: min(620px, 94vw);
  height: 100%;
  flex-direction: column;
  background: var(--el-bg-color);
  box-shadow: var(--el-box-shadow-light);
}
.organization-member-drawer__header {
  display: flex;
  min-height: 84px;
  padding: 0 var(--el-space-4xl);
  border-bottom: 1px solid var(--el-border-color-lighter);
  align-items: center;
  gap: var(--el-space-xl);
}
.organization-member-drawer__avatar {
  display: grid;
  width: 52px;
  height: 52px;
  place-items: center;
  border-radius: var(--el-border-radius-half);
  color: #fff;
  background: #f25555;
  font-size: var(--el-font-size-medium);
}
.organization-member-drawer__header h2 {
  margin: 0 0 var(--el-space-sm);
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-large);
  line-height: 24px;
}
.organization-member-drawer__tags {
  display: flex;
  gap: var(--el-space-md);
}
.organization-member-drawer__tags :deep(.el-tag) {
  border: 0;
  color: #fff;
  background: #377ff5;
}
.organization-member-drawer__tags :deep(.el-tag--success) {
  background: var(--el-color-success);
}
.organization-member-drawer__close {
  display: inline-flex;
  width: 36px;
  height: 36px;
  margin-left: auto;
  padding: 0;
  border: 0;
  border-radius: var(--el-border-radius-medium);
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  background: transparent;
  cursor: pointer;
}
.organization-member-drawer__close:hover {
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
}
.organization-member-drawer__close svg {
  width: 23px;
  height: 23px;
}
.organization-member-drawer__tabs {
  display: grid;
  height: 54px;
  padding: 0 var(--el-space-3xl);
  grid-template-columns: 1fr 1fr;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.organization-member-drawer__tabs button {
  border: 0;
  border-bottom: 3px solid var(--el-color-transparent);
  color: var(--el-text-color-regular);
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: var(--el-font-size-medium);
}
.organization-member-drawer__tabs button:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.organization-member-drawer__tab--active {
  border-bottom-color: var(--el-color-primary) !important;
  color: var(--el-color-primary) !important;
}
.organization-member-drawer__content {
  min-height: 0;
  flex: 1;
}
.organization-member-drawer__form {
  display: grid;
  padding: var(--el-space-3xl) var(--el-space-3xl) var(--el-space-5xl);
  gap: var(--el-space-3xl);
}
.organization-member-drawer__field-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--el-space-2xl);
}
.organization-member-drawer__field-grid label,
.organization-member-drawer__field-section {
  display: grid;
  gap: var(--el-space-md);
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-medium);
  line-height: 22px;
}
.organization-member-drawer__field-grid label span {
  color: var(--el-color-danger);
}
.organization-member-drawer__field-section {
  padding-top: var(--el-space-3xl);
  border-top: 1px dashed var(--el-border-color);
}
.organization-member-drawer__form input,
.organization-member-drawer__form textarea {
  box-sizing: border-box;
  width: 100%;
  min-height: 42px;
  padding: 0 var(--el-space-lg);
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-medium);
  outline: none;
  color: var(--el-text-color-primary);
  font: inherit;
}
.organization-member-drawer__form textarea {
  min-height: 100px;
  padding-top: var(--el-space-md);
  resize: vertical;
}
.organization-member-drawer__form input:focus,
.organization-member-drawer__form textarea:focus {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
}
.organization-member-drawer__form input:disabled {
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
}
.organization-member-drawer__form small {
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-small);
}
.organization-member-drawer__phone {
  display: flex;
}
.organization-member-drawer__phone span {
  display: flex;
  height: 40px;
  padding: 0 var(--el-space-lg);
  border: 1px solid var(--el-border-color);
  border-right: 0;
  border-radius: var(--el-border-radius-medium) 0 0 var(--el-border-radius-medium);
  align-items: center;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
}
.organization-member-drawer__phone input {
  border-radius: 0 var(--el-border-radius-medium) var(--el-border-radius-medium) 0;
}
.organization-member-drawer__selection {
  display: flex;
  min-height: 78px;
  padding: var(--el-space-lg);
  border: 1px dashed var(--el-border-color);
  border-radius: var(--el-border-radius-medium);
  align-items: flex-start;
  gap: var(--el-space-md);
}
.organization-member-drawer__selection svg {
  width: 20px;
  height: 20px;
  color: var(--el-color-primary);
}
.organization-member-drawer__footer {
  display: flex;
  min-height: 64px;
  padding: 0 var(--el-space-3xl);
  border-top: 1px solid var(--el-border-color-lighter);
  align-items: center;
  gap: var(--el-space-lg);
}
</style>
