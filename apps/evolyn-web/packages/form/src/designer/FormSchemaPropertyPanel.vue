<template>
  <aside v-if="!item" class="form-schema-property" aria-label="字段属性面板">
    <div class="form-schema-property__empty">请在画布中选择字段</div>
  </aside>

  <aside v-else class="form-schema-property" aria-label="字段属性面板">
    <header class="form-schema-property__header">
      <span class="form-schema-property__type-tag">{{ typeLabel }}</span>
      <span class="form-schema-property__title">{{ item.label || typeLabel }}</span>
    </header>

    <EvolynScrollbar class="form-schema-property__body">
      <el-form label-position="top" size="default" @submit.prevent>
        <!-- —— 公共属性（字段字典 §2） —— -->
        <el-form-item label="字段名称">
          <el-input
            v-model="item.label"
            :maxlength="64"
            placeholder="请输入字段名称"
            :disabled="isSeparator"
          />
        </el-form-item>
        <el-form-item label="字段键（widgetName）">
          <el-input
            :model-value="item.widget.widgetName"
            placeholder="字段值与规则引用的稳定键"
            @update:model-value="$emit('rename-key', String($event ?? ''))"
          />
        </el-form-item>
        <el-form-item label="说明">
          <el-input
            v-model="item.description"
            type="textarea"
            :rows="2"
            :maxlength="500"
            show-word-limit
            placeholder="字段说明文案（选填）"
          />
        </el-form-item>
        <div class="form-schema-property__switches">
          <div class="form-schema-property__switch">
            <span>必填</span>
            <el-switch
              :model-value="!item.widget.allowBlank"
              @update:model-value="item.widget.allowBlank = !$event"
            />
          </div>
          <div class="form-schema-property__switch">
            <span>可填写</span>
            <el-switch v-model="item.widget.enable" />
          </div>
          <div class="form-schema-property__switch">
            <span>可见</span>
            <el-switch v-model="item.widget.visible" />
          </div>
          <div class="form-schema-property__switch">
            <span>隐藏标签</span>
            <el-switch v-model="item.labelHidden" />
          </div>
        </div>
        <el-form-item v-if="!isSeparator" label="栅格宽度（桌面 12 列）">
          <el-select
            :model-value="item.lineWidth"
            @update:model-value="item.lineWidth = Number($event)"
          >
            <el-option
              v-for="width in lineWidthOptions"
              :key="width"
              :label="lineWidthLabel(width)"
              :value="width"
            />
          </el-select>
        </el-form-item>

        <!-- —— 控件专属属性（字段字典 §3 逐控件） —— -->
        <template v-if="widget.type === 'text'">
          <el-form-item label="占位提示">
            <el-input v-model="widget.placeholder" :maxlength="100" placeholder="请输入" />
          </el-form-item>
          <el-form-item label="格式">
            <el-select v-model="widget.format">
              <el-option label="不限" value="" />
              <el-option label="邮箱" value="email" />
            </el-select>
          </el-form-item>
          <div class="form-schema-property__pair">
            <el-form-item label="最小长度">
              <el-input-number
                :model-value="widget.minLength ?? undefined"
                :min="0"
                :max="1000"
                placeholder="不限"
                @update:model-value="widget.minLength = $event ?? null"
              />
            </el-form-item>
            <el-form-item label="最大长度">
              <el-input-number
                :model-value="widget.maxLength ?? undefined"
                :min="1"
                :max="1000"
                placeholder="不限"
                @update:model-value="widget.maxLength = $event ?? null"
              />
            </el-form-item>
          </div>
        </template>

        <template v-else-if="widget.type === 'textarea'">
          <el-form-item label="占位提示">
            <el-input v-model="widget.placeholder" :maxlength="100" placeholder="请输入" />
          </el-form-item>
          <el-form-item label="自动增高">
            <el-switch v-model="widget.autoHeight" />
          </el-form-item>
          <div class="form-schema-property__pair">
            <el-form-item label="最小长度">
              <el-input-number
                :model-value="widget.minLength ?? undefined"
                :min="0"
                :max="2000"
                placeholder="不限"
                @update:model-value="widget.minLength = $event ?? null"
              />
            </el-form-item>
            <el-form-item label="最大长度">
              <el-input-number
                :model-value="widget.maxLength ?? undefined"
                :min="1"
                :max="2000"
                placeholder="不限"
                @update:model-value="widget.maxLength = $event ?? null"
              />
            </el-form-item>
          </div>
        </template>

        <template v-else-if="widget.type === 'number'">
          <el-form-item label="占位提示">
            <el-input v-model="widget.placeholder" :maxlength="100" placeholder="请输入" />
          </el-form-item>
          <div class="form-schema-property__pair">
            <el-form-item label="最小值">
              <el-input-number
                :model-value="widget.min ?? undefined"
                placeholder="不限"
                @update:model-value="widget.min = $event ?? null"
              />
            </el-form-item>
            <el-form-item label="最大值">
              <el-input-number
                :model-value="widget.max ?? undefined"
                placeholder="不限"
                @update:model-value="widget.max = $event ?? null"
              />
            </el-form-item>
          </div>
          <el-form-item label="小数位数">
            <el-input-number
              :model-value="widget.precision ?? undefined"
              :min="0"
              :max="8"
              placeholder="不限"
              @update:model-value="widget.precision = $event ?? null"
            />
          </el-form-item>
        </template>

        <template v-else-if="widget.type === 'datetime'">
          <el-form-item label="格式">
            <el-select v-model="widget.format">
              <el-option label="日期时间（年月日 时分秒）" value="datetime" />
              <el-option label="日期（年月日）" value="date" />
              <el-option label="月份（年月）" value="month" />
              <el-option label="时间（时分）" value="time" />
            </el-select>
          </el-form-item>
        </template>

        <template v-else-if="hasOptions">
          <el-form-item v-if="hasPlaceholder" label="占位提示">
            <el-input v-model="widget.placeholder" :maxlength="100" placeholder="请选择" />
          </el-form-item>
          <el-form-item v-if="widget.type === 'combo'" label="可搜索">
            <el-switch v-model="widget.filterable" />
          </el-form-item>
          <el-form-item
            v-if="widget.type === 'radiogroup' || widget.type === 'checkboxgroup'"
            label="布局"
          >
            <el-radio-group v-model="widget.layout">
              <el-radio-button value="vertical">纵向</el-radio-button>
              <el-radio-button value="horizontal">横向</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="选项（label 与 value 同步维护）">
            <div class="form-schema-property__options">
              <div
                v-for="(option, index) in widget.options"
                :key="index"
                class="form-schema-property__option"
              >
                <el-input
                  :model-value="option.label"
                  :maxlength="100"
                  :placeholder="`选项${index + 1}`"
                  @update:model-value="updateOption(index, String($event ?? ''))"
                />
                <el-button
                  text
                  type="danger"
                  :icon="RiDeleteBin6Fill"
                  :disabled="widget.options.length <= 1"
                  @click="removeOption(index)"
                />
              </div>
              <el-button
                class="form-schema-property__option-add"
                text
                type="primary"
                :icon="RiAddFill"
                :disabled="widget.options.length >= 200"
                @click="addOption"
              >
                添加选项
              </el-button>
            </div>
          </el-form-item>
        </template>

        <template v-else-if="widget.type === 'separator'">
          <el-form-item label="分割线样式">
            <el-radio-group v-model="widget.style">
              <el-radio-button value="solid">实线</el-radio-button>
              <el-radio-button value="dashed">虚线</el-radio-button>
            </el-radio-group>
          </el-form-item>
        </template>

        <p v-else class="form-schema-property__deferred">
          该控件的专属配置已按协议保存，运行能力随后续版本开放。
        </p>
      </el-form>
    </EvolynScrollbar>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { RiAddFill, RiDeleteBin6Fill } from '@remixicon/vue';
