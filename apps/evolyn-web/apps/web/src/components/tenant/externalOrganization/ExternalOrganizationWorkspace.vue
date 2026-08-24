<script setup lang="ts">
import ExternalOrganizationEmptyState from './ExternalOrganizationEmptyState.vue';
import ExternalOrganizationInviteDialog from './ExternalOrganizationInviteDialog.vue';
import { useExternalOrganization } from './useExternalOrganization';

const {
  links,
  inviteVisible,
  activeInviteMode,
  editingLink,
  openInvite,
  addLink,
  updateLink,
  removeLink,
} = useExternalOrganization();
</script>

<template>
  <section class="external-organization-workspace" aria-label="互联组织">
    <!-- 当前连接列表为空时展示引导，已有链接仍保留入口方便继续新建连接。 -->
    <ExternalOrganizationEmptyState @create="openInvite" />
    <ExternalOrganizationInviteDialog
      v-model:visible="inviteVisible"
      v-model:mode="activeInviteMode"
      v-model:editing-link="editingLink"
      :links="links"
      @add="addLink"
      @update="updateLink"
      @remove="removeLink"
    />
  </section>
</template>

<style scoped lang="scss">
.external-organization-workspace {
  height: 100%;
  min-height: 100%;
}
</style>
