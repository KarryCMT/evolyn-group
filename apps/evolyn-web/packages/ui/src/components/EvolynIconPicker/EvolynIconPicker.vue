<script setup lang="ts">
import Cropper from 'cropperjs';
import 'cropperjs/dist/cropper.css';
import { computed, onBeforeUnmount, shallowRef, useTemplateRef, watch } from 'vue';
import { RiImageAddFill, RiUpload2Fill } from '@remixicon/vue';
import EvolynScrollbar from '../EvolynScrollbar/EvolynScrollbar.vue';
import { defaultIconColors, defaultSystemIcons } from './iconOptions';
import type {
  EvolynIconPickerEmits,
  EvolynIconPickerProps,
  EvolynIconPickerValue,
} from './EvolynIconPicker.types';

defineOptions({ name: 'EvolynIconPicker' });

const props = withDefaults(defineProps<EvolynIconPickerProps>(), {
  systemIcons: () => defaultSystemIcons,
  colors: () => defaultIconColors,
  outputSize: 200,
  maxFileSize: 20 * 1024 * 1024,
});
const emit = defineEmits<EvolynIconPickerEmits>();
const modelValue = defineModel<EvolynIconPickerValue | undefined>();

const activeTab = shallowRef<'system' | 'custom'>('system');
const activeBackground = shallowRef(props.colors[0]?.background ?? '');
const validationMessage = shallowRef('');
const cropper = shallowRef<Cropper>();
const imageUrl = shallowRef('');
const objectUrl = shallowRef('');
const imageInput = useTemplateRef<HTMLInputElement>('imageInput');
const cropImage = useTemplateRef<HTMLImageElement>('cropImage');
const selectedBackground = computed(() => `linear-gradient(135deg, ${activeBackground.value})`);

function revokePreview() {
  cropper.value?.destroy();
  cropper.value = undefined;
  if (objectUrl.value) URL.revokeObjectURL(objectUrl.value);
  objectUrl.value = '';
  imageUrl.value = '';
}

function chooseImage() {
  imageInput.value?.click();
}

function handleImageChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  if (!['image/jpeg', 'image/png'].includes(file.type.toLowerCase())) {
    validationMessage.value = '请选择 JPG、JPEG 或 PNG 格式的图片';
    return;
  }
  if (file.size > props.maxFileSize) {
    validationMessage.value = `图片大小不能超过 ${Math.floor(props.maxFileSize / 1024 / 1024)} MB`;
    return;
  }

  validationMessage.value = '';
  revokePreview();
  objectUrl.value = URL.createObjectURL(file);
  imageUrl.value = objectUrl.value;
  // 清空 input，允许用户再次选择同一个文件并重新裁剪。
  input.value = '';
  activeTab.value = 'custom';
}

function initializeCropper() {
  if (!cropImage.value) return;
  cropper.value?.destroy();
  cropper.value = new Cropper(cropImage.value, {
    aspectRatio: 1,
    autoCropArea: 1,
    background: false,
    center: true,
    guides: true,
    highlight: false,
    movable: true,
    responsive: true,
    viewMode: 1,
    zoomable: true,
  });
}

function toCroppedFile(canvas: HTMLCanvasElement) {
  return new Promise<File>((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (!blob) return reject(new Error('图标裁剪失败'));
      resolve(new File([blob], `application-icon-${Date.now()}.png`, { type: 'image/png' }));
    }, 'image/png');
  });
}

async function applyCustomIcon() {
  if (!cropper.value) return;
  try {
    const canvas = cropper.value.getCroppedCanvas({
      width: props.outputSize,
      height: props.outputSize,
      imageSmoothingEnabled: true,
      imageSmoothingQuality: 'high',
    });
    const file = await toCroppedFile(canvas);
    emit('upload', file);
  } catch {
    validationMessage.value = '图标处理失败，请更换图片后重试';
  }
}

function selectSystemIcon(name: string) {
  const value: EvolynIconPickerValue = {
    type: 'remix',
    name,
    background: activeBackground.value,
  };
  modelValue.value = value;
  emit('change', value);
}

watch(
  () => props.colors,
  (colors) => {
    if (!colors.some((option) => option.background === activeBackground.value)) {
      activeBackground.value = colors[0]?.background ?? '';
    }
  },
  { deep: 1 },
);

