<script setup lang="ts">
import { RiComputerFill, RiSmartphoneFill } from '@remixicon/vue';
import { computed, ref, shallowRef } from 'vue';
import previewBackdrop from '~/assets/images/tenant/member-card-preview-desktop-v1.png';
import { memberPreviewValues, memberProfileFields } from './memberField.types';

defineOptions({ name: 'MemberCardDisplay' });

const previewMode = shallowRef<'desktop' | 'mobile'>('desktop');
const selectedCardFieldKeys = ref(
  memberProfileFields.filter((field) => field.cardVisible).map((field) => field.key),
);
const previewDevices = ['desktop', 'mobile'];

const selectedFields = computed(() =>
  memberProfileFields.filter((field) => selectedCardFieldKeys.value.includes(field.key)),
);
</script>

<template>
  <section class="member-card-display" aria-labelledby="member-card-display-title">
    <aside class="member-card-display__selection">
      <h2 id="member-card-display-title">选择可见字段</h2>
      <p>管理成员卡片显示字段</p>
      <el-scrollbar class="member-card-display__field-scrollbar">
        <el-checkbox-group v-model="selectedCardFieldKeys" class="member-card-display__field-list">
          <el-checkbox
            v-for="field in memberProfileFields"
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
              <span class="member-card__avatar">帆</span>
              <div>
                <h3>帆小云</h3>
                <span>内部成员</span>
              </div>
            </div>
            <dl class="member-card__details">
              <template v-for="field in selectedFields" :key="field.key">
                <dt v-if="!field.cardLocked">{{ field.label }}</dt>
                <dd v-if="!field.cardLocked">{{ memberPreviewValues[field.key] }}</dd>
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
                <span class="member-card__avatar">帆</span>
                <div>
                  <h3>帆小云</h3>
                  <span>内部成员</span>
                </div>
              </div>
              <dl class="member-card__details">
                <template v-for="field in selectedFields" :key="field.key">
                  <dt v-if="!field.cardLocked">{{ field.label }}</dt>
                  <dd v-if="!field.cardLocked">{{ memberPreviewValues[field.key] }}</dd>
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
    padding: 24px 22px 22px;

    h2,
    p {
      margin: 0;
    }

    h2 {
      color: var(--el-text-color-primary);
      font-size: 16px;
      font-weight: 700;
      line-height: 24px;
    }

    p {
      margin-top: 4px;
      color: var(--el-text-color-secondary);
      font-size: 14px;
      line-height: 22px;
    }
  }

  &__field-scrollbar {
    min-height: 0;
    flex: 1;
    margin-top: 24px;
  }

  &__field-list {
    display: flex;
    flex-direction: column;
    gap: 18px;
    padding: 2px 0 24px 6px;
  }

  &__preview {
    display: flex;
    min-width: 0;
    min-height: 0;
    flex-direction: column;
    padding: 24px 48px 22px 12px;
  }

  &__preview-header {
    display: flex;
    min-height: 32px;
    align-items: flex-start;
    justify-content: space-between;

    h2 {
      margin: 0;
      color: var(--el-text-color-primary);
      font-size: 16px;
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
    border-radius: 8px;
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
    border-radius: 8px;
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
  box-shadow: 0 12px 24px rgb(31 41 55 / 8%);

  &--desktop {
    position: relative;
    z-index: 1;
    width: min(280px, calc(100% - 48px));
    padding: 18px 20px 20px;
    border-radius: 8px;
    transform: translateX(-24px);
  }

  &--mobile {
    width: min(392px, calc(100% - 34px));
    overflow: hidden;
    border-radius: 8px;
  }

  &__identity {
    display: flex;
    align-items: center;
    gap: 14px;

    h3,
    span {
      margin: 0;
    }

    h3 {
      color: var(--el-text-color-primary);
      font-size: 16px;
      font-weight: 700;
      line-height: 24px;
    }

    div > span {
      display: inline-flex;
      padding: 1px 5px;
      color: var(--el-text-color-regular);
      background: var(--el-fill-color);
      font-size: 14px;
      line-height: 20px;
    }
  }

  &__avatar {
    display: inline-flex;
    width: 40px;
    height: 40px;
    flex: 0 0 40px;
    border-radius: 50%;
    align-items: center;
    justify-content: center;
    color: #ffffff;
    background: #f64d52;
    font-size: 21px;
    line-height: 1;
  }

  &__details {
    display: grid;
    margin: 16px 0 0;
    padding-top: 14px;
    border-top: 1px solid var(--el-border-color-lighter);
    grid-template-columns: 66px minmax(0, 1fr);
    row-gap: 10px;
    font-size: 14px;
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
    padding: 22px 24px 24px;
  }
}

@media (max-width: 960px) {
  .member-card-display {
    grid-template-columns: 252px minmax(0, 1fr);

    &__preview {
      padding-right: 22px;
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
      padding: 22px 16px;
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
