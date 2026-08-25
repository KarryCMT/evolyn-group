<script setup lang="ts">
import Cropper from 'cropperjs';
import 'cropperjs/dist/cropper.css';
import { ElMessage } from 'element-plus';
import { onBeforeUnmount, shallowRef, useTemplateRef, watch } from 'vue';
import { RiCloseFill } from '@remixicon/vue';

defineOptions({ name: 'AvatarEditorDialog' });

const visible = defineModel<boolean>({ required: true });

const props = defineProps<{
  loading?: boolean;
  sourceFile?: File | null;
  avatar?: string;
}>();

const emit = defineEmits<{
  submit: [avatar: File];
}>();

const maxFileSize = 20 * 1024 * 1024;
const cropper = shallowRef<Cropper>();
const imageUrl = shallowRef('');
const objectUrl = shallowRef('');
const imageInputRef = useTemplateRef<HTMLInputElement>('imageInput');
const cropImageRef = useTemplateRef<HTMLImageElement>('cropImage');

function clearPreview() {
  cropper.value?.destroy();
  cropper.value = undefined;
  if (objectUrl.value) URL.revokeObjectURL(objectUrl.value);
  objectUrl.value = '';
  imageUrl.value = '';
  if (imageInputRef.value) imageInputRef.value.value = '';
}

function close() {
  visible.value = false;
}

function chooseImage() {
  imageInputRef.value?.click();
}

function handleImageChange(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;

  const normalizedType = file.type.toLowerCase();
  if (!['image/jpeg', 'image/png'].includes(normalizedType)) {
    ElMessage.warning('请选择 jpg、jpeg 或 png 格式的图片');
    return;
  }
  if (file.size > maxFileSize) {
    ElMessage.warning('图片大小不能超过 20 MB');
    return;
  }

  setPreview(file);
}

function setPreview(file: File) {
  clearPreview();
  objectUrl.value = URL.createObjectURL(file);
  imageUrl.value = objectUrl.value;
}

function restoreAvatar() {
  clearPreview();
  if (props.sourceFile) {
    setPreview(props.sourceFile);
    return;
  }
  imageUrl.value = props.avatar || '';
}

function initializeCropper() {
  if (!cropImageRef.value) return;
  cropper.value?.destroy();
  cropper.value = new Cropper(cropImageRef.value, {
    aspectRatio: 1,
    viewMode: 1,
    autoCropArea: 0.84,
    background: true,
    center: true,
    guides: true,
    highlight: false,
    movable: true,
    responsive: true,
    scalable: true,
    zoomable: true,
  });
}

function canvasToFile(canvas: HTMLCanvasElement) {
  return new Promise<File>((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) {
          reject(new Error('头像裁剪失败'));
          return;
        }
        resolve(new File([blob], `avatar-${Date.now()}.jpg`, { type: 'image/jpeg' }));
      },
      'image/jpeg',
      0.92,
    );
  });
}

async function save() {
  if (!cropper.value) {
    ElMessage.warning('请先选择图片');
    return;
  }

  try {
    // 统一压缩为 512px JPEG，既保证头像清晰，也避免把 20MB 原图直接写入账号资料。
    const canvas = cropper.value.getCroppedCanvas({
      width: 512,
      height: 512,
      imageSmoothingEnabled: true,
      imageSmoothingQuality: 'high',
    });
    emit('submit', await canvasToFile(canvas));
  } catch {
    ElMessage.error('头像处理失败，请更换图片后重试');
  }
}

watch(visible, (isVisible) => {
  if (isVisible) restoreAvatar();
  else clearPreview();
});

onBeforeUnmount(clearPreview);
</script>

