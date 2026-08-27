<script setup lang="ts">
import { ERROR_CODES, isKnownErrorCode } from '@evolyn.do/utils';
import { ElMessage } from 'element-plus';
import { onMounted } from 'vue';
import type { MemberFieldSettingDto } from '~/api/memberField';
import { useMemberFields } from '~/composables/memberFields';

defineOptions({ name: 'MemberFieldSettings' });

// 字段配置以服务端快照为唯一来源：首屏拉取，勾选即时 PATCH 保存（无整页
// 保存按钮）。失败时以暂存的行旧值回滚控件状态并按 errCode 提示；成功后
// store 以服务端响应整页覆盖（revision 随之推进，两页签共享同一份状态）
const { fields, loading, load, updateField } = useMemberFields();

/** 单字段提交中的行 key 集合：请求期间禁用对应控件，避免重复提交。 */
const pendingKeys = new Set<string>();

type FieldSwitches = Pick<
  MemberFieldSettingDto,
  'personalVisible' | 'personalEditable' | 'cardVisible'
>;

onMounted(() => {
  void load();
});

/**
 * 勾选「可见」：关闭可见时前端先联动关闭「可编辑」（服务端兜底同样的
 * 联动规则），再发送一次 PATCH；提交的开关值取联动后的最终状态。
 * change 事件触发时 v-model 已写入新值，操作前的旧可见值即当前值取反。
 */
async function handleVisibleChange(field: MemberFieldSettingDto) {
  const previous: FieldSwitches = {
    personalVisible: !field.personalVisible,
    personalEditable: field.personalEditable,
    cardVisible: field.cardVisible,
  };
  if (!field.personalVisible) {
    field.personalEditable = false;
  }
  await submitFieldChange(field, previous, {
    personalVisible: field.personalVisible,
    personalEditable: field.personalEditable,
  });
}

/** 勾选「可编辑」：仅提交编辑开关（可见状态未变，不重复提交）。 */
async function handleEditableChange(field: MemberFieldSettingDto) {
  const previous: FieldSwitches = {
    personalVisible: field.personalVisible,
    personalEditable: !field.personalEditable,
    cardVisible: field.cardVisible,
  };
  await submitFieldChange(field, previous, { personalEditable: field.personalEditable });
}

/**
 * 单字段即时保存：previous 为用户操作前的行开关旧值（回滚锚点），changes
 * 为本次提交给服务端的变更开关；revision 由 store 统一携带。
 */
async function submitFieldChange(
  field: MemberFieldSettingDto,
  previous: FieldSwitches,
  changes: Partial<FieldSwitches>,
) {
  pendingKeys.add(field.key);
  try {
    await updateField(field.key, changes);
  } catch (err) {
    Object.assign(field, previous);
    notifyFieldError(err);
    if (errorCodeOf(err) === ERROR_CODES.MEMBER_FIELD_CONFIG_CONFLICT) {
      void load(true);
    }
  } finally {
    pendingKeys.delete(field.key);
  }
}

function errorCodeOf(err: unknown): string | undefined {
  const code = (err as { errCode?: string } | null)?.errCode;
  return isKnownErrorCode(code) ? code : undefined;
}

/** 失败提示：已知业务码给出可操作文案，未知错误给通用提示（不匹配错误文本）。 */
function notifyFieldError(err: unknown) {
  switch (errorCodeOf(err)) {
    case ERROR_CODES.MEMBER_FIELD_LOCKED:
      ElMessage.error('该字段配置不允许修改');
      break;
    case ERROR_CODES.MEMBER_FIELD_CONFIG_CONFLICT:
      ElMessage.error('配置已被其他管理员更新，已为您刷新');
      break;
    case ERROR_CODES.MEMBER_FIELD_CONFIG_INVALID:
      ElMessage.error('字段配置不合法，已还原');
      break;
    default:
      ElMessage.error('保存失败，请稍后重试');
  }
}
</script>

<template>
  <section class="member-field-settings" aria-labelledby="member-field-settings-title">
    <p id="member-field-settings-title" class="member-field-settings__description">
      管理成员的信息字段及权限
    </p>

    <section class="member-field-settings__table" aria-label="成员字段权限设置">
      <el-table
        v-loading="loading"
        :data="fields"
        row-key="key"
        height="100%"
        table-layout="fixed"
        scrollbar-always-on
        class="member-field-settings__data-table"
      >
        <el-table-column prop="label" label="成员信息" min-width="180" />
        <el-table-column prop="type" label="字段类型" min-width="250" />
        <el-table-column label="成员在「个人设置」页的权限" min-width="320">
          <template #default="{ row: field }">
            <div class="member-field-settings__permissions">
              <el-checkbox
                v-model="field.personalVisible"
                :disabled="field.visibilityLocked || pendingKeys.has(field.key)"
                :aria-label="`${field.label}在个人设置页可见`"
                @change="handleVisibleChange(field as MemberFieldSettingDto)"
              >
                可见
              </el-checkbox>
              <el-checkbox
                v-model="field.personalEditable"
                :disabled="
                  field.editableLocked || !field.personalVisible || pendingKeys.has(field.key)
                "
                :aria-label="`${field.label}在个人设置页可编辑`"
                @change="handleEditableChange(field as MemberFieldSettingDto)"
              >
                可编辑
              </el-checkbox>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </section>
</template>

<style scoped lang="scss">
.member-field-settings {
  display: flex;
  box-sizing: border-box;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  padding: var(--el-space-2xl) var(--el-space-2xl) var(--el-space-2xl);

  &__description {
    flex: 0 0 auto;
    margin: 0 0 var(--el-space-md);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
    line-height: 24px;
  }

  &__table {
    min-height: 0;
    flex: 1;
    overflow: hidden;
  }

  &__data-table {
    height: 100%;
    --el-table-border-color: var(--el-border-color-lighter);
    --el-table-header-bg-color: var(--el-fill-color-light);
    --el-table-header-text-color: var(--el-text-color-primary);
    --el-table-row-hover-bg-color: var(--el-fill-color-light);
    --el-table-text-color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);

    :deep(.el-table__header-wrapper .el-table__cell) {
      height: 56px;
      color: var(--el-text-color-primary);
      font-weight: 700;
    }

    :deep(.el-table__body-wrapper .el-table__cell) {
      height: 56px;
    }

    :deep(.el-table__cell) {
      padding: 0 var(--el-space-xl);
    }
  }

  &__permissions {
    display: flex;
    align-items: center;
    gap: var(--el-space-xl);
  }
}

@media (max-width: 840px) {
  .member-field-settings {
    padding: var(--el-space-xl) var(--el-space-xl) var(--el-space-xl);

    &__description {
      margin-bottom: var(--el-space-xl);
    }
  }
}
</style>
