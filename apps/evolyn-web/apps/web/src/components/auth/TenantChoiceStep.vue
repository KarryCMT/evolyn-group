<script setup lang="ts">
// 注册向导第 2 步：选择团队去向——自助创建租户（企业画像表单，创建者即
// 管理员）或加入已有租户（邀请码，后端能力未上线前为占位）；也可跳过留在默认团队
import { computed, reactive, useTemplateRef } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { OfficeBuilding, Promotion } from '@element-plus/icons-vue'

defineProps<{
  /** 提交中：按钮显示 loading 并防重复提交 */
  loading?: boolean
}>()

const emit = defineEmits<{
  /** 完成注册：mode=create 携带企业画像；mode=join 携带邀请码（父级占位提示） */
  submit: [choice: { mode: 'create' | 'join'; tenantName?: string; demand?: string; industry?: string; managementNeeds?: string[] }]
  /** 跳过：不创建/加入团队，留在默认租户 */
  skip: []
  /** 返回上一步修改账号信息 */
  back: []
}>()

type ChoiceMode = 'create' | 'join'

const formRef = useTemplateRef<FormInstance>('formRef')

const form = reactive({
  mode: 'create' as ChoiceMode,
  tenantName: '',
  demand: '',
  industry: '',
  managementNeeds: [] as string[],
  inviteCode: '',
})

// 选项卡片：创建（默认）与加入两态
const options = [
  { value: 'create' as ChoiceMode, label: '创建新团队', desc: '你将成为团队管理员', icon: OfficeBuilding },
  { value: 'join' as ChoiceMode, label: '加入已有团队', desc: '输入邀请码加入（即将上线）', icon: Promotion },
]

// 「你的需求」单选（选填）
const demandOptions = ['低代码应用搭建', '流程自动化', '数据分析与报表', '团队协作', '其他']

// 「所属行业」单选（必填）
const industryOptions = [
  '互联网/软件', '制造业', '零售/电商', '教育', '金融',
  '医疗健康', '建筑/房地产', '专业服务', '政府/事业单位', '其他',
]

// 「企业内部管理需求」多选（必填至少 1 项），口径对齐设计稿
const managementNeedOptions = ['IT项目', '任务', 'CRM/销售', 'OA', '进销存', '人事', '合同', '售后']

// 主按钮文案：创建模式走完画像表单进入完成页（截图口径「下一步」），
// 加入模式为占位提交
const submitLabel = computed(() => (form.mode === 'create' ? '下一步' : '完成注册'))

// 校验规则随所选模式切换：仅校验当前展示的输入项
const rules = computed<FormRules>(() => ({
  tenantName:
    form.mode === 'create'
      ? [
          { required: true, message: '请输入企业名称', trigger: 'blur' },
          { min: 2, max: 50, message: '长度需为 2 - 50 个字符', trigger: 'blur' },
        ]
      : [],
  industry:
    form.mode === 'create'
      ? [{ required: true, message: '请选择所属行业', trigger: 'change' }]
      : [],
  managementNeeds:
    form.mode === 'create'
      ? [
          {
            type: 'array',
            required: true,
            message: '请至少选择一项管理需求',
            trigger: 'change',
          },
        ]
      : [],
  inviteCode:
    form.mode === 'join'
      ? [{ required: true, message: '请输入邀请码', trigger: 'blur' }]
      : [],
}))

/** 管理需求多选开关（el-check-tag 无 v-model，手动维护数组） */
function toggleManagementNeed(need: string) {
  const index = form.managementNeeds.indexOf(need)
  if (index >= 0) {
    form.managementNeeds.splice(index, 1)
  } else {
    form.managementNeeds.push(need)
  }
}

async function handleSubmit() {
  const valid = await formRef.value?.validate().then(() => true, () => false)
  if (!valid) return

  emit(
    'submit',
    form.mode === 'create'
      ? {
          mode: 'create',
          tenantName: form.tenantName.trim(),
          demand: form.demand,
          industry: form.industry,
          managementNeeds: [...form.managementNeeds],
        }
      : { mode: 'join' },
  )
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
    <el-form-item prop="mode">
      <div class="tenant-choice-step__options">
        <button
          v-for="option in options"
          :key="option.value"
          class="tenant-choice-step__option"
          :class="{ 'tenant-choice-step__option--active': form.mode === option.value }"
          type="button"
          @click="form.mode = option.value"
        >
          <el-icon class="tenant-choice-step__option-icon" :size="22">
            <component :is="option.icon" />
          </el-icon>
          <span class="tenant-choice-step__option-label">{{ option.label }}</span>
          <span class="tenant-choice-step__option-desc">{{ option.desc }}</span>
        </button>
      </div>
    </el-form-item>

    <template v-if="form.mode === 'create'">
      <el-form-item prop="tenantName" label="企业名称">
        <el-input
          v-model="form.tenantName"
          name="tenantName"
          placeholder="请输入企业/团队名称"
          maxlength="50"
          clearable
          :prefix-icon="OfficeBuilding"
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

      <el-form-item prop="managementNeeds" label="你企业内部有哪些管理需求">
        <div class="tenant-choice-step__needs">
          <el-check-tag
            v-for="need in managementNeedOptions"
            :key="need"
            :checked="form.managementNeeds.includes(need)"
            @change="toggleManagementNeed(need)"
          >
            {{ need }}
          </el-check-tag>
        </div>
      </el-form-item>
    </template>

    <el-form-item v-else prop="inviteCode" label="邀请码">
      <el-input
        v-model="form.inviteCode"
        name="inviteCode"
        placeholder="请输入团队成员分享的邀请码"
        clearable
        :prefix-icon="Promotion"
      />
    </el-form-item>

    <div class="tenant-choice-step__actions">
      <el-button size="large" :disabled="loading" @click="emit('back')">上一步</el-button>
      <el-button
        class="tenant-choice-step__submit"
        type="primary"
        size="large"
        native-type="submit"
        :loading="loading"
      >
        {{ submitLabel }}
      </el-button>
    </div>

    <button class="tenant-choice-step__skip" type="button" :disabled="loading" @click="emit('skip')">
      跳过，稍后在设置中创建团队
    </button>
  </el-form>
</template>

<style lang="scss" scoped>
.tenant-choice-step__options {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  width: 100%;
}

.tenant-choice-step__option {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
  padding: 18px 10px;
  background-color: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  cursor: pointer;
  transition: border-color 0.2s, background-color 0.2s;

  &:hover {
    border-color: var(--el-color-primary-light-5);
  }

  &--active {
    background-color: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary);
  }
}

.tenant-choice-step__option-icon {
  color: var(--el-color-primary);
}

.tenant-choice-step__option-label {
  font-size: var(--el-font-size-medium);
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.tenant-choice-step__option-desc {
  font-size: var(--el-font-size-small);
  color: var(--el-text-color-secondary);
  text-align: center;
}

// 管理需求多选标签组：自动换行
.tenant-choice-step__needs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  width: 100%;
}

.tenant-choice-step__actions {
  display: flex;
  gap: 12px;
}

.tenant-choice-step__submit {
  flex: 1;
}

.tenant-choice-step__skip {
  display: block;
  margin: 16px auto 0;
  padding: 0;
  font-size: var(--el-font-size-base);
  color: var(--el-text-color-secondary);
  background: none;
  border: none;
  cursor: pointer;

  &:hover:not(:disabled) {
    color: var(--el-color-primary);
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}
</style>