<template>
  <el-dialog
    v-model="visible"
    class="avatar-editor-dialog"
    width="440px"
    :show-close="false"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <template #header>
      <header class="avatar-editor-dialog__header">
        <h2>修改头像</h2>
        <button type="button" aria-label="关闭修改头像" @click="close"><RiCloseFill /></button>
      </header>
    </template>

    <section class="avatar-editor-dialog__body" aria-label="头像裁剪区域">
      <p>请选择 20 MB以内的 jpg、jpeg 或 png图片</p>
      <div class="avatar-editor-dialog__cropper">
        <img
          v-if="imageUrl"
          ref="cropImage"
          :src="imageUrl"
          alt="待裁剪的头像"
          @load="initializeCropper"
        />
        <div v-else class="avatar-editor-dialog__empty">选择图片后可拖动并裁剪头像</div>
      </div>
      <input
        ref="imageInput"
        class="avatar-editor-dialog__file-input"
        type="file"
        accept="image/jpeg,image/png,.jpg,.jpeg,.png"
        @change="handleImageChange"
      />
    </section>

    <template #footer>
      <div class="avatar-editor-dialog__footer">
        <el-button :disabled="loading" @click="chooseImage">更换图片</el-button>
        <el-button type="primary" :loading="loading" @click="save">保存头像</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style lang="scss">
.avatar-editor-dialog {
  max-width: calc(100vw - 32px);
  border-radius: 12px;

  .el-dialog__header {
    margin-right: 0;
    padding: 12px 14px 10px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .el-dialog__body {
    box-sizing: border-box;
    height: 357px;
    padding: 10px 14px 8px;
  }

  .el-dialog__footer {
    padding: 0 14px 10px;
  }

  &__header {
    display: flex;
    height: 26px;
    align-items: center;
    justify-content: space-between;

    h2 {
      margin: 0;
      color: var(--el-text-color-primary);
      font-size: 18px;
      font-weight: 600;
      line-height: 26px;
    }

    button {
      display: inline-flex;
      width: 32px;
      height: 32px;
      align-items: center;
      justify-content: center;
      padding: 0;
      border: 0;
      border-radius: 4px;
      background: transparent;
      color: var(--el-text-color-secondary);
      cursor: pointer;
      transition:
        background-color 0.2s ease,
        color 0.2s ease;

      &:hover {
        background: var(--el-fill-color-light);
        color: var(--el-text-color-primary);
      }

      svg {
        width: 22px;
        height: 22px;
      }
    }
  }

  &__body > p {
    margin: 0 0 14px;
    color: var(--el-text-color-secondary);
    font-size: 16px;
    line-height: 24px;
  }

  &__body {
    height: 100%;
  }

  &__cropper {
    position: relative;
    height: calc(100% - 38px);
    overflow: hidden;
    background-color: #b5b5b5;
    background-image:
      linear-gradient(45deg, #969696 25%, transparent 25%),
      linear-gradient(-45deg, #969696 25%, transparent 25%),
      linear-gradient(45deg, transparent 75%, #969696 75%),
      linear-gradient(-45deg, transparent 75%, #969696 75%);
    background-position:
      0 0,
      0 10px,
      10px -10px,
      -10px 0;
    background-size: 20px 20px;

    img {
      display: block;
      max-width: 100%;
    }
  }

  &__empty {
    display: flex;
    height: 100%;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }

  &__file-input {
    display: none;
  }

  &__footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;

    .el-button {
      min-width: 112px;
      height: 40px;
      margin: 0;
      font-size: 16px;
    }
  }

  .cropper-view-box,
  .cropper-face {
    border-radius: 0;
  }

  .cropper-point {
    width: 9px;
    height: 9px;
    opacity: 1;
  }
}

@media (max-width: 640px) {
  .avatar-editor-dialog {
    .el-dialog__header {
      padding: 14px 18px;
    }

    .el-dialog__body {
      padding: 16px 18px 10px;
    }

    .el-dialog__footer {
      padding: 0 18px 16px;
    }

    &__header h2 {
      font-size: 18px;
      line-height: 26px;
    }

    &__cropper {
      height: min(72vw, 360px);
    }

    &__footer .el-button {
      min-width: 0;
      flex: 1;
    }
  }
}
</style>