// 已持久化的图标值由接口回填到 v-model：系统图标恢复选中态，自定义图标直接使用文件地址预览。
watch(
  modelValue,
  (value) => {
    if (!value) return;
    if (value.type === 'remix') {
      activeTab.value = 'system';
      activeBackground.value = value.background;
      return;
    }
    revokePreview();
    imageUrl.value = value.name;
    activeTab.value = 'custom';
  },
  { immediate: true },
);

onBeforeUnmount(revokePreview);
</script>

<template>
  <section class="evolyn-icon-picker" aria-label="应用图标选择器">
    <div class="evolyn-icon-picker__tabs" role="tablist" aria-label="图标类型">
      <button
        class="evolyn-icon-picker__tab"
        :class="{ 'evolyn-icon-picker__tab--active': activeTab === 'system' }"
        type="button"
        role="tab"
        :aria-selected="activeTab === 'system'"
        @click="activeTab = 'system'"
      >
        系统图标
      </button>
      <button
        class="evolyn-icon-picker__tab"
        :class="{ 'evolyn-icon-picker__tab--active': activeTab === 'custom' }"
        type="button"
        role="tab"
        :aria-selected="activeTab === 'custom'"
        @click="activeTab = 'custom'"
      >
        自定义图标
      </button>
    </div>

    <section v-if="activeTab === 'system'" class="evolyn-icon-picker__system" role="tabpanel">
      <div class="evolyn-icon-picker__colors" aria-label="图标颜色">
        <button
          v-for="option in props.colors"
          :key="option.background"
          class="evolyn-icon-picker__color"
          :class="{ 'evolyn-icon-picker__color--active': activeBackground === option.background }"
          type="button"
          :aria-label="`选择${option.label}`"
          :aria-pressed="activeBackground === option.background"
          :style="{ backgroundImage: `linear-gradient(135deg, ${option.background})` }"
          @click="activeBackground = option.background"
        />
      </div>
      <EvolynScrollbar class="evolyn-icon-picker__icon-scrollbar" height="276px">
        <div class="evolyn-icon-picker__icon-grid" role="listbox" aria-label="系统图标">
          <button
            v-for="option in props.systemIcons"
            :key="option.name"
            class="evolyn-icon-picker__icon-option"
            :class="{
              'evolyn-icon-picker__icon-option--active':
                modelValue?.type === 'remix' &&
                modelValue.name === option.name &&
                modelValue.background === activeBackground,
            }"
            type="button"
            role="option"
            :aria-label="`选择${option.label}图标`"
            :aria-selected="modelValue?.type === 'remix' && modelValue.name === option.name"
            :style="{ backgroundImage: selectedBackground }"
            @click="selectSystemIcon(option.name)"
          >
            <component :is="option.icon" />
          </button>
        </div>
      </EvolynScrollbar>
    </section>

    <section v-else class="evolyn-icon-picker__custom" role="tabpanel">
      <input
        ref="imageInput"
        class="evolyn-icon-picker__file-input"
        type="file"
        accept="image/jpeg,image/png,.jpg,.jpeg,.png"
        @change="handleImageChange"
      />
      <template v-if="imageUrl">
        <div class="evolyn-icon-picker__cropper">
          <img ref="cropImage" :src="imageUrl" alt="待裁剪的应用图标" @load="initializeCropper" />
        </div>
        <div class="evolyn-icon-picker__custom-actions">
          <button type="button" @click="chooseImage"><RiUpload2Fill /> 更换图片</button>
          <button type="button" @click="applyCustomIcon">使用图标</button>
        </div>
      </template>
      <button v-else class="evolyn-icon-picker__upload-empty" type="button" @click="chooseImage">
        <RiImageAddFill aria-hidden="true" />
        <strong>暂无自定义图标</strong>
        <span>请选择20MB以内的 JPG、JPEG 或 PNG 图片</span>
        <em>建议尺寸 200×200 像素</em>
        <b><RiUpload2Fill aria-hidden="true" /> 选择上传新图片</b>
      </button>
      <p v-if="validationMessage" class="evolyn-icon-picker__validation" role="alert">
        {{ validationMessage }}
      </p>
    </section>
  </section>
</template>

<style lang="scss">
@use './EvolynIconPicker.scss' as *;
</style>
