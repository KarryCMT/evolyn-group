<script setup lang="ts">
import { ElMessage } from 'element-plus';
import { shallowRef } from 'vue';
import {
  EvolynMemberDepartmentRolePicker,
  type EvolynMemberDepartmentRolePickerMember,
  type EvolynMemberDepartmentRolePickerSelection,
  type EvolynMemberDepartmentRolePickerTreeNode,
} from '@evolyn.do/ui';
import ProductCenterProductCard from './components/ProductCenterProductCard.vue';
import type { ProductMemberScope } from './components/ProductCenterProductCard.vue';

defineOptions({ name: 'ProductCenterPage' });

// 后续接入产品中心接口时，只需以接口数据替换此展示模型，卡片的交互契约无需改变。
const product = {
  memberCount: 1,
  name: '简道云',
  versionName: '试用版',
};

const productEnabled = shallowRef(true);
const memberScope = shallowRef<ProductMemberScope>('all');
const memberPickerVisible = shallowRef(false);
const memberScopeSelections = shallowRef<EvolynMemberDepartmentRolePickerSelection[]>([]);

// 产品可用范围最终会从租户成员接口获取；这里沿用通用选择器的部门与成员数据协议。
const departments: EvolynMemberDepartmentRolePickerTreeNode[] = [
  {
    id: 'tenant-root',
    label: '重庆万柯互联网科技有限责任公司',
    children: [
      { id: 'research', label: '研发部' },
      { id: 'product', label: '产品部' },
    ],
  },
];

const members: EvolynMemberDepartmentRolePickerMember[] = [
  { id: 'li-classmate', label: '李同学', departmentIds: ['research'] },
];

function openProduct() {
  ElMessage.info('产品入口正在建设中');
}

/** 仅在用户确认至少一个部门或成员后，才将范围从“全部成员”切换为“部分成员”。 */
function confirmMemberScope(selections: EvolynMemberDepartmentRolePickerSelection[]) {
  memberScopeSelections.value = selections;
  memberScope.value = 'partial';
}
</script>

<template>
  <section class="product-center-page" aria-label="产品中心">
    <ProductCenterProductCard
      :enabled="productEnabled"
      :product="product"
      :selections="memberScopeSelections"
      :scope="memberScope"
      @enter-product="openProduct"
      @edit-member-scope="memberPickerVisible = true"
      @select-partial-scope="memberPickerVisible = true"
      @update-enabled="productEnabled = $event"
      @update-scope="memberScope = $event"
    />

    <EvolynMemberDepartmentRolePicker
      v-model="memberScopeSelections"
      v-model:open="memberPickerVisible"
      title="部门成员列表"
      :departments="departments"
      :members="members"
      :selectable-types="['department', 'member']"
      @confirm="confirmMemberScope"
    />
  </section>
</template>

<style scoped lang="scss">
.product-center-page {
  box-sizing: border-box;
  min-height: 100%;
  padding: 26px 28px;
}

@media (max-width: 640px) {
  .product-center-page {
    padding: 16px;
  }
}
</style>
