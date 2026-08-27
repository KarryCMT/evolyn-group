<script setup lang="ts">
import { ERROR_CODES, isKnownErrorCode } from '@evolyn.do/utils';
import { RiComputerFill, RiSmartphoneFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, onMounted, shallowRef } from 'vue';
import type { MemberFieldSettingDto } from '~/api/memberField';
import previewBackdrop from '~/assets/images/tenant/member-card-preview-desktop-v1.png';
import { useMemberFields } from '~/composables/memberFields';
import { memberPreviewIdentity, memberPreviewValues } from './memberField.types';

defineOptions({ name: 'MemberCardDisplay' });

// 卡片展示与字段设置共用同一份服务端配置（store 驱动，页签切换不重复
// 拉取）：勾选直接以服务端 cardVisible 初始化与提交，即时 PATCH 保存；
// 预览保留样例成员，生产成员卡片由服务端按 cardVisible 裁剪后下发
const { fields, loading, load, updateField } = useMemberFields();

const previewMode = shallowRef<'desktop' | 'mobile'>('desktop');
const previewDevices = ['desktop', 'mobile'];

/** 勾选中的卡片字段 key（服务端 cardVisible 的本地镜像）。 */
const selectedCardFieldKeys = computed({
  get: () => fields.value.filter((field) => field.cardVisible).map((field) => field.key),
  set: (keys: string[]) => {
    // checkbox-group 整组写入：对发生变化的字段逐个即时保存
    for (const field of fields.value) {
      const next = keys.includes(field.key);
      if (field.cardVisible !== next && !field.cardLocked) {
        void submitCardChange(field, next);
      }
    }
  },
});

/** 预览渲染的字段：锁定字段（姓名为卡片固定信息）不进入明细列表。 */
const selectedFields = computed(() =>
  fields.value.filter((field) => field.cardVisible && !field.cardLocked),
);

onMounted(() => {
  void load();
});

/** 单字段卡片开关即时保存：失败回滚该行并按 errCode 提示。 */
async function submitCardChange(field: MemberFieldSettingDto, next: boolean) {
  const previous = field.cardVisible;
  field.cardVisible = next;
  try {
    await updateField(field.key, { cardVisible: next });
  } catch (err) {
    field.cardVisible = previous;
    notifyCardError(err);
    if (errorCodeOf(err) === ERROR_CODES.MEMBER_FIELD_CONFIG_CONFLICT) {
      void load(true);
    }
  }
}

function errorCodeOf(err: unknown): string | undefined {
  const code = (err as { errCode?: string } | null)?.errCode;
  return isKnownErrorCode(code) ? code : undefined;
}

function notifyCardError(err: unknown) {
  switch (errorCodeOf(err)) {
    case ERROR_CODES.MEMBER_FIELD_LOCKED:
      ElMessage.error('该字段为卡片固定信息，不可调整');
      break;
    case ERROR_CODES.MEMBER_FIELD_CONFIG_CONFLICT:
      ElMessage.error('配置已被其他管理员更新，已为您刷新');
      break;
    default:
      ElMessage.error('保存失败，请稍后重试');
  }
}
</script>

