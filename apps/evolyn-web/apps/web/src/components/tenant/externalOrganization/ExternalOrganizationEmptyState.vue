<script setup lang="ts">
import externalOrganizationBanner from '~/assets/images/tenant/external-organization-banner.png';

const emit = defineEmits<{
  create: [];
}>();
</script>

<template>
  <section class="external-organization-empty" aria-label="互联组织介绍">
    <div class="external-organization-empty__content">
      <img
        class="external-organization-empty__banner"
        :src="externalOrganizationBanner"
        alt="2分钟了解互联组织"
      />
      <ul class="external-organization-empty__description">
        <li>通过「互联组织」功能可以连接经销商、供应商、母子公司等外部合作伙伴</li>
        <li>连接后可以将你的表单/流程权限分配给对方，线上完成企业间数据填报、流程审批等</li>
        <li>
          快来连接合作伙伴，提升企业间协作效率吧
          <button type="button" class="external-organization-empty__learn">了解更多</button>
        </li>
      </ul>
      <el-button type="primary" class="external-organization-empty__create" @click="emit('create')">
        新建连接
      </el-button>
    </div>
  </section>
</template>

<style scoped lang="scss">
.external-organization-empty {
  display: grid;
  box-sizing: border-box;
  height: 100%;
  min-height: 100%;
  padding: 48px;
  place-items: center;

  &__content {
    width: min(760px, 100%);
  }

  &__banner {
    display: block;
    width: 284px;
    max-width: 100%;
    height: auto;
    margin: 0 auto;
    border-radius: 10px;
  }

  &__description {
    margin: 20px auto 18px;
    padding-left: 22px;
    color: var(--el-text-color-secondary);
    font-size: 16px;
    line-height: 28px;
  }

  &__learn {
    margin-left: 12px;
    padding: 0 2px;
    border: 0;
    border-bottom: 1px solid currentcolor;
    color: var(--el-color-primary);
    background: transparent;
    cursor: pointer;
    font: inherit;
    line-height: 24px;

    &:hover {
      background: var(--el-color-primary-light-9);
    }
  }

  &__create {
    display: block;
    min-width: 112px;
    height: 44px;
    margin: 0 auto;
    font-size: 16px;
  }
}

@media (max-width: 720px) {
  .external-organization-empty {
    padding: 28px 20px;

    &__content {
      margin-top: 0;
    }
    &__description {
      margin-top: 28px;
      font-size: 14px;
      line-height: 24px;
    }
  }
}
</style>
