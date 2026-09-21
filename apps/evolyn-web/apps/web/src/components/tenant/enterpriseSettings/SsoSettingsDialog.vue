<script setup lang="ts">
import { ElMessage } from 'element-plus';
import { computed, reactive, shallowRef } from 'vue';

type SsoProtocol = 'saml' | 'custom' | 'cas';

defineOptions({ name: 'SsoSettingsDialog' });

const emit = defineEmits<{
  saved: [protocol: SsoProtocol];
}>();
const visible = defineModel<boolean>({ required: true });
const protocol = shallowRef<SsoProtocol>('saml');
const form = reactive({
  samlEndpoint: '',
  samlPublicKey: '',
  samlAlgorithm: 'SHA-1',
  samlIssuer: '',
  samlLogoutEndpoint: '',
  customLoginEndpoint: '',
  customSecret: '',
  customAlgorithm: 'HS256',
  customIssuer: '',
  customLogoutEndpoint: '',
  casLoginEndpoint: '',
});

// 三种协议共用保存入口，但按照各自协议只校验截图中标记的必填字段。
const canSave = computed(() => {
  if (protocol.value === 'saml') {
    return Boolean(form.samlEndpoint && form.samlPublicKey && form.samlIssuer);
  }
  if (protocol.value === 'custom') {
    return Boolean(form.customLoginEndpoint && form.customSecret);
  }
  return Boolean(form.casLoginEndpoint);
});

function generateSecret() {
  form.customSecret = crypto.randomUUID().replaceAll('-', '');
  ElMessage.success('认证密钥已生成');
}

function save() {
  if (!canSave.value) {
    ElMessage.warning('请先填写所有必填项');
    return;
  }
  emit('saved', protocol.value);
  visible.value = false;
  ElMessage.success('单点登录配置已保存');
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="enterprise-sso-dialog"
    width="860px"
    top="3vh"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <template #header>
      <h2 class="enterprise-sso-dialog__title">
        配置单点登录
      </h2>
    </template>

    <div class="enterprise-sso-dialog__content">
      <fieldset class="enterprise-sso-dialog__protocols">
        <legend>请选择单点登录配置方式</legend>
        <el-radio-group v-model="protocol">
          <el-radio value="saml">
            SAML 2.0
          </el-radio>
          <el-radio value="custom">
            自定义接口
          </el-radio>
          <el-radio value="cas">
            CAS
          </el-radio>
        </el-radio-group>
      </fieldset>

      <div class="enterprise-sso-dialog__divider" />

      <el-form v-show="protocol === 'saml'" label-position="top" class="enterprise-sso-dialog__form">
        <p class="enterprise-sso-dialog__lead">
          请从支持 SAML 认证协议的身份认证服务商中获取以下信息进行填写
        </p>
        <el-form-item label="SAML 2.0 Endpoint (HTTP)" required>
          <el-input v-model="form.samlEndpoint" />
        </el-form-item>
        <el-form-item label="IdP 公钥" required>
          <el-input v-model="form.samlPublicKey" type="textarea" :rows="4" resize="none" />
        </el-form-item>
        <el-form-item label="SAML 加密算法">
          <el-select v-model="form.samlAlgorithm" style="width: 300px">
            <el-option label="SHA-1" value="SHA-1" />
            <el-option label="SHA-256" value="SHA-256" />
          </el-select>
        </el-form-item>
        <el-form-item label="Issuer URL" required>
          <el-input v-model="form.samlIssuer" />
        </el-form-item>
        <el-form-item label="SLO Endpoint (HTTP)">
          <el-input v-model="form.samlLogoutEndpoint" />
          <p class="enterprise-sso-dialog__hint">
            若填写此项，用户访问登出地址或点击退出按钮进行登出时，均会跳转到此页面
          </p>
        </el-form-item>
      </el-form>

      <el-form v-show="protocol === 'custom'" label-position="top" class="enterprise-sso-dialog__form">
        <el-form-item label="IdP 登录接口" required>
          <el-input v-model="form.customLoginEndpoint" />
        </el-form-item>
        <el-form-item label="认证密钥" required>
          <div class="enterprise-sso-dialog__secret-row">
            <el-input v-model="form.customSecret" show-password />
            <el-button type="primary" @click="generateSecret">
              生成密钥
            </el-button>
          </div>
        </el-form-item>
        <el-form-item label="认证加密算法">
          <el-select v-model="form.customAlgorithm" style="width: 300px">
            <el-option label="HS256" value="HS256" />
            <el-option label="HS384" value="HS384" />
            <el-option label="HS512" value="HS512" />
          </el-select>
        </el-form-item>
        <el-form-item label="Issuer URL">
          <el-input v-model="form.customIssuer" />
        </el-form-item>
        <el-form-item label="IdP 登出接口">
          <el-input v-model="form.customLogoutEndpoint" />
          <p class="enterprise-sso-dialog__hint">
            若填写此项，用户访问登出地址进行登出或点击退出按钮进行登出时，均会跳转到此页面
          </p>
        </el-form-item>
      </el-form>

      <el-form v-show="protocol === 'cas'" label-position="top" class="enterprise-sso-dialog__form">
        <el-form-item label="IdP 登录接口" required>
          <el-input v-model="form.casLoginEndpoint" />
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <div class="enterprise-sso-dialog__footer">
        <el-link type="primary" :underline="true">
          如何配置？
        </el-link>
        <div>
          <el-button @click="visible = false">
            取消
          </el-button>
          <el-button type="primary" @click="save">
            保存
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<style lang="scss">
.enterprise-sso-dialog {
  --el-color-primary: #0bb6aa;
  display: flex;
  max-height: 94vh;
  margin-bottom: 0;
  flex-direction: column;
  border-radius: 14px;

  .el-dialog__header {
    padding: 22px 30px 18px;
    border-bottom: 1px solid #e7eaf0;
  }

  .el-dialog__body {
    min-height: 560px;
    padding: 0;
    overflow: hidden auto;
  }

  .el-dialog__footer {
    padding: 16px 30px;
    border-top: 1px solid #e7eaf0;
  }

  &__title {
    margin: 0;
    color: #182230;
    font-size: 24px;
    font-weight: 650;
    line-height: 32px;
  }

  &__content {
    padding: 28px 30px 24px;
  }

  &__protocols {
    margin: 0;
    padding: 0;
    border: 0;

    legend {
      margin-bottom: 12px;
      color: #344054;
      font-size: 16px;
    }

    .el-radio {
      margin-right: 34px;
      font-size: 16px;
    }
  }

  &__divider {
    height: 1px;
    margin: 26px 0 24px;
    background: #e5e9f0;
  }

  &__form {
    .el-form-item {
      margin-bottom: 24px;
    }

    .el-form-item__label {
      margin-bottom: 8px;
      color: #344054;
      font-size: 16px;
      line-height: 22px;
    }

    .el-input__wrapper,
    .el-textarea__inner,
    .el-select__wrapper {
      min-height: 46px;
      border-radius: 8px;
    }
  }

  &__lead,
  &__hint {
    color: #596579;
    font-size: 15px;
    line-height: 24px;
  }

  &__lead {
    margin: 0 0 22px;
  }

  &__hint {
    margin: 10px 0 0;
  }

  &__secret-row {
    display: grid;
    width: 100%;
    grid-template-columns: 1fr auto;
    gap: 12px;
  }

  &__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
}
</style>
