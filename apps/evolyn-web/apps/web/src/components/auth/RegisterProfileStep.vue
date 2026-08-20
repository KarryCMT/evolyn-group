<script setup lang="ts">
// 注册向导第 3 步「完善信息」：采集称呼、角色与了解渠道后「进入产品」。
// 角色与渠道是「人」的画像，随向导最终提交（POST /auth/register）落到
// 账号 onboarding；昵称同步 owner 成员的租户内称呼（后端事务内完成）。
// 注册全程不设密码：账号为免密状态，密码由用户后续在个人中心自行首设
import { reactive, useTemplateRef } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { User } from '@element-plus/icons-vue'

const props = defineProps<{
  /** 昵称默认值：取第 1 步注册手机号的脱敏形式，降低输入成本 */
  defaultNickname?: string
  /** 提交中：按钮显示 loading 并防重复提交 */
  loading?: boolean
}>()

const emit = defineEmits<{
  /** 进入产品：携带完善信息表单，由父级汇总三步数据一次性提交 */
  submit: [profile: { nickname: string; role: string; channel: string }]
}>()

const formRef = useTemplateRef<FormInstance>('formRef')

const form = reactive({
  nickname: props.defaultNickname ?? '',
  role: '',
  channel: '',
})

// 角色选项（单选）：值为运营分析用的稳定编码，展示用中文
const roleOptions = [
  { value: 'ceo', label: 'CEO/老板' },
  { value: 'manager', label: '业务总监/经理/主管' },
  { value: 'it', label: 'IT/信息化人员' },
  { value: 'member', label: '普通成员' },
  { value: 'teacher', label: '老师' },
  { value: 'student', label: '学生' },
]

// 了解渠道选项（单选）
const channelOptions = [
  { value: 'xiaohongshu', label: '小红书' },
  { value: 'zhihu', label: '知乎' },
  { value: 'referral', label: '他人推荐' },
  { value: 'ai', label: 'AI 推荐' },
  { value: 'search', label: '百度等搜索' },
  { value: 'toutiao', label: '今日头条' },
  { value: 'shortvideo', label: '短视频' },
  { value: 'wechat', label: '微信' },
  { value: 'other', label: '其他' },
]

const rules: FormRules = {
  nickname: [
    { required: true, message: '请输入你的姓名', trigger: 'blur' },
    { min: 2, max: 20, message: '长度需为 2 - 20 个字符', trigger: 'blur' },
  ],
  role: [{ required: true, message: '请选择你的角色', trigger: 'change' }],
  channel: [{ required: true, message: '请选择了解渠道', trigger: 'change' }],
}

async function handleSubmit() {
  const valid = await formRef.value?.validate().then(() => true, () => false)
  if (!valid) return

  emit('submit', {
    nickname: form.nickname.trim(),
    role: form.role,
    channel: form.channel,
  })
}
</script>

<template>
  <el-form
    ref="formRef"
    class="profile-step"
    :model="form"
    :rules="rules"
    size="large"
    label-position="top"
    @submit.prevent="handleSubmit"
  >
    <el-form-item prop="nickname" label="怎么称呼你">
      <el-input
        v-model="form.nickname"
        name="nickname"
        placeholder="你的姓名"
        maxlength="20"
        clearable
        :prefix-icon="User"
      />
    </el-form-item>

    <el-form-item prop="role" label="你的角色">
      <div class="profile-step__tags">
        <el-check-tag
          v-for="option in roleOptions"
          :key="option.value"
          :checked="form.role === option.value"
          @change="form.role = option.value"
        >
          {{ option.label }}
        </el-check-tag>
      </div>
    </el-form-item>

    <el-form-item prop="channel" label="你从哪里了解到我们">
      <el-select v-model="form.channel" placeholder="请选择了解渠道">
        <el-option
          v-for="option in channelOptions"
          :key="option.value"
          :label="option.label"
          :value="option.value"
        />
      </el-select>
    </el-form-item>

    <el-button
      class="profile-step__submit"
      type="primary"
      size="large"
      native-type="submit"
      :loading="loading"
    >
      进入产品
    </el-button>
  </el-form>
</template>

<style lang="scss" scoped>
// 角色单选标签组：自动换行
.profile-step__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  width: 100%;
}

.profile-step__submit {
  width: 100%;
  margin-top: 8px;
}
</style>
