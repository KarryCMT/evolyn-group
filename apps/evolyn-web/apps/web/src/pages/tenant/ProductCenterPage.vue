<script setup lang="ts">
import { ElMessage } from 'element-plus';
import { computed, onMounted, shallowRef } from 'vue';
import { useRouter } from 'vue-router';
import {
  EvolynMemberDepartmentRolePicker,
  type EvolynMemberDepartmentRolePickerMember,
  type EvolynMemberDepartmentRolePickerSelection,
  type EvolynMemberDepartmentRolePickerTreeNode,
} from '@evolyn.do/ui';
import { ERROR_CODES, isKnownErrorCode } from '@evolyn.do/utils';
import { getDepartmentTree, type DepartmentDto } from '~/api/department';
import { listMembers, type MemberListItemDto } from '~/api/member';
import {
  getTenantProducts,
  setTenantProductEnabled,
  updateTenantProductAccessScope,
  type TenantProductCard,
} from '~/api/tenantProduct';
import ProductCenterProductCard from './components/ProductCenterProductCard.vue';
import type { ProductMemberScope } from './components/ProductCenterProductCard.vue';

defineOptions({ name: 'ProductCenterPage' });

const router = useRouter();

// 一期仅一个内置产品：接口契约已支持多产品，页面按卡片列表首项渲染
const card = shallowRef<TenantProductCard | null>(null);
const loading = shallowRef(false);
const loaded = shallowRef(false);

// 范围选择器：打开时按需加载部门树与有效成员
const memberPickerVisible = shallowRef(false);
const departments = shallowRef<EvolynMemberDepartmentRolePickerTreeNode[]>([]);
const members = shallowRef<EvolynMemberDepartmentRolePickerMember[]>([]);

// 卡片交互契约不变：展示模型只消费接口返回的名称、版本与有效成员数
const product = computed(() => ({
  memberCount: card.value?.accessScope.eligibleMemberCount ?? 0,
  name: card.value?.name ?? '',
  versionName: card.value?.edition.planName || '—',
}));
const productEnabled = computed(() => card.value?.enabled ?? false);
const memberScope = computed<ProductMemberScope>(() => card.value?.accessScope.mode ?? 'all');
// 选择器 selections 采用字符串 ID 协议，提交时再转回稳定数字 ID
const memberScopeSelections = shallowRef<EvolynMemberDepartmentRolePickerSelection[]>([]);

async function loadProducts() {
  loading.value = true;
  try {
    const view = await getTenantProducts();
    card.value = view.items[0] ?? null;
    syncSelections();
    loaded.value = true;
  } catch {
    ElMessage.error('产品信息加载失败，请稍后重试');
  } finally {
    loading.value = false;
  }
}

/** 服务端 selections（数字 ID）→ 选择器 selections（字符串 ID 协议）。 */
function syncSelections() {
  memberScopeSelections.value = (card.value?.accessScope.selections ?? []).map((selection) => ({
    id: String(selection.id),
    label: selection.label,
    type: selection.type,
  }));
}

function errorCodeOf(err: unknown): string | undefined {
  const code = (err as { errCode?: string } | null)?.errCode;
  return isKnownErrorCode(code) ? code : undefined;
}

/** 409 冲突统一出口：提示后刷新为服务端最新配置（文档 10）。 */
function notifyConflict() {
  ElMessage.warning('配置已被其他管理员更新，已为您刷新');
  void loadProducts();
}

/** 启停开关：提交当前版本号，成功后以响应卡片整体替换本地状态。 */
async function handleUpdateEnabled(next: boolean) {
  const current = card.value;
  if (!current) return;
  try {
    card.value = await setTenantProductEnabled(current.code, {
      enabled: next,
      revision: current.revision,
    });
    syncSelections();
  } catch (err) {
    if (errorCodeOf(err) === ERROR_CODES.TENANT_PRODUCT_REVISION_CONFLICT) {
      notifyConflict();
      return;
    }
    ElMessage.error(next ? '启用失败，请稍后重试' : '停用失败，请稍后重试');
  }
}

/** 范围全量提交：携带读取到的版本号，成功后以响应卡片整体替换本地状态。 */
async function submitScope(
  mode: ProductMemberScope,
  departmentIds: number[] = [],
  memberIds: number[] = [],
) {
  const current = card.value;
  if (!current) return;
  try {
    card.value = await updateTenantProductAccessScope(current.code, {
      mode,
      departmentIds: departmentIds.length > 0 ? departmentIds : undefined,
      memberIds: memberIds.length > 0 ? memberIds : undefined,
      revision: current.revision,
    });
    syncSelections();
  } catch (err) {
    if (errorCodeOf(err) === ERROR_CODES.TENANT_PRODUCT_REVISION_CONFLICT) {
      notifyConflict();
      return;
    }
    ElMessage.error('可用范围保存失败，请稍后重试');
  }
}