<template>
  <section class="member-card-display" aria-labelledby="member-card-display-title">
    <aside class="member-card-display__selection">
      <h2 id="member-card-display-title">选择可见字段</h2>
      <p>管理成员卡片显示字段</p>
      <el-scrollbar class="member-card-display__field-scrollbar">
        <el-checkbox-group
          v-model="selectedCardFieldKeys"
          class="member-card-display__field-list"
          :disabled="loading"
        >
          <el-checkbox
            v-for="field in fields"
            :key="field.key"
            :value="field.key"
            :disabled="field.cardLocked"
            :aria-label="`${field.label}在成员卡片显示`"
          >
            {{ field.label }}
          </el-checkbox>
        </el-checkbox-group>
      </el-scrollbar>
    </aside>

    <section class="member-card-display__preview" aria-label="成员卡片预览图">
      <header class="member-card-display__preview-header">
        <h2>成员卡片预览图</h2>
        <el-segmented
          v-model="previewMode"
          class="member-card-display__device-switch"
          :options="previewDevices"
          aria-label="预览设备"
        >
          <template #default="{ item }">
            <RiComputerFill
              v-if="item === 'desktop'"
              class="member-card-display__device-icon"
              aria-label="桌面端预览"
            />
            <RiSmartphoneFill
              v-else
              class="member-card-display__device-icon"
              aria-label="移动端预览"
            />
          </template>
        </el-segmented>
      </header>

      <div class="member-card-display__preview-stage">
        <template v-if="previewMode === 'desktop'">
          <img class="member-card-display__backdrop" :src="previewBackdrop" alt="" />
          <div class="member-card-display__desktop-shade" aria-hidden="true" />
          <article class="member-card member-card--desktop">
            <div class="member-card__identity">
              <span class="member-card__avatar">{{ memberPreviewIdentity.avatarText }}</span>
              <div>
                <h3>{{ memberPreviewIdentity.name }}</h3>
                <span>{{ memberPreviewIdentity.tag }}</span>
              </div>
            </div>
            <dl class="member-card__details">
              <template v-for="field in selectedFields" :key="field.key">
                <dt>{{ field.label }}</dt>
                <dd>{{ memberPreviewValues[field.key] }}</dd>
              </template>
            </dl>
          </article>
        </template>

        <template v-else>
          <article class="member-card member-card--mobile">
            <div class="member-card__mobile-backdrop">
              <img :src="previewBackdrop" alt="" />
              <div aria-hidden="true" />
            </div>
            <div class="member-card__mobile-content">
              <div class="member-card__identity">
                <span class="member-card__avatar">{{ memberPreviewIdentity.avatarText }}</span>
                <div>
                  <h3>{{ memberPreviewIdentity.name }}</h3>
                  <span>{{ memberPreviewIdentity.tag }}</span>
                </div>
              </div>
              <dl class="member-card__details">
                <template v-for="field in selectedFields" :key="field.key">
                  <dt>{{ field.label }}</dt>
                  <dd>{{ memberPreviewValues[field.key] }}</dd>
                </template>
              </dl>
            </div>
          </article>
        </template>
      </div>
    </section>
  </section>
</template>

<style scoped lang="scss">
.member-card-display {
  display: grid;
  box-sizing: border-box;
  height: 100%;
  min-height: 0;
  grid-template-columns: 268px minmax(0, 1fr);
  background: var(--el-bg-color);

  &__selection {
    display: flex;
    min-height: 0;
    flex-direction: column;
    padding: var(--el-space-3xl) var(--el-space-2xl) var(--el-space-2xl);

    h2,
    p {
      margin: 0;
    }

    h2 {
      color: var(--el-text-color-primary);
      font-size: var(--el-font-size-medium);
      font-weight: 700;
      line-height: 24px;
    }

    p {
      margin-top: var(--el-space-xs);
      color: var(--el-text-color-secondary);
      font-size: var(--el-font-size-base);
      line-height: 22px;
    }
  }

  &__field-scrollbar {
    min-height: 0;
    flex: 1;
    margin-top: var(--el-space-3xl);
  }

  &__field-list {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-xl);
    padding: var(--el-space-xs) 0 var(--el-space-3xl) var(--el-space-sm);
  }

  &__preview {
    display: flex;
    min-width: 0;
    min-height: 0;
    flex-direction: column;
    padding: var(--el-space-3xl) var(--el-space-6xl) var(--el-space-2xl) var(--el-space-lg);
  }

  &__preview-header {
    display: flex;
    min-height: 32px;
    align-items: flex-start;
    justify-content: space-between;

    h2 {
      margin: 0;
      color: var(--el-text-color-primary);
      font-size: var(--el-font-size-medium);
      font-weight: 700;
      line-height: 24px;
    }
  }

  &__device-switch {
    width: 120px;
  }

  &__device-icon {
    width: 18px;
    height: 18px;
  }

  &__preview-stage {
    position: relative;
    display: flex;
    min-height: 0;
    flex: 1;
    overflow: hidden;
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    justify-content: center;
    background: var(--el-fill-color-lighter);
  }

  &__backdrop,
  &__desktop-shade {
    position: absolute;
    inset: 62px 100px 90px;
    width: auto;
    height: auto;
    border-radius: var(--el-border-radius-medium);
  }

  &__backdrop {
    object-fit: cover;
    opacity: 0.56;
  }

  &__desktop-shade {
    background: rgb(248 250 252 / 18%);
  }
}