import { EvolynScrollbar } from '@evolyn.do/ui';
import {
  ElButton,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElOption,
  ElRadioButton,
  ElRadioGroup,
  ElSelect,
  ElSwitch,
} from 'element-plus';
import type { FormItem } from '../schema/types';
import { widgetTypeLabel } from '../schema/dictionary';

/**
 * 字段属性面板：编辑 item 公共属性与按 widget.type 分派的专属配置。
 * item 是画布数组中的响应式对象，除 widgetName（需同步页面选中键）外
 * 直接就地修改；全部取值范围与字段字典一致，保存时校验器兜底。
 */
const props = defineProps<{ item?: FormItem }>();

defineEmits<{
  (event: 'rename-key', key: string): void;
}>();

const widget = computed(() => props.item!.widget as NonNullable<props.item>['widget']);
const typeLabel = computed(() => widgetTypeLabel(widget.value.type));
const isSeparator = computed(() => widget.value.type === 'separator');
const hasOptions = computed(() =>
  ['radiogroup', 'checkboxgroup', 'combo', 'combocheck'].includes(widget.value.type),
);
const hasPlaceholder = computed(() => ['combo', 'combocheck'].includes(widget.value.type));
const lineWidthOptions = [12, 8, 6, 4, 3, 2, 1];
const lineWidthLabel = (width: number) =>
  ({ 12: '整行', 8: '2/3 行', 6: '半行', 4: '1/3 行', 3: '1/4 行', 2: '1/6 行', 1: '1/12 行' })[
    width
  ] ?? `${width} 列`;

/** 选项编辑：label 与 value 同步维护（P2 简化，协议仍允许二者不同）。 */
function updateOption(index: number, label: string): void {
  const options = (widget.value as { options: Array<{ label: string; value: string }> }).options;
  options[index] = { label, value: label };
}

function addOption(): void {
  const options = (widget.value as { options: Array<{ label: string; value: string }> }).options;
  const next = `选项${options.length + 1}`;
  options.push({ label: next, value: next });
}

function removeOption(index: number): void {
  (widget.value as { options: unknown[] }).options.splice(index, 1);
}
</script>

<style lang="scss">
.form-schema-property {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  width: 264px;
  background-color: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color);
  border-left: 1px solid var(--el-border-color);

  &__empty {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
  }

  &__header {
    display: flex;
    gap: var(--el-space-sm);
    align-items: center;
    padding: var(--el-space-lg) var(--el-space-xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__type-tag {
    padding: 0 var(--el-space-sm);
    font-size: var(--el-font-size-extra-small);
    line-height: 20px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: var(--el-border-radius-base);
  }

  &__title {
    overflow: hidden;
    font-size: var(--el-font-size-base);
    font-weight: 600;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__body {
    flex: 1;
    min-height: 0;

    .el-form-item {
      margin-bottom: var(--el-space-lg);

      .el-select,
      .el-input-number {
        width: 100%;
      }
    }
  }

  &__switches {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--el-space-md);
    margin-bottom: var(--el-space-lg);
  }

  &__switch {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 32px;
    padding: 0 var(--el-space-md);
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-lighter);
    border-radius: var(--el-border-radius-base);
  }

  &__pair {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--el-space-md);
  }

  &__options {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-sm);
    width: 100%;
  }

  &__option {
    display: flex;
    gap: var(--el-space-xs);
    align-items: center;
  }

  &__option-add {
    align-self: flex-start;
  }

  &__deferred {
    margin: 0;
    padding: var(--el-space-md);
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-lighter);
    border-radius: var(--el-border-radius-base);
  }
}
</style>
