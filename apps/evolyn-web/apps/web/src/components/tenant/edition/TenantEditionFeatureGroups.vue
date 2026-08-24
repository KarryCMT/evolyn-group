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
  margin-top: 42px;

  &__header,
  &__legend,
  &__group-footer,
  &__group li {
    display: flex;
    align-items: center;
  }

  &__header {
    gap: 18px;

    h2 {
      display: flex;
      margin: 0 2px 0 0;
      align-items: center;
      color: #1f2937;
      font-size: 18px;
      font-weight: 650;
      line-height: 28px;

      &::before {
        width: 5px;
        height: 20px;
        margin-right: 10px;
        border-radius: 4px;
        background: var(--el-color-primary);
        content: '';
      }
    }
  }

  &__legend {
    gap: 5px;
    color: #596577;
    font-size: 14px;
    line-height: 20px;

    &::before {
      width: 16px;
      height: 16px;
      box-sizing: border-box;
      border: 1.5px solid;
      border-radius: 50%;
      content: '';
    }

    &--available::before {
      border-color: #12b6e9;
      background: #ebfaff;
    }

    &--disabled::before {
      border-color: #7f8999;
      background: #f4f5f7;
    }
  }

  &__grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 20px;
    margin-top: 16px;
    align-items: start;
  }

  &__group {
    overflow: hidden;
    border: 1px solid #e8ebf0;
    border-radius: 14px;
    background: #fff;
    box-shadow: 0 1px 3px rgb(36 51 73 / 5%);

    h3 {
      margin: 0;
      padding: 23px 28px 13px;
      color: #596577;
      font-size: 17px;
      font-weight: 650;
      line-height: 28px;
    }

    ul {
      margin: 0;
      padding: 0 28px;
      list-style: none;
    }

    li {
      min-height: 84px;
      gap: 18px;
      border-bottom: 1px solid #edf0f4;
    }
  }

  &__item-icon {
    display: grid;
    width: 46px;
    height: 46px;
    flex: 0 0 46px;
    place-items: center;
    border-radius: 50%;
    color: #10b7e4;
    background: #eaf9fd;

    svg {
      width: 22px;
      height: 22px;
    }
  }

  &__group strong {
    min-width: 0;
    color: #202c3f;
    font-size: 17px;
    font-weight: 600;
    line-height: 26px;
  }

  &__item-meta {
    margin-left: auto;
    color: #6e7888;
    font-size: 15px;
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
    padding: 0 28px;
    justify-content: space-between;
    gap: 16px;
    color: #5f6a7b;
    font-size: 14px;
    line-height: 22px;

    button {
      flex: 0 0 auto;
      padding: 5px 8px;
      border: 0;
      border-radius: 5px;
      color: var(--el-color-primary);
      background: transparent;
      font-size: 15px;
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
    margin-top: 32px;

    &__header {
      flex-wrap: wrap;
      gap: 8px 14px;

      h2 {
        width: 100%;
      }
    }

    &__group h3 {
      padding: 20px 20px 8px;
    }

    &__group ul {
      padding: 0 20px;
    }

    &__group li {
      min-height: 72px;
      gap: 12px;
    }

    &__item-icon {
      width: 40px;
      height: 40px;
      flex-basis: 40px;
    }

    &__group strong {
      font-size: 15px;
    }

    &__item-meta {
      font-size: 13px;
    }

    &__group-footer {
      padding: 10px 20px;
    }
  }
}
</style>
