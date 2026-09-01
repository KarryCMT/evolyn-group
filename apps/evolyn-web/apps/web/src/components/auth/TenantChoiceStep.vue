<script setup lang="ts">
// 注册向导第 2 步：创建团队（企业画像表单，创建者即管理员）。
// 视觉口径对齐设计稿：无选项卡片（直接建团队表单）、无步骤条，
// 表单仅企业名称/需求/行业三项，按钮为全宽「下一步」主按钮
import { reactive, useTemplateRef } from 'vue';
import type { FormInstance, FormRules } from 'element-plus';
import { RiBuildingFill } from '@remixicon/vue';

defineProps<{
  /** 提交中：按钮显示 loading 并防重复提交 */
  loading?: boolean;
}>();

const emit = defineEmits<{
  /** 校验通过后上抛企业画像，由父级开通团队并进入下一步 */
  submit: [profile: { tenantName: string; demand?: string; industry: string }];
}>();

const formRef = useTemplateRef<FormInstance>('formRef');

const form = reactive({
  tenantName: '',
  demand: '',
  industry: '',
});

// 「你的需求」单选（选填）
const demandOptions = ['低代码应用搭建', '流程自动化', '数据分析与报表', '团队协作', '其他'];

// 「所属行业」单选（必填）
const industryOptions = [
  '互联网/软件',
  '制造业',
  '零售/电商',
  '教育',
  '金融',
  '医疗健康',
  '建筑/房地产',
  '专业服务',
  '政府/事业单位',
  '其他',
];

const rules: FormRules = {
  tenantName: [
    { required: true, message: '请输入企业名称', trigger: 'blur' },
    { min: 2, max: 50, message: '长度需为 2 - 50 个字符', trigger: 'blur' },
  ],
  industry: [{ required: true, message: '请选择所属行业', trigger: 'change' }],
};

async function handleSubmit() {
  const valid = await formRef.value?.validate().then(
    () => true,
    () => false,
  );
  if (!valid) return;

  emit('submit', {
    tenantName: form.tenantName.trim(),
    demand: form.demand,
    industry: form.industry,
  });
}
</script>

<template>
  <el-form
    ref="formRef"
    class="tenant-choice-step"
    :model="form"
    :rules="rules"
    size="large"
    label-position="top"
    @submit.prevent="handleSubmit"
  >
    <el-form-item prop="tenantName" label="企业名称">
      <el-input
        v-model="form.tenantName"
        name="tenantName"
        placeholder="请输入企业/团队名称"
        maxlength="50"
        clearable
        :prefix-icon="RiBuildingFill"
      />
    </el-form-item>

    <el-form-item prop="demand" label="你的需求">
      <el-select v-model="form.demand" placeholder="请选择你的需求（选填）" clearable>
        <el-option v-for="item in demandOptions" :key="item" :label="item" :value="item" />
      </el-select>
    </el-form-item>

    <el-form-item prop="industry" label="所属行业">
      <el-select v-model="form.industry" placeholder="请选择所属行业">
        <el-option v-for="item in industryOptions" :key="item" :label="item" :value="item" />
      </el-select>
    </el-form-item>

    <!-- 设计稿口径：全宽单按钮 -->
    <el-button
      class="tenant-choice-step__submit"
      type="primary"
      size="large"
      native-type="submit"
      :loading="loading"
    >
      下一步
    </el-button>
  </el-form>
</template>

<style lang="scss" scoped>
// 主按钮全宽（设计稿口径：单按钮）
.tenant-choice-step__submit {
  width: 100%;
}
</style>
