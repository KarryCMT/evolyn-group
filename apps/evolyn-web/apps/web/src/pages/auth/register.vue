<script setup lang="ts">
// 注册向导：三步（注册账号 → 选择团队 → 完善信息）。
// 提交编排要点：
// - 注册即登录：第 1 步「手机号+验证码」通过 POST /auth/user 验证码校验后
//   直接返回会话令牌，前端 applyJwt 建立会话；手机号已注册时后端等价短信
//   登录放行（created=false），重试天然幂等，无需「先登录试探」
// - 创建团队：已是所有者则复用既有团队（重试场景），否则自助开通新租户并切换进入
// - 加入团队：后端邀请码能力未上线，暂提示占位
// - 完善信息：称呼/角色/了解渠道经 PUT /accounts/me 落账号画像，昵称同步成员称呼；
//   密码为免密注册的补设置入口（选填，首设免旧密码）
import { computed, shallowRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { changeMyPassword, updateMyProfile } from '~/api/account'
import { openMyTenant, register, sendSmsCode } from '~/api/auth'
import { useAuth } from '~/composables'

const route = useRoute()
const router = useRouter()
const { applyJwt, loadTenants, switchTenant } = useAuth()

const step = shallowRef(0)
/** 已注册登录的手机号（第 3 步昵称默认带出脱敏手机号） */
const registeredPhone = shallowRef('')
const submitting = shallowRef(false)

/** 手机号脱敏（138****1234）：第 3 步昵称默认值，与后端免密注册默认昵称同口径 */
const maskedPhone = computed(() => {
  const phone = registeredPhone.value
  return phone.length === 11 ? `${phone.slice(0, 3)}****${phone.slice(7)}` : phone
})

// 卡片标题与副标题随步骤切换（第 3 步口径对齐设计稿）
const titles = ['注册账号', '选择团队', '欢迎使用']
const subtitles = [
  '免费创建账号，开启你的低代码之旅',
  '创建新团队（你将成为管理员）或加入已有团队',
  '完善信息，即刻开启高效协同之旅',
]
const title = computed(() => titles[step.value])
const subtitle = computed(() => subtitles[step.value])

/** 第 1 步「注册」：验证码通过即注册并登录，进入团队选择 */
async function handleRegisterSubmit(payload: { phone: string; smsCode: string }) {
  submitting.value = true
  try {
    const result = await register(payload)
    applyJwt(result)
    registeredPhone.value = payload.phone
    if (!result.created) {
      ElMessage.info('该手机号已注册，已为你直接登录')
    }
    step.value = 1
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '注册失败，请稍后重试')
  } finally {
    submitting.value = false
  }
}

/** 发送注册验证码：本地联调（devEcho）时后端回显验证码，弹长时提示便于取码 */
async function handleSendCode(phone: string) {
  try {
    const result = await sendSmsCode(phone, 'register')
    if (result.code) {
      ElMessage({
        message: `【本地联调】验证码：${result.code}（5 分钟内有效）`,
        type: 'info',
        duration: 10000,
        showClose: true,
      })
    } else {
      ElMessage.success('验证码已发送，请注意查收短信')
    }
  } catch (err) {
    // 冷却中给更友好的中文提示
    if (err instanceof Error && err.message.includes('cooldown')) {
      ElMessage.warning('发送太频繁，请稍后再试')
    } else {
      ElMessage.error(err instanceof Error ? err.message : '验证码发送失败')
    }
  }
}

/** 第 2 步选择「创建团队」：开通（或复用）团队（含企业画像）→ 切换进入 */
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

  submitting.value = true
  try {
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
    ElMessage.error(err instanceof Error ? err.message : '开通团队失败，请稍后重试')
  } finally {
    submitting.value = false
  }
}

/** 第 2 步跳过：不建团队，留在默认租户直接完善信息 */
function handleSkip() {
  step.value = 2
}

/** 第 3 步「进入产品」：完善信息落账号画像（昵称同步成员称呼）；
 *  填写了密码则先首设密码（免密注册账号免旧密码），再进入平台 */
async function handleProfileSubmit(profile: {
  nickname: string
  role: string
  channel: string
  password?: string
}) {
  submitting.value = true
  try {
    if (profile.password) {
      await changeMyPassword({ newPassword: profile.password })
    }
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
</script>

<template>
  <AuthLayout :title="title" :subtitle="subtitle">
    <el-steps class="register-page__steps" :active="step" align-center>
      <el-step title="注册账号" />
      <el-step title="选择团队" />
      <el-step title="完善信息" />
    </el-steps>

    <RegisterAccountStep
      v-if="step === 0"
      :loading="submitting"
      @submit="handleRegisterSubmit"
      @send-code="handleSendCode"
    />

    <TenantChoiceStep
      v-else-if="step === 1"
      :loading="submitting"
      @submit="handleTenantSubmit"
      @skip="handleSkip"
      @back="step = 0"
    />

    <RegisterProfileStep
      v-else
      :default-nickname="maskedPhone"
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