/** 切回「全部成员」直接提交 mode=all，不能只改本地 radio 状态（文档 10）。 */
function handleScopeChange(next: ProductMemberScope) {
  if (next === 'all') void submitScope('all');
}

/** 打开范围选择器：按需加载部门树与有效成员（active），成员分页拉全量。 */
async function openMemberPicker() {
  memberPickerVisible.value = true;
  if (departments.value.length > 0) return;
  try {
    const [tree, memberItems] = await Promise.all([getDepartmentTree(), loadAllActiveMembers()]);
    departments.value = tree.map(toPickerNode);
    members.value = memberItems.map(toPickerMember);
  } catch {
    ElMessage.error('部门与成员数据加载失败，请稍后重试');
    memberPickerVisible.value = false;
  }
}

const MEMBER_PAGE_SIZE = 100;
const MEMBER_MAX_PAGES = 20; // 兜底上限 2000 人，超出时仍可按部门圈选

async function loadAllActiveMembers(): Promise<MemberListItemDto[]> {
  const items: MemberListItemDto[] = [];
  for (let page = 1; page <= MEMBER_MAX_PAGES; page += 1) {
    const result = await listMembers({ status: 'active', page, pageSize: MEMBER_PAGE_SIZE });
    items.push(...result.items);
    if (items.length >= result.total) break;
  }
  return items;
}

function toPickerNode(node: DepartmentDto): EvolynMemberDepartmentRolePickerTreeNode {
  return {
    id: String(node.id),
    label: node.name,
    children: node.children?.map(toPickerNode),
  };
}

function toPickerMember(member: MemberListItemDto): EvolynMemberDepartmentRolePickerMember {
  return {
    id: String(member.id),
    label: member.name,
    departmentIds: member.departments.map((department) => String(department.id)),
  };
}

/** 确认部分成员范围：只提交稳定数字 ID；空选择不提交（服务端同样拒绝空范围）。 */
async function confirmMemberScope(selections: EvolynMemberDepartmentRolePickerSelection[]) {
  const departmentIds: number[] = [];
  const memberIds: number[] = [];
  for (const selection of selections) {
    const id = Number(selection.id);
    if (!Number.isInteger(id) || id <= 0) continue;
    if (selection.type === 'department') departmentIds.push(id);
    if (selection.type === 'member') memberIds.push(id);
  }
  if (departmentIds.length === 0 && memberIds.length === 0) {
    ElMessage.warning('请至少选择一个部门或成员');
    return;
  }
  await submitScope('partial', departmentIds, memberIds);
}

/** 查看当前版本：跳转版本信息页，不再复用产品入口（文档 10）。 */
function viewEdition() {
  void router.push({ name: 'tenant-edition' });
}

/** 进入产品：仅当站内已有入口路由时跳转；产品后端仍会独立做访问判定。 */
function openProduct() {
  const entryPath = card.value?.entryPath;
  if (entryPath && router.resolve(entryPath).matched.length > 0) {
    void router.push(entryPath);
    return;
  }
  ElMessage.info('产品入口正在建设中');
}

onMounted(loadProducts);
</script>

<template>
  <section v-loading="loading" class="product-center-page" aria-label="产品中心">
    <ProductCenterProductCard
      v-if="card"
      :enabled="productEnabled"
      :product="product"
      :selections="memberScopeSelections"
      :scope="memberScope"
      @enter-product="openProduct"
      @edit-member-scope="openMemberPicker"
      @select-partial-scope="openMemberPicker"
      @update-enabled="handleUpdateEnabled"
      @update-scope="handleScopeChange"
      @view-edition="viewEdition"
    />
    <el-empty v-else-if="loaded" description="暂无可用产品" />
    <el-empty v-else-if="!loading" description="暂无法获取产品信息，请稍后重试">
      <el-button @click="loadProducts">重新加载</el-button>
    </el-empty>

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
  padding: var(--el-space-3xl) var(--el-space-3xl);
}

@media (max-width: 640px) {
  .product-center-page {
    padding: var(--el-space-xl);
  }
}
</style>
