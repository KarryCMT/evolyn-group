<script setup lang="ts">
import { RiAddLine, RiDeleteBin6Line, RiUser3Line } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { reactive } from 'vue';

interface SuppressionRow {
  id: number;
  member: string;
  application: string;
}

defineOptions({ name: 'ReminderSuppressionDialog' });

const visible = defineModel<boolean>({ required: true });
// 使用稳定行标识支持名单的就地增删，接入成员选择器后可替换为真实 memberId。
const rows = reactive<SuppressionRow[]>([
  { id: 1, member: '林墨', application: '灵衍云示例应用' },
]);

function addRow() {
  rows.push({ id: Date.now(), member: '', application: '灵衍云示例应用' });
}

function removeRow(id: number) {
  const index = rows.findIndex((row) => row.id === id);
  if (index >= 0) rows.splice(index, 1);
}

function save() {
  visible.value = false;
  ElMessage.success('提醒屏蔽名单已保存');
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="enterprise-reminder-dialog"
    width="860px"
    :close-on-click-modal="false"
  >
    <template #header>
      <h2 class="enterprise-reminder-dialog__title">
        提醒屏蔽名单
      </h2>
    </template>

    <el-alert
      title="在名单中添加成员后，成员将不会接收到任何应用的提醒，可以减少成员被误扰的情况。若成员想接收部分应用提醒，请配置提醒应用。"
      type="info"
      :closable="false"
      show-icon
    />

    <button class="enterprise-reminder-dialog__add" type="button" @click="addRow">
      <RiAddLine aria-hidden="true" />添加成员
    </button>

    <div class="enterprise-reminder-dialog__head">
      <strong>成员</strong>
      <strong>接收提醒的应用</strong>
      <span />
    </div>

    <div v-for="row in rows" :key="row.id" class="enterprise-reminder-dialog__row">
      <el-input v-model="row.member" placeholder="请选择成员">
        <template #prefix>
          <RiUser3Line />
        </template>
      </el-input>
      <el-select v-model="row.application">
        <el-option label="灵衍云示例应用" value="灵衍云示例应用" />
        <el-option label="客户管理" value="客户管理" />
        <el-option label="项目协作" value="项目协作" />
      </el-select>
      <el-button
        text
        circle
        aria-label="删除成员"
        @click="removeRow(row.id)"
      >
        <RiDeleteBin6Line />
      </el-button>
    </div>

    <template #footer>
      <el-button @click="visible = false">
        取消
      </el-button>
      <el-button type="primary" @click="save">
        确定
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss">
.enterprise-reminder-dialog {
  --el-color-primary: #0bb6aa;
  border-radius: 14px;

  .el-dialog__header {
    padding: 22px 30px 18px;
    border-bottom: 1px solid #e7eaf0;
  }

  .el-dialog__body {
    min-height: 440px;
    padding: 26px 30px;
  }

  .el-dialog__footer {
    padding: 16px 30px;
    border-top: 1px solid #e7eaf0;
  }

  &__title {
    margin: 0;
    color: #182230;
    font-size: 22px;
    font-weight: 650;
  }

  &__add {
    display: flex;
    margin: 22px 0 12px;
    padding: 0;
    border: 0;
    align-items: center;
    gap: 5px;
    color: #0bb6aa;
    background: transparent;
    font-size: 15px;
    cursor: pointer;

    svg {
      width: 20px;
    }
  }

  &__head,
  &__row {
    display: grid;
    grid-template-columns: 260px 1fr 36px;
    gap: 22px;
    align-items: center;
  }

  &__head {
    margin-bottom: 10px;
    color: #344054;
    font-size: 14px;
  }

  &__row {
    margin-bottom: 12px;

    svg {
      width: 18px;
      height: 18px;
    }
  }
}
</style>
