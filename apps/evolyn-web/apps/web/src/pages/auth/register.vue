<script setup lang="ts">
// 注册向导：三步（注册账号 → 选择团队 → 完善信息）。
// 提交编排要点：
// - 幂等重试：先尝试用表单凭据登录，账号不存在才注册并登录——第 2 步失败重试
//   不会因「账号已存在」而卡死
// - 创建团队：已是所有者则复用既有团队（重试场景），否则自助开通新租户并切换进入
// - 加入团队：后端邀请码能力未上线，暂提示占位
// - 完善信息：称呼/角色/了解渠道经 PUT /accounts/me 落账号画像，昵称同步成员称呼
import { computed, shallowRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { updateMyProfile } from '~/api/account'
import { openMyTenant, register } from '~/api/auth'
import { useAuth } from '~/composables'
import type { RegisterPayload } from '~/types'

const route = useRoute()
const router = useRouter()
const { login, loadTenants, switchTenant } = useAuth()

const step = shallowRef(0)
const accountDraft = shallowRef<RegisterPayload | null>(null)
const submitting = shallowRef(false)

// 卡片标题与副标题随步骤切换（第 3 步口径对齐设计稿）
const titles = ['注册账号', '选择团队', '欢迎使用']
const subtitles = [
  '免费创建账号，开启你的低代码之旅',
  '创建新团队（你将成为管理员）或加入已有团队',
  '完善信息，即刻开启高效协同之旅',
]
const title = computed(() => titles[step.value])
const subtitle = computed(() => subtitles[step.value])

/** 第 1 步通过：暂存账号草稿，进入团队选择 */
function handleAccountNext(payload: RegisterPayload) {
  accountDraft.value = payload
  step.value = 1
}

/** 第 2 步选择「创建团队」：注册/登录 → 开通（或复用）团队（含企业画像）→ 切换进入 */
async function handleTenantSubmit(choice: {
  mode: 'create' | 'join'
  tenantName?: string
  demand?: string
  industry?: string
  managementNeeds?: string[]
}) {
  if (choice.mode === 'join') {
    ElMessage.info('邀请加入即将上线，可先创建自己的团队')
    return
  }

  const account = accountDraft.value
  if (!account) {
    step.value = 0
    return
  }

  submitting.value = true
  try {
    await ensureAccount(account)

    // 重试安全：已拥有团队则直接复用，避免重复开通；开通成功用返回的租户切换
    const tenants = await loadTenants()
    const owned = tenants.find(t => t.isOwner)
    const target = owned
      ? { tenantId: owned.tenantId }
      : await openMyTenant({
          name: choice.tenantName!,
          demand: choice.demand,
          industry: choice.industry,
          managementNeeds: choice.managementNeeds,
        }).then(t => ({ tenantId: t.id }))

    await switchTenant(target.tenantId)
    step.value = 2
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '注册失败，请稍后重试')
  } finally {
    submitting.value = false
  }
}

/** 第 2 步跳过：不建团队，注册并登录后直接完善信息（留在默认租户） */
async function handleSkip() {
  const account = accountDraft.value
  if (!account) {
    step.value = 0
    return
  }

  submitting.value = true
  try {
    await ensureAccount(account)
    step.value = 2
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '注册失败，请稍后重试')
  } finally {
    submitting.value = false
  }
}

/** 第 3 步「进入产品」：完善信息落账号画像（昵称同步成员称呼）后进入平台 */
async function handleProfileSubmit(profile: { nickname: string; role: string; channel: string }) {
  submitting.value = true
  try {
    await updateMyProfile({
      nickname: profile.nickname,
      onboarding: { role: profile.role, channel: profile.channel },
    })
    ElMessage.success('注册完成，欢迎加入！')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    router.replace(redirect)
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '提交失败，请稍后重试')
  } finally {
    submitting.value = false
  }
}

/** 幂等保障：能登录说明账号已注册（重试场景），否则注册后再登录 */
async function ensureAccount(account: RegisterPayload) {
  try {
    await login({ phone: account.phone, password: account.password })
  } catch {
    await register(account)
    await login({ phone: account.phone, password: account.password })
  }
}
</script>

<template>
  <AuthLayout :title="title" :subtitle="subtitle">
    <el-steps class="register-page__steps" :active="step" align-center>
      <el-step title="注册账号" />
      <el-step title="选择团队" />
      <el-step title="完善信息" />
    </el-steps>

    <RegisterAccountStep v-if="step === 0" :loading="submitting" @next="handleAccountNext" />

    <TenantChoiceStep
      v-else-if="step === 1"
      :loading="submitting"
      @submit="handleTenantSubmit"
      @skip="handleSkip"
      @back="step = 0"
    />

    <RegisterProfileStep
      v-else
      :default-nickname="accountDraft?.name"
      :loading="submitting"
      @submit="handleProfileSubmit"
    />

    <template #footer>
      <span class="register-page__login-tip">
        已有账号？
        <router-link to="/auth/login">直接登录</router-link>
      </span>
    </template>
  </AuthLayout>
</template>

<style lang="scss" scoped>
.register-page__steps {
  margin-bottom: 28px;
}

.register-page__login-tip a {
  font-weight: 500;
  color: var(--el-color-primary);
}
</style>
