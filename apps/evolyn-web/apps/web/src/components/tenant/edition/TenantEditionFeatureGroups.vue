<script setup lang="ts">
import type { EditionFeatureGroup } from './types';

const props = defineProps<{
  groups: EditionFeatureGroup[];
}>();

const emit = defineEmits<{
  upgrade: [group: EditionFeatureGroup];
}>();
</script>

<template>
  <section class="edition-features" aria-labelledby="edition-feature-title">
    <header class="edition-features__header">
      <h2 id="edition-feature-title">版本功能</h2>
      <span class="edition-features__legend edition-features__legend--available">可用</span>
      <span class="edition-features__legend edition-features__legend--disabled">不可用</span>
    </header>

    <div class="edition-features__grid">
      <article v-for="group in props.groups" :key="group.id" class="edition-features__group">
        <h3>{{ group.title }}</h3>
        <ul>
          <li
            v-for="item in group.items"
            :key="item.id"
            :class="{ 'edition-features__item--disabled': !item.available }"
          >
            <span class="edition-features__item-icon">
              <component :is="item.icon" aria-hidden="true" />
            </span>
            <strong>{{ item.title }}</strong>
            <span v-if="item.meta" class="edition-features__item-meta">{{ item.meta }}</span>
          </li>
        </ul>
        <footer v-if="group.requiresUpgrade" class="edition-features__group-footer">
          <span>功能试用中，为确保试用到期后正常使用，请升级版本。</span>
          <button type="button" @click="emit('upgrade', group)">升级版本</button>
        </footer>
      </article>
    </div>
  </section>
</template>

<style scoped lang="scss">
.edition-features {
  margin-top: var(--el-space-5xl);

  &__header,
  &__legend,
  &__group-footer,
  &__group li {
    display: flex;
    align-items: center;
  }

  &__header {
    gap: var(--el-space-xl);

    h2 {
      display: flex;
      margin: 0 var(--el-space-xs) 0 0;
      align-items: center;
      color: #1f2937;
      font-size: var(--el-font-size-large);
      font-weight: 650;
      line-height: 28px;

      &::before {
        width: 5px;
        height: 20px;
        margin-right: var(--el-space-md);
        border-radius: var(--el-border-radius-base);
        background: var(--el-color-primary);
        content: '';
      }
    }
  }

  &__legend {
    gap: var(--el-space-xs);
    color: #596577;
    font-size: var(--el-font-size-base);
    line-height: 20px;

    &::before {
      width: 16px;
      height: 16px;
      box-sizing: border-box;
      border: 1.5px solid;
      border-radius: var(--el-border-radius-half);
      content: '';
    }

    &--available::before {
      border-color: var(--el-color-primary);
      background: var(--el-bg-color-page);
    }

    &--disabled::before {
      border-color: var(--el-text-color-secondary);
      background: var(--el-bg-color-page);
    }
  }

  &__grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--el-space-2xl);
    margin-top: var(--el-space-xl);
    align-items: start;
  }

  &__group {
    overflow: hidden;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-large);
    background: var(--el-bg-color);
    box-shadow: var(--el-box-shadow-light);

    h3 {
      margin: 0;
      padding: var(--el-space-3xl) var(--el-space-3xl) var(--el-space-lg);
      color: #596577;
      font-size: var(--el-font-size-medium);
      font-weight: 650;
      line-height: 28px;
    }

    ul {
      margin: 0;
      padding: 0 var(--el-space-3xl);
      list-style: none;
    }

    li {
      min-height: 84px;
      gap: var(--el-space-xl);
      border-bottom: 1px solid var(--el-border-color-lighter);
    }
  }

  &__item-icon {
    display: grid;
    width: 46px;
    height: 46px;
    flex: 0 0 46px;
    place-items: center;
    border-radius: var(--el-border-radius-half);
    color: #10b7e4;
    background: var(--el-bg-color-page);

    svg {
      width: 22px;
      height: 22px;
    }
  }

  &__group strong {
    min-width: 0;
    color: #202c3f;
    font-size: var(--el-font-size-medium);
    font-weight: 600;
    line-height: 26px;
  }

  &__item-meta {
    margin-left: auto;
    color: #6e7888;
    font-size: var(--el-font-size-base);
    line-height: 24px;
    text-align: right;
  }

  &__item--disabled {
    .edition-features__item-icon {
      color: #8993a1;
      background: #f0f2f5;
    }

    strong,
    .edition-features__item-meta {
      color: #737d8c;
    }
  }

  &__group-footer {
    min-height: 70px;
    padding: 0 var(--el-space-3xl);
    justify-content: space-between;
    gap: var(--el-space-xl);
    color: #5f6a7b;
    font-size: var(--el-font-size-base);
    line-height: 22px;

    button {
      flex: 0 0 auto;
      padding: var(--el-space-xs) var(--el-space-md);
      border: 0;
      border-radius: var(--el-border-radius-base);
      color: var(--el-color-primary);
      background: transparent;
      font-size: var(--el-font-size-base);
      cursor: pointer;

      &:hover {
        background: var(--el-color-primary-light-9);
      }

      &:focus-visible {
        outline: 2px solid var(--el-color-primary);
        outline-offset: 2px;
      }
    }
  }
}

@media (max-width: 980px) {
  .edition-features__grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .edition-features {
    margin-top: var(--el-space-4xl);

    &__header {
      flex-wrap: wrap;
      gap: var(--el-space-md) var(--el-space-lg);

      h2 {
        width: 100%;
      }
    }

    &__group h3 {
      padding: var(--el-space-2xl) var(--el-space-2xl) var(--el-space-md);
    }

    &__group ul {
      padding: 0 var(--el-space-2xl);
    }

    &__group li {
      min-height: 72px;
      gap: var(--el-space-lg);
    }

    &__item-icon {
      width: 40px;
      height: 40px;
      flex-basis: 40px;
    }

    &__group strong {
      font-size: var(--el-font-size-base);
    }

    &__item-meta {
      font-size: var(--el-font-size-small);
    }

    &__group-footer {
      padding: var(--el-space-md) var(--el-space-2xl);
    }
  }
}
</style>