.member-card {
  box-sizing: border-box;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  box-shadow: var(--el-box-shadow-light);

  &--desktop {
    position: relative;
    z-index: 1;
    width: min(280px, calc(100% - 48px));
    padding: var(--el-space-xl) var(--el-space-2xl) var(--el-space-2xl);
    border-radius: var(--el-border-radius-medium);
    transform: translateX(-24px);
  }

  &--mobile {
    width: min(392px, calc(100% - 34px));
    overflow: hidden;
    border-radius: var(--el-border-radius-medium);
  }

  &__identity {
    display: flex;
    align-items: center;
    gap: var(--el-space-lg);

    h3,
    span {
      margin: 0;
    }

    h3 {
      color: var(--el-text-color-primary);
      font-size: var(--el-font-size-medium);
      font-weight: 700;
      line-height: 24px;
    }

    div > span {
      display: inline-flex;
      padding: var(--el-space-xs) var(--el-space-xs);
      color: var(--el-text-color-regular);
      background: var(--el-fill-color);
      font-size: var(--el-font-size-base);
      line-height: 20px;
    }
  }

  &__avatar {
    display: inline-flex;
    width: 40px;
    height: 40px;
    flex: 0 0 40px;
    border-radius: var(--el-border-radius-half);
    align-items: center;
    justify-content: center;
    color: var(--el-color-primary-light-9);
    background: var(--el-color-primary);
    font-size: var(--el-font-size-medium);
    line-height: 1;
  }

  &__details {
    display: grid;
    margin: var(--el-space-xl) 0 0;
    padding-top: var(--el-space-lg);
    border-top: 1px solid var(--el-border-color-lighter);
    grid-template-columns: 66px minmax(0, 1fr);
    row-gap: var(--el-space-md);
    font-size: var(--el-font-size-base);
    line-height: 20px;

    dt {
      color: var(--el-text-color-secondary);
    }

    dd {
      min-width: 0;
      margin: 0;
      color: var(--el-text-color-regular);
      font-weight: 500;
      overflow-wrap: anywhere;
    }
  }

  &__mobile-backdrop {
    position: relative;
    height: 344px;
    overflow: hidden;
    background: var(--el-fill-color-lighter);

    img,
    div {
      position: absolute;
      inset: 0;
      width: 100%;
      height: 100%;
    }

    img {
      object-fit: cover;
    }

    div {
      background: rgb(30 41 59 / 36%);
    }
  }

  &__mobile-content {
    padding: var(--el-space-2xl) var(--el-space-3xl) var(--el-space-3xl);
  }
}

@media (max-width: 960px) {
  .member-card-display {
    grid-template-columns: 252px minmax(0, 1fr);

    &__preview {
      padding-right: var(--el-space-2xl);
    }
  }
}

@media (max-width: 720px) {
  .member-card-display {
    display: flex;
    height: auto;
    flex-direction: column;
    overflow: visible;

    &__selection,
    &__preview {
      padding: var(--el-space-2xl) var(--el-space-xl);
    }

    &__field-scrollbar {
      max-height: 360px;
    }

    &__preview-stage {
      min-height: 560px;
    }
  }
}
</style>
