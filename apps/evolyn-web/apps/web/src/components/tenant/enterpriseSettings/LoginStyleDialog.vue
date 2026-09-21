<script setup lang="ts">
import {
  RiDraggable,
  RiEarthFill,
  RiImageAddLine,
  RiRefreshLine,
} from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { reactive } from 'vue';
import loginArtwork from '~/assets/images/login_bg_1.png';
import brandLogo from '~/assets/logo/logo.png';

defineOptions({ name: 'LoginStyleDialog' });

const visible = defineModel<boolean>({ required: true });

// 配置项与左侧登录页预览共享状态，确保编辑操作可以即时反馈到预览区域。
const settings = reactive({
  accountLogin: true,
  ssoLogin: false,
  autoLogin: 'unchecked',
  registration: 'show',
});

function notifyUpload() {
  ElMessage.info('图片上传交互已就绪，接入文件服务后即可保存素材');
}

function save() {
  visible.value = false;
  ElMessage.success('自定义登录样式已保存');
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="enterprise-login-style-dialog"
    fullscreen
    :show-close="true"
    :close-on-click-modal="false"
  >
    <template #header>
      <h2 class="enterprise-login-style-dialog__title">
        自定义登录样式
      </h2>
    </template>

    <div class="enterprise-login-style-dialog__workspace">
      <main class="enterprise-login-style-dialog__preview-area">
        <div class="enterprise-login-style-dialog__brand">
          <img :src="brandLogo" alt="灵衍云">
          <strong>灵衍云</strong>
        </div>

        <div class="enterprise-login-style-dialog__preview-card">
          <div class="enterprise-login-style-dialog__artwork">
            <img :src="loginArtwork" alt="登录页展示图预览">
          </div>
          <div class="enterprise-login-style-dialog__login-preview">
            <span class="enterprise-login-style-dialog__language">
              <RiEarthFill aria-hidden="true" /> 简体中文
            </span>
            <div class="enterprise-login-style-dialog__login-form">
              <h3>账号登录</h3>
              <p>没有账号？<a href="#" @click.prevent>免费注册</a></p>
              <div class="enterprise-login-style-dialog__phone-row">
                <span>+86⌄</span><span>你的手机号</span>
              </div>
              <div class="enterprise-login-style-dialog__code-row">
                <span>收到的验证码</span><span>获取验证码</span>
              </div>
              <label><input type="checkbox"> 下次自动登录</label>
              <button type="button">
                登录
              </button>
              <a href="#" @click.prevent>密码登录</a>
            </div>
          </div>
        </div>
      </main>

      <aside class="enterprise-login-style-dialog__inspector" aria-label="登录样式配置">
        <section class="enterprise-login-style-dialog__inspector-section">
          <h3>企业 logo</h3>
          <div class="enterprise-login-style-dialog__upload-row">
            <button type="button" @click="notifyUpload">
              <RiImageAddLine aria-hidden="true" />选择或拖拽上传图片
            </button>
            <button type="button" class="is-secondary" @click="notifyUpload">
              <RiRefreshLine aria-hidden="true" />重置
            </button>
          </div>
          <p>支持 jpg、gif、png 格式，不超过 5M 的图片</p>
        </section>

        <section class="enterprise-login-style-dialog__inspector-section">
          <h3>展示图 <small>（建议尺寸：340*450px）</small></h3>
          <div class="enterprise-login-style-dialog__upload-row">
            <button type="button" @click="notifyUpload">
              <RiImageAddLine aria-hidden="true" />选择或拖拽上传图片
            </button>
            <button type="button" class="is-secondary" @click="notifyUpload">
              <RiRefreshLine aria-hidden="true" />重置
            </button>
          </div>
          <p>支持 jpg、gif、png 格式，不超过 5M 的图片</p>
        </section>

        <section class="enterprise-login-style-dialog__inspector-section">
          <h3>登录方式</h3>
          <label class="enterprise-login-style-dialog__method">
            <el-checkbox v-model="settings.accountLogin">账号登录</el-checkbox>
            <RiDraggable aria-label="拖动排序" />
          </label>
          <label class="enterprise-login-style-dialog__method is-disabled">
            <el-checkbox v-model="settings.ssoLogin" disabled>单点登录</el-checkbox>
            <RiDraggable aria-label="拖动排序" />
          </label>
        </section>

        <section class="enterprise-login-style-dialog__inspector-section">
          <h3>自动登录选项</h3>
          <el-radio-group v-model="settings.autoLogin">
            <el-radio value="checked">
              勾选
            </el-radio>
            <el-radio value="unchecked">
              不勾选
            </el-radio>
            <el-radio value="hidden">
              隐藏
            </el-radio>
          </el-radio-group>
        </section>

        <section class="enterprise-login-style-dialog__inspector-section">
          <h3>注册入口</h3>
          <el-radio-group v-model="settings.registration">
            <el-radio value="show">
              显示
            </el-radio>
            <el-radio value="hidden">
              隐藏
            </el-radio>
          </el-radio-group>
        </section>
      </aside>
    </div>

    <template #footer>
      <el-button @click="visible = false">
        取消
      </el-button>
      <el-button type="primary" @click="save">
        保存
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss">
.enterprise-login-style-dialog {
  --el-color-primary: #0bb6aa;
  display: flex;
  flex-direction: column;
  background: #f6f8fb;

  .el-dialog__header {
    position: relative;
    margin: 0;
    padding: 20px 56px;
    border-bottom: 1px solid #dfe4ec;
    background: #fff;
    text-align: center;
  }

  .el-dialog__headerbtn {
    top: 14px;
    right: 24px;
    width: 44px;
    height: 44px;
  }

  .el-dialog__body {
    min-height: 0;
    padding: 0;
    flex: 1;
    overflow: hidden;
  }

  .el-dialog__footer {
    padding: 14px 24px;
    border-top: 1px solid #dfe4ec;
    background: #fff;
    text-align: center;
  }

  &__title {
    margin: 0;
    color: #172033;
    font-size: 22px;
    font-weight: 650;
  }

  &__workspace {
    display: grid;
    height: 100%;
    grid-template-columns: minmax(680px, 1fr) 360px;
  }

  &__preview-area {
    min-width: 0;
    padding: 24px 34px 40px;
    overflow: auto;
  }

  &__brand {
    display: flex;
    margin-bottom: 12px;
    align-items: center;
    gap: 7px;
    color: #1677ff;
    font-size: 15px;

    img {
      width: 26px;
      height: 26px;
      object-fit: contain;
    }
  }

  &__preview-card {
    display: grid;
    width: min(1020px, 92%);
    min-height: 620px;
    margin: 0 auto;
    grid-template-columns: 1fr 1fr;
    overflow: hidden;
    border-radius: 14px;
    background: #fff;
    box-shadow: 0 12px 34px rgba(40, 55, 80, 0.08);
  }

  &__artwork {
    background: #edf4fb;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }

  &__login-preview {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #fff;
  }

  &__language {
    position: absolute;
    top: 24px;
    right: 24px;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: #7c8696;
    font-size: 12px;

    svg {
      width: 13px;
    }
  }

  &__login-form {
    display: grid;
    width: 310px;
    gap: 12px;
    color: #344054;
    font-size: 12px;

    h3 {
      margin: 0;
      color: #172033;
      font-size: 28px;
      line-height: 36px;
    }

    p {
      margin: -6px 0 18px;
    }

    a {
      color: #0bb6aa;
      text-decoration: none;
    }

    label {
      display: flex;
      align-items: center;
      gap: 6px;
    }

    button {
      height: 40px;
      border: 0;
      border-radius: 5px;
      color: #fff;
      background: #0bb6aa;
      cursor: pointer;
    }
  }

  &__phone-row,
  &__code-row {
    display: grid;
    height: 38px;
    border: 1px solid #d8dee8;
    border-radius: 5px;
    align-items: center;
    color: #98a2b3;
  }

  &__phone-row {
    grid-template-columns: 78px 1fr;

    span {
      padding-inline: 10px;
    }

    span:first-child {
      border-right: 1px solid #e2e6ed;
      color: #4d5768;
    }
  }

  &__code-row {
    grid-template-columns: 1fr 116px;

    span {
      padding-inline: 10px;
    }

    span:last-child {
      height: 100%;
      border-left: 1px solid #e2e6ed;
      text-align: center;
      line-height: 38px;
    }
  }

  &__inspector {
    min-height: 0;
    padding: 22px 24px 40px;
    overflow-y: auto;
    border-left: 1px solid #e2e6ed;
    background: #fff;
  }

  &__inspector-section {
    padding-bottom: 22px;
    border-bottom: 1px solid #e5e9f0;

    & + & {
      padding-top: 22px;
    }

    h3 {
      margin: 0 0 12px;
      color: #202939;
      font-size: 15px;
      font-weight: 650;
    }

    small,
    p {
      color: #7d8796;
      font-size: 12px;
      font-weight: 400;
    }

    p {
      margin: 8px 0 0;
    }
  }

  &__upload-row {
    display: grid;
    grid-template-columns: 1fr 72px;
    gap: 8px;

    button {
      display: flex;
      height: 40px;
      border: 1px dashed #cfd6e2;
      border-radius: 6px;
      align-items: center;
      justify-content: center;
      gap: 6px;
      color: #7d8796;
      background: #fff;
      cursor: pointer;

      svg {
        width: 17px;
      }
    }

    .is-secondary {
      border-style: solid;
    }
  }

  &__method {
    display: flex;
    min-height: 36px;
    margin-bottom: 8px;
    padding: 0 10px;
    align-items: center;
    justify-content: space-between;
    border-radius: 4px;
    background: #f7f8fa;

    > svg {
      width: 19px;
      color: #7d8796;
    }

    &.is-disabled {
      opacity: 0.7;
    }
  }
}

@media (max-width: 980px) {
  .enterprise-login-style-dialog__workspace {
    grid-template-columns: 1fr;
    overflow-y: auto;
  }

  .enterprise-login-style-dialog__preview-card {
    min-height: 520px;
  }

  .enterprise-login-style-dialog__inspector {
    border-top: 1px solid #e2e6ed;
    border-left: 0;
  }
}
</style>
