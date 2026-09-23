<script setup lang="ts">
import type { FormItem } from '@evolyn.do/form/schema';
import type {
  LabelElement,
  LabelRenderData,
  LabelSchema,
  TextStyle,
  ValueSource,
} from '@evolyn.do/label';
import type { LabelTemplateDetailDto } from '~/api/label';
import { millimetersToPixels, pixelsToMillimeters, renderLabelSvg } from '@evolyn.do/label';
import { ApiError } from '@evolyn.do/utils';
import { RiAddCircleLine, RiCheckLine, RiDeleteBin6Line, RiDraggable } from '@remixicon/vue';
import { ElMessage, ElMessageBox, ElOptionGroup } from 'element-plus';
import { computed, onBeforeUnmount, ref, shallowRef, watch } from 'vue';
import {
  createLabelTemplate,
  getLabelTemplate,
  listLabelTemplates,
  previewLabelTemplate,
  publishLabelTemplate,
  renderPublishedLabel,
  saveLabelTemplateDraft,
} from '~/api/label';
import {
  labelLifecycleFeedback,
  withPreviewedLabelDraft,
  withPublishedLabelDraft,
  withSavedLabelDraft,
} from './label-lifecycle';
import { useFormWorkspaceContext } from '../workspace-context';

defineOptions({ name: 'FormQrCodeSettingsPage' });

type ContentSource = 'fixed' | 'field';
type TemplateKind =
  | 'wide-left'
  | 'wide-right'
  | 'split-left'
  | 'portrait'
  | 'plain-right'
  | 'plain-left'
  | 'compact-right'
  | 'compact-left'
  | 'table'
  | 'qr-only';

interface LabelContentRow {
  id: string;
  label: string;
  source: ContentSource;
  value: string;
  fontSize: number;
}

interface LabelTemplate {
  kind: TemplateKind;
  name: string;
  ratio: '3:2' | '2:3';
  downloadSize: string;
  maxFields: number;
  accent: boolean;
  qrPosition: 'left' | 'right' | 'center';
}

interface QrCodeSettings {
  enabled: boolean;
  template: TemplateKind;
  watermark: string;
  background: string;
  title: LabelContentRow;
  subtitle: LabelContentRow;
  fields: LabelContentRow[];
}

interface FieldOption {
  label: string;
  value: string;
}

const { detail } = useFormWorkspaceContext();
const templates: LabelTemplate[] = [
  {
    kind: 'wide-left',
    name: '蓝色标题·左文右码',
    ratio: '3:2',
    downloadSize: '150 X 100mm、60 X 40mm',
    maxFields: 4,
    accent: true,
    qrPosition: 'right',
  },
  {
    kind: 'wide-right',
    name: '蓝色标题·左文右码',
    ratio: '3:2',
    downloadSize: '150 X 100mm、60 X 40mm',
    maxFields: 4,
    accent: true,
    qrPosition: 'right',
  },
  {
    kind: 'split-left',
    name: '蓝色分栏·左码右文',
    ratio: '3:2',
    downloadSize: '150 X 100mm、60 X 40mm',
    maxFields: 4,
    accent: true,
    qrPosition: 'left',
  },
  {
    kind: 'portrait',
    name: '竖版蓝色标题',
    ratio: '2:3',
    downloadSize: '100 X 150mm、40 X 60mm',
    maxFields: 4,
    accent: true,
    qrPosition: 'center',
  },
  {
    kind: 'plain-right',
    name: '简洁左文右码',
    ratio: '3:2',
    downloadSize: '150 X 100mm、60 X 40mm',
    maxFields: 4,
    accent: false,
    qrPosition: 'right',
  },
  {
    kind: 'plain-left',
    name: '简洁左码右文',
    ratio: '3:2',
    downloadSize: '150 X 100mm、60 X 40mm',
    maxFields: 4,
    accent: false,
    qrPosition: 'left',
  },
  {
    kind: 'compact-right',
    name: '紧凑左文右码',
    ratio: '3:2',
    downloadSize: '100 X 60mm、60 X 40mm',
    maxFields: 4,
    accent: false,
    qrPosition: 'right',
  },
  {
    kind: 'compact-left',
    name: '紧凑左码右文',
    ratio: '3:2',
    downloadSize: '100 X 60mm、60 X 40mm',
    maxFields: 4,
    accent: false,
    qrPosition: 'left',
  },
  {
    kind: 'table',
    name: '表格标签',
    ratio: '3:2',
    downloadSize: '150 X 100mm、60 X 40mm',
    maxFields: 4,
    accent: false,
    qrPosition: 'right',
  },
  {
    kind: 'qr-only',
    name: '纯二维码',
    ratio: '3:2',
    downloadSize: '60 X 60mm、40 X 40mm',
    maxFields: 4,
    accent: false,
    qrPosition: 'center',
  },
];
const backgroundColors = ['#168bf2', '#ff710a', '#46bd00', '#0873c9', '#d83b14', '#8c949b'];
const systemFields: FieldOption[] = [
  { label: '实例标题', value: 'sys.title' },
  { label: '提交人', value: 'sys.submittedBy' },
  { label: '创建时间', value: 'sys.submittedAt' },
  { label: '修改时间', value: 'sys.updatedAt' },
];
const fontSizes = [12, 14, 16, 18, 19, 20, 22];

function createRow(label: string, value = '', fontSize = 14): LabelContentRow {
  return { id: globalThis.crypto.randomUUID(), label, source: 'field', value, fontSize };
}

const settings = ref<QrCodeSettings>({
  enabled: true,
  template: 'wide-left',
  watermark: 'Power by 灵衍云',
  background: backgroundColors[0],
  title: { ...createRow('未命名表单', '', 22), source: 'fixed' },
  subtitle: createRow('', '', 14),
  fields: [
    createRow('用户名', '用户名'),
    createRow('用户类型', '用户类型'),
    createRow('性别', '性别'),
  ],
});
const svgMarkup = shallowRef('');
const svgPreviewUrl = shallowRef('');
const draggedRowIndex = shallowRef<number | null>(null);
const persistedTemplate = shallowRef<LabelTemplateDetailDto | null>(null);
const saving = shallowRef(false);
const downloading = shallowRef(false);
const realPreviewing = shallowRef(false);

const currentTemplate = computed(
  () => templates.find((item) => item.kind === settings.value.template) ?? templates[0],
);
const formFieldOptions = computed<FieldOption[]>(() =>
  (detail.value?.draft.content.items ?? [])
    .filter((item: FormItem) => item.widget.type !== 'separator' && item.widget.type !== 'button')
    .map((item: FormItem) => ({
      label: item.label,
      value: item.widget.fieldId ?? item.widget.widgetName,
    })),
);

function createDefaultSettings(): QrCodeSettings {
  const fields = formFieldOptions.value.slice(0, 3);
  return {
    enabled: true,
    template: 'wide-left',
    watermark: 'Power by 灵衍云',
    background: backgroundColors[0],
    title: { ...createRow(detail.value?.name ?? '未命名表单', '', 22), source: 'fixed' },
    subtitle: createRow('', '', 14),
    fields: fields.length
      ? fields.map((field) => createRow(field.label, field.value))
      : [
          createRow('用户名', '用户名'),
          createRow('用户类型', '用户类型'),
          createRow('性别', '性别'),
        ],
  };
}
const visibleFields = computed(() =>
  settings.value.fields.slice(0, currentTemplate.value.maxFields),
);

function rowValueSource(row: LabelContentRow, title = false): ValueSource {
  if (row.source === 'fixed')
    return { type: 'static', value: row.value.trim() || (title ? row.label : '') };
  // “实例标题”在标签协议中不是可授权表单字段，降级为当前表单名称静态值。
  if (row.value === 'sys.title')
    return { type: 'static', value: detail.value?.name ?? '未命名表单' };
  if (row.value === 'sys.submittedBy') return { type: 'system', key: 'createdBy' };
  if (row.value === 'sys.submittedAt') return { type: 'system', key: 'createdAt' };
  if (row.value === 'sys.updatedAt') return { type: 'system', key: 'updatedAt' };
  return { type: 'field', fieldId: row.value };
}

function textStyle(row: LabelContentRow, color: string, weight = 600): TextStyle {
  return {
    fontFamily: 'Noto Sans CJK SC',
    fontSize: pixelsToMillimeters(row.fontSize, 96),
    fontWeight: weight,
    color,
    lineHeight: 1.25,
    textAlign: 'left',
    verticalAlign: 'middle',
    overflow: 'ellipsis',
  };
}

function elementBase(
  id: string,
  x: number,
  y: number,
  width: number,
  height: number,
  zIndex: number,
) {
  return {
    id,
    x,
    y,
    width,
    height,
    rotation: 0,
    zIndex,
    visible: true,
    locked: false,
  };
}

/** 截图式快速设置只负责组装 Schema；预览和导出始终经由 Label Engine 渲染。 */
const labelSchema = computed<LabelSchema>(() => {
  const template = currentTemplate.value;
  const portrait = template.ratio === '2:3';
  const pageWidth = portrait ? 60 : 90;
  const pageHeight = portrait ? 90 : 60;
  const accent = settings.value.background;
  const isPlain = !template.accent;
  const headerHeight = template.kind === 'qr-only' ? 12 : 16;
  const bodyTop = headerHeight;
  const contentHeight = pageHeight - bodyTop;
  const qrSize = portrait ? 31 : 29;
  const qrX =
    template.qrPosition === 'right'
      ? pageWidth - qrSize - 5
      : template.qrPosition === 'center'
        ? (pageWidth - qrSize) / 2
        : 5;
  const qrY = portrait ? 21 : bodyTop + (contentHeight - qrSize) / 2;
  const textX = template.qrPosition === 'left' ? qrX + qrSize + 5 : 5;
  const textWidth = template.qrPosition === 'center' ? pageWidth - 10 : pageWidth - qrSize - 15;
  const elements: LabelElement[] = [];

  if (template.accent) {
    elements.push({
      ...elementBase('accent-border', 0.7, 0.7, pageWidth - 1.4, pageHeight - 1.4, 1),
      type: 'rect',
      fill: '#ffffff',
      stroke: accent,
      strokeWidth: 1.4,
      radius: 1.5,
    });
  }
  elements.push({
    ...elementBase(
      'header-background',
      template.accent ? 0.7 : 0,
      template.accent ? 0.7 : 0,
      pageWidth - (template.accent ? 1.4 : 0),
      headerHeight,
      2,
    ),
    type: 'rect',
    fill: template.accent ? accent : '#ffffff',
    radius: template.accent ? 1.2 : 0,
  });
  elements.push({
    ...elementBase('title', 5, 1.5, pageWidth - 10, headerHeight - 3, 3),
    type: 'text',
    value: rowValueSource(settings.value.title, true),
    style: textStyle(settings.value.title, isPlain ? '#1b2129' : '#ffffff', 700),
  });
  if (settings.value.subtitle.label || settings.value.subtitle.value) {
    elements.push({
      ...elementBase('subtitle', textX, bodyTop + 2, textWidth, 6, 4),
      type: 'field',
      label: settings.value.subtitle.label,
      value: rowValueSource(settings.value.subtitle),
      separator: settings.value.subtitle.label ? ': ' : '',
      style: textStyle(settings.value.subtitle, '#1b2129'),
    });
  }
  if (template.kind !== 'qr-only') {
    const startY = portrait ? qrY + qrSize + 2 : bodyTop + 5;
    visibleFields.value.forEach((row, index) => {
      elements.push({
        ...elementBase(
          `field-${row.id}`,
          portrait ? 5 : textX,
          startY + index * 7.2,
          portrait ? pageWidth - 10 : textWidth,
          6.4,
          10 + index,
        ),
        type: 'field',
        label: row.label,
        value: rowValueSource(row),
        separator: row.label ? ': ' : '',
        style: textStyle(row, '#1b2129'),
      });
    });
  }
  elements.push({
    ...elementBase('qrcode', qrX, qrY, qrSize, qrSize, 30),
    type: 'qrcode',
    value: { type: 'system', key: 'recordId' },
    options: {
      errorCorrection: 'M',
      quietZone: 1,
      foreground: '#000000',
      background: '#ffffff',
    },
  });
  elements.push({
    ...elementBase('watermark', pageWidth - 28, pageHeight - 4, 25, 2.8, 40),
    type: 'text',
    value: { type: 'static', value: settings.value.watermark },
    style: {
      fontFamily: 'Noto Sans CJK SC',
      fontSize: 1.5,
      fontWeight: 400,
      color: '#90939b',
      lineHeight: 1,
      textAlign: 'right',
      verticalAlign: 'middle',
      overflow: 'clip',
    },
  });

  return {
    schemaVersion: '1.0',
    name: `${detail.value?.name ?? '未命名表单'}二维码标签`,
    page: { width: pageWidth, height: pageHeight, unit: 'mm', dpi: 300, background: '#ffffff' },
    source: { type: 'form', formId: detail.value?.code ?? '' },
    elements,
    settings: { snapToGrid: true, gridSize: 1, showGrid: false },
  };
});

const previewData = computed<LabelRenderData>(() => ({
  fields: Object.fromEntries([
    ...formFieldOptions.value.map((field) => [field.value, `\${${field.label}}`]),
    ['sys.title', detail.value?.name ?? '未命名表单'],
  ]),
  system: {
    recordId: 'https://lingyanyun.com/q/PREVIEW',
    createdAt: '2026-09-22 10:30:00',
    updatedAt: '2026-09-22 10:30:00',
    createdBy: '提交人',
    currentUser: '当前用户',
    currentDate: '2026-09-22',
  },
}));

function selectTemplate(kind: TemplateKind): void {
  settings.value.template = kind;
}

function addField(): void {
  if (settings.value.fields.length >= currentTemplate.value.maxFields) {
    ElMessage.warning(`当前模板最多支持${currentTemplate.value.maxFields}个字段`);
    return;
  }
  settings.value.fields.push(createRow('', ''));
}

function removeField(index: number): void {
  settings.value.fields.splice(index, 1);
}

function startDrag(index: number): void {
  draggedRowIndex.value = index;
}

function dropField(index: number): void {
  const from = draggedRowIndex.value;
  draggedRowIndex.value = null;
  if (from === null || from === index) return;
  const [row] = settings.value.fields.splice(from, 1);
  if (row) settings.value.fields.splice(index, 0, row);
}

function valueSourceToRow(source: ValueSource, label: string, fontSize: number): LabelContentRow {
  if (source.type === 'static') {
    return { ...createRow(label, source.value, fontSize), source: 'fixed' };
  }
  if (source.type === 'system') {
    const systemValue: Partial<Record<Extract<ValueSource, { type: 'system' }>['key'], string>> = {
      createdBy: 'sys.submittedBy',
      createdAt: 'sys.submittedAt',
      updatedAt: 'sys.updatedAt',
    };
    return createRow(label, systemValue[source.key] ?? '', fontSize);
  }
  if (source.type === 'field') return createRow(label, source.fieldId, fontSize);
  return createRow(label, '', fontSize);
}

/** 从持久化 Schema 反投影快速设置面板；Schema 本身仍是唯一持久化事实源。 */
function applySchema(schema: LabelSchema): void {
  const qr = schema.elements.find((element) => element.type === 'qrcode');
  const accent = schema.elements.find(
    (element) => element.type === 'rect' && element.id === 'accent-border',
  );
  const fields = schema.elements.filter(
    (element) => element.type === 'field' && element.id.startsWith('field-'),
  );
  let template: TemplateKind;
  if (schema.page.height > schema.page.width) template = 'portrait';
  else if (fields.length === 0 && qr && qr.x + qr.width / 2 === schema.page.width / 2)
    template = 'qr-only';
  else if (qr && qr.x < schema.page.width / 2) template = accent ? 'split-left' : 'plain-left';
  else template = accent ? 'wide-left' : 'plain-right';

  const title = schema.elements.find(
    (element) => element.type === 'text' && element.id === 'title',
  );
  const subtitle = schema.elements.find(
    (element) => element.type === 'field' && element.id === 'subtitle',
  );
  const watermark = schema.elements.find(
    (element) => element.type === 'text' && element.id === 'watermark',
  );
  const header = schema.elements.find(
    (element) => element.type === 'rect' && element.id === 'header-background',
  );
  settings.value = {
    enabled: true,
    template,
    background:
      (accent?.type === 'rect' ? accent.stroke : undefined) ??
      (header?.type === 'rect' ? header.fill : undefined) ??
      backgroundColors[0],
    title:
      title?.type === 'text'
        ? valueSourceToRow(
            title.value,
            detail.value?.name ?? '未命名表单',
            Math.round(millimetersToPixels(title.style.fontSize, 96)),
          )
        : { ...createRow(detail.value?.name ?? '未命名表单', '', 22), source: 'fixed' },
    subtitle:
      subtitle?.type === 'field'
        ? valueSourceToRow(
            subtitle.value,
            subtitle.label ?? '',
            Math.round(millimetersToPixels(subtitle.style.fontSize, 96)),
          )
        : createRow('', '', 14),
    fields: fields.map((element) =>
      element.type === 'field'
        ? valueSourceToRow(
            element.value,
            element.label ?? '',
            Math.round(millimetersToPixels(element.style.fontSize, 96)),
          )
        : createRow('', '', 14),
    ),
    watermark:
      watermark?.type === 'text' && watermark.value.type === 'static'
        ? watermark.value.value
        : 'Power by 灵衍云',
  };
}

async function loadSettings(formCode: string): Promise<void> {
  try {
    const page = await listLabelTemplates({ formCode, limit: 1 });
    if (detail.value?.code !== formCode) return;
    if (page.items.length === 0) {
      persistedTemplate.value = null;
      settings.value = createDefaultSettings();
      return;
    }
    const loaded = await getLabelTemplate(page.items[0].code);
    if (detail.value?.code !== formCode) return;
    persistedTemplate.value = loaded;
    applySchema(loaded.draft);
  } catch (error) {
    persistedTemplate.value = null;
    settings.value = createDefaultSettings();
    showLifecycleError(error, 'load');
  }
}

async function ensureTemplate(): Promise<LabelTemplateDetailDto> {
  if (persistedTemplate.value) return persistedTemplate.value;
  const formCode = detail.value?.code;
  if (!formCode) throw new Error('表单信息尚未加载');
  try {
    const created = await createLabelTemplate({
      name: `${detail.value?.name ?? '未命名表单'}二维码标签`,
      formCode,
      width: labelSchema.value.page.width,
      height: labelSchema.value.page.height,
      unit: labelSchema.value.page.unit,
      dpi: labelSchema.value.page.dpi,
    });
    persistedTemplate.value = created;
    return created;
  } catch (error) {
    // 多窗口首次保存由数据库唯一约束裁决；读取胜出的模板继续保存。
    if (!(error instanceof ApiError) || error.errCode !== 'LABEL_FORM_ALREADY_BOUND') throw error;
    const page = await listLabelTemplates({ formCode, limit: 1 });
    if (!page.items[0]) throw error;
    const loaded = await getLabelTemplate(page.items[0].code);
    persistedTemplate.value = loaded;
    return loaded;
  }
}

function stableValue(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(stableValue);
  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value)
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([key, item]) => [key, stableValue(item)]),
    );
  }
  return value;
}

function schemaFingerprint(schema: LabelSchema): string {
  return JSON.stringify(stableValue(schema));
}

/** 仅在 Schema 发生变化时推进草稿修订号，避免预览和下载制造无意义版本。 */
async function persistDraft(): Promise<LabelTemplateDetailDto> {
  const target = await ensureTemplate();
  if (schemaFingerprint(target.draft) === schemaFingerprint(labelSchema.value)) return target;
  const schema = structuredClone(labelSchema.value);
  const saved = await saveLabelTemplateDraft(target.code, {
    draftRevision: target.draftRevision,
    schema,
  });
  const updated = withSavedLabelDraft(target, schema, saved.draftRevision);
  persistedTemplate.value = updated;
  return updated;
}

function showLifecycleError(
  error: unknown,
  action: 'load' | 'preview' | 'publish' | 'render' | 'save',
): void {
  const feedback = labelLifecycleFeedback(error, action);
  ElMessage[feedback.tone](feedback.message);
  if (feedback.reload) {
    const formCode = detail.value?.code;
    if (formCode) void loadSettings(formCode);
  }
}

function showFilePreview(blob: Blob): void {
  const nextUrl = URL.createObjectURL(blob);
  if (svgPreviewUrl.value) URL.revokeObjectURL(svgPreviewUrl.value);
  svgPreviewUrl.value = nextUrl;
}

/** 真实数据预览成功后，后端会为当前 draftRevision 写入发布准入标记。 */
async function previewRealData(): Promise<void> {
  if (realPreviewing.value) return;
  try {
    const { value } = await ElMessageBox.prompt(
      '请输入用于校验标签效果的表单记录 ID',
      '真实数据预览',
      {
        confirmButtonText: '生成预览',
        cancelButtonText: '取消',
        inputPattern: /^[1-9]\d*$/u,
        inputErrorMessage: '记录 ID 必须是正整数',
      },
    );
    realPreviewing.value = true;
    const target = await persistDraft();
    const blob = await previewLabelTemplate(target.code, { recordId: value, format: 'svg' });
    showFilePreview(blob);
    persistedTemplate.value = withPreviewedLabelDraft(target);
    ElMessage.success('真实数据预览通过，当前草稿可以发布');
  } catch (error) {
    if (error === 'cancel' || error === 'close') return;
    showLifecycleError(error, 'preview');
  } finally {
    realPreviewing.value = false;
  }
}

/** 保存草稿不隐式发布；新修订会使旧真实预览准入自然失效。 */
async function saveDraftOnly(): Promise<boolean> {
  if (saving.value) return false;
  saving.value = true;
  try {
    await persistDraft();
    ElMessage.success('二维码标签草稿已保存');
    return true;
  } catch (error) {
    showLifecycleError(error, 'save');
    return false;
  } finally {
    saving.value = false;
  }
}

/** 发布显式冻结不可变版本；若本地有新编辑，先 CAS 保存再检查真实预览。 */
async function publishSettings(): Promise<boolean> {
  if (saving.value) return false;
  saving.value = true;
  try {
    const target = await persistDraft();
    if (target.publishedDraftRevision === target.draftRevision && target.publishedVersion > 0) {
      ElMessage.success(`二维码标签当前已是发布版本 V${target.publishedVersion}`);
      return true;
    }
    const published = await publishLabelTemplate(target.code, target.draftRevision);
    persistedTemplate.value = withPublishedLabelDraft(target, published.versionNo);
    ElMessage.success(`二维码标签已发布 V${published.versionNo}`);
    return true;
  } catch (error) {
    showLifecycleError(error, 'publish');
    return false;
  } finally {
    saving.value = false;
  }
}

async function downloadLabel(): Promise<void> {
  if (downloading.value) return;
  try {
    const { value } = await ElMessageBox.prompt('请输入需要生成标签的表单记录 ID', '下载标签', {
      confirmButtonText: '生成 SVG',
      cancelButtonText: '取消',
      inputPattern: /^[1-9]\d*$/u,
      inputErrorMessage: '记录 ID 必须是正整数',
    });
    downloading.value = true;
    if (!(await publishSettings())) return;
    const target = persistedTemplate.value;
    if (!target) return;
    const blob = await renderPublishedLabel({
      templateCode: target.code,
      recordId: value,
      format: 'svg',
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.download = `${detail.value?.name ?? '表单'}-${value}-二维码标签.svg`;
    link.href = url;
    link.click();
    URL.revokeObjectURL(url);
    ElMessage.success('正式 SVG 标签已生成');
  } catch (error) {
    if (error === 'cancel' || error === 'close') return;
    showLifecycleError(error, 'render');
  } finally {
    downloading.value = false;
  }
}

let previewRenderVersion = 0;
watch(
  [labelSchema, previewData],
  async ([schema, data]) => {
    const version = ++previewRenderVersion;
    try {
      const svg = await renderLabelSvg(schema, data);
      if (version !== previewRenderVersion) return;
      const nextUrl = URL.createObjectURL(new Blob([svg], { type: 'image/svg+xml;charset=utf-8' }));
      if (svgPreviewUrl.value) URL.revokeObjectURL(svgPreviewUrl.value);
      svgMarkup.value = svg;
      svgPreviewUrl.value = nextUrl;
    } catch {
      if (version !== previewRenderVersion) return;
      ElMessage.error('标签 SVG 预览生成失败');
    }
  },
  { immediate: true },
);
watch(
  () => detail.value?.code,
  (code) => {
    if (code) void loadSettings(code);
  },
  { immediate: true },
);
onBeforeUnmount(() => {
  if (svgPreviewUrl.value) URL.revokeObjectURL(svgPreviewUrl.value);
});
</script>

<template>
  <section class="qr-settings" aria-label="二维码标签设置">
    <header class="qr-settings__header">
      <h1>二维码标签设置</h1>
      <p>表单的每条数据可生成二维码标签，例如物料标签、固定资产标签等</p>
      <el-switch v-model="settings.enabled" inline-prompt active-text="开" inactive-text="关" />
    </header>

    <div
      class="qr-settings__workspace"
      :class="{ 'qr-settings__workspace--disabled': !settings.enabled }"
    >
      <section class="qr-settings__preview-panel" aria-label="标签预览与模板选择">
        <div class="qr-settings__preview-stage">
          <div
            class="label-preview"
            :class="{ 'label-preview--portrait': currentTemplate.ratio === '2:3' }"
          >
            <img :src="svgPreviewUrl" alt="SVG 标签实时预览">
          </div>
          <p>
            当前标签比例{{ currentTemplate.ratio }}，可下载尺寸：{{ currentTemplate.downloadSize }}
          </p>
          <p>
            标签样式及字段规则修改后，将对当前表单的所有数据的二维码生效
            <a href="#label-preview" @click.prevent="previewRealData">
              {{ realPreviewing ? '正在预览...' : '预览效果' }}
            </a>
          </p>
        </div>

        <div class="qr-settings__templates" role="list" aria-label="标签模板">
          <button
            v-for="template in templates"
            :key="template.kind"
            class="template-card"
            :class="{ 'template-card--active': template.kind === settings.template }"
            type="button"
            :title="template.name"
            :aria-pressed="template.kind === settings.template"
            @click="selectTemplate(template.kind)"
          >
            <span class="template-card__mini" :class="[`template-card__mini--${template.kind}`]">
              <i class="template-card__title" />
              <i class="template-card__text" />
              <i class="template-card__qr" />
            </span>
            <RiCheckLine v-if="template.kind === settings.template" />
          </button>
        </div>
      </section>

      <section class="qr-settings__form" aria-label="标签字段设置">
        <div class="qr-settings__form-heading">
          <h2>字段设置</h2>
          <span>字段内容超长时，将被自动截断!</span>
        </div>

        <div class="content-row content-row--heading">
          <RiDraggable aria-hidden="true" />
          <span>主标题</span>
          <el-input v-model="settings.title.label" maxlength="20" />
          <el-select v-model="settings.title.source">
            <el-option label="固定值" value="fixed" />
            <el-option label="字段" value="field" />
          </el-select>
          <el-input
            v-if="settings.title.source === 'fixed'"
            v-model="settings.title.value"
            placeholder="请输入"
          />
          <el-select v-else v-model="settings.title.value" filterable placeholder="请选择">
            <ElOptionGroup label="系统字段">
              <el-option
                v-for="field in systemFields"
                :key="field.value"
                :label="field.label"
                :value="field.value"
              />
            </ElOptionGroup>
            <ElOptionGroup label="表单字段">
              <el-option
                v-for="field in formFieldOptions"
                :key="field.value"
                :label="field.label"
                :value="field.value"
              />
            </ElOptionGroup>
          </el-select>
          <el-select v-model="settings.title.fontSize" class="content-row__font" title="字体大小">
            <el-option v-for="size in fontSizes" :key="size" :label="String(size)" :value="size" />
          </el-select>
        </div>

        <div class="content-row content-row--heading">
          <RiDraggable aria-hidden="true" />
          <span>副标题</span>
          <el-input v-model="settings.subtitle.label" maxlength="20" placeholder="请输入" />
          <el-select v-model="settings.subtitle.source">
            <el-option label="固定值" value="fixed" />
            <el-option label="字段" value="field" />
          </el-select>
          <el-input
            v-if="settings.subtitle.source === 'fixed'"
            v-model="settings.subtitle.value"
            placeholder="请输入"
          />
          <el-select v-else v-model="settings.subtitle.value" filterable placeholder="请选择">
            <ElOptionGroup label="系统字段">
              <el-option
                v-for="field in systemFields"
                :key="field.value"
                :label="field.label"
                :value="field.value"
              />
            </ElOptionGroup>
            <ElOptionGroup label="表单字段">
              <el-option
                v-for="field in formFieldOptions"
                :key="field.value"
                :label="field.label"
                :value="field.value"
              />
            </ElOptionGroup>
          </el-select>
          <el-select
            v-model="settings.subtitle.fontSize"
            class="content-row__font"
            title="字体大小"
          >
            <el-option v-for="size in fontSizes" :key="size" :label="String(size)" :value="size" />
          </el-select>
        </div>

        <p class="qr-settings__field-tip">
          当前选择的模板最多支持展示{{ currentTemplate.maxFields }}个字段，默认展示以下前{{
            currentTemplate.maxFields
          }}个字段
        </p>

        <div
          v-for="(row, index) in settings.fields"
          :key="row.id"
          class="content-row content-row--field"
          draggable="true"
          @dragstart="startDrag(index)"
          @dragover.prevent
          @drop="dropField(index)"
        >
          <RiDraggable class="content-row__drag" aria-label="拖动排序" />
          <span>{{ index + 1 }}</span>
          <el-input v-model="row.label" maxlength="20" placeholder="请输入" />
          <el-select v-model="row.source">
            <el-option label="固定值" value="fixed" />
            <el-option label="字段" value="field" />
          </el-select>
          <el-input v-if="row.source === 'fixed'" v-model="row.value" placeholder="请输入" />
          <el-select v-else v-model="row.value" filterable placeholder="请选择">
            <ElOptionGroup label="系统字段">
              <el-option
                v-for="field in systemFields"
                :key="field.value"
                :label="field.label"
                :value="field.value"
              />
            </ElOptionGroup>
            <ElOptionGroup label="表单字段">
              <el-option
                v-for="field in formFieldOptions"
                :key="field.value"
                :label="field.label"
                :value="field.value"
              />
            </ElOptionGroup>
          </el-select>
          <el-select v-model="row.fontSize" class="content-row__font" title="字体大小">
            <el-option v-for="size in fontSizes" :key="size" :label="String(size)" :value="size" />
          </el-select>
          <button type="button" aria-label="删除字段" @click="removeField(index)">
            <RiDeleteBin6Line />
          </button>
        </div>

        <el-button
          :disabled="settings.fields.length >= currentTemplate.maxFields"
          @click="addField"
        >
          <RiAddCircleLine />添加一项
        </el-button>

        <label class="qr-settings__control-label" for="watermark">水印内容</label>
        <el-input id="watermark" v-model="settings.watermark" maxlength="20" show-word-limit />

        <span class="qr-settings__control-label">标签背景</span>
        <div class="qr-settings__colors" role="radiogroup" aria-label="标签背景颜色">
          <button
            v-for="color in backgroundColors"
            :key="color"
            type="button"
            :style="{ backgroundColor: color }"
            :aria-label="`选择背景颜色 ${color}`"
            :aria-checked="settings.background === color"
            role="radio"
            @click="settings.background = color"
          >
            <RiCheckLine v-if="settings.background === color" />
          </button>
        </div>
      </section>
    </div>

    <footer class="qr-settings__footer">
      <el-button type="primary" :loading="saving" @click="saveDraftOnly">
        保存草稿
      </el-button>
      <el-button plain type="primary" :loading="saving" @click="publishSettings">
        发布
      </el-button>
      <el-button plain type="primary" :loading="downloading" @click="downloadLabel">
        下载/打印标签
      </el-button>
    </footer>
  </section>
</template>

<style scoped lang="scss">
.qr-settings {
  display: flex;
  flex-direction: column;
  min-width: 980px;
  min-height: 100%;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);

  &__header {
    padding: var(--el-space-2xl) var(--el-space-3xl) var(--el-space-xl);

    h1,
    p {
      margin: 0;
    }

    h1 {
      font-size: var(--el-font-size-large);
      line-height: 28px;
    }

    p {
      margin: var(--el-space-xs) 0 var(--el-space-lg);
      color: var(--el-text-color-secondary);
    }
  }

  &__workspace {
    display: grid;
    grid-template-columns: 480px minmax(660px, 800px);
    gap: var(--el-space-2xl);
    align-items: start;
    padding: 0 var(--el-space-3xl) var(--el-space-3xl);
    transition: opacity 0.18s ease;

    &--disabled {
      pointer-events: none;
      opacity: 0.45;
    }
  }

  &__preview-panel {
    overflow: hidden;
    background: var(--el-bg-color-page);
  }

  &__preview-stage {
    min-height: 334px;
    padding: var(--el-space-6xl) var(--el-space-2xl) var(--el-space-2xl);
    text-align: center;

    > p {
      margin: var(--el-space-lg) 0 0;
      font-size: var(--el-font-size-small);
      color: var(--el-text-color-secondary);

      & + p {
        margin-top: var(--el-space-md);
      }
    }

    a {
      margin-left: var(--el-space-sm);
      text-decoration: none;
    }
  }

  &__templates {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--el-space-lg);
    padding: var(--el-space-2xl);
    background: var(--el-fill-color-light);
  }

  &__form {
    min-width: 0;
  }

  &__form-heading {
    display: flex;
    gap: var(--el-space-lg);
    align-items: baseline;
    margin-bottom: var(--el-space-lg);

    h2 {
      margin: 0;
      font-size: var(--el-font-size-large);
    }

    span {
      font-size: var(--el-font-size-small);
      color: var(--el-color-warning);
    }
  }

  &__field-tip {
    padding-top: var(--el-space-2xl);
    margin: var(--el-space-xl) 0 var(--el-space-lg);
    font-size: var(--el-font-size-small);
    color: var(--el-text-color-secondary);
    border-top: 1px dashed var(--el-border-color-lighter);
  }

  &__control-label {
    display: block;
    margin: var(--el-space-3xl) 0 var(--el-space-lg);
    color: var(--el-text-color-primary);
  }

  &__colors {
    display: flex;
    gap: var(--el-space-md);

    button {
      display: inline-flex;
      align-items: flex-end;
      justify-content: flex-end;
      width: 48px;
      height: 48px;
      padding: 0;
      color: var(--el-color-white);
      cursor: pointer;
      border: 2px solid transparent;
      border-radius: var(--el-border-radius-medium);

      svg {
        width: 16px;
        height: 16px;
      }

      &:focus-visible {
        outline: none;
        border-color: var(--el-text-color-primary);
      }
    }
  }

  &__footer {
    position: sticky;
    bottom: 0;
    z-index: 2;
    display: flex;
    gap: var(--el-space-md);
    padding: var(--el-space-lg) var(--el-space-3xl);
    background: var(--el-bg-color-page);
  }
}

.label-preview {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 360px;
  height: 240px;
  margin: 0 auto;

  &--portrait {
    width: 210px;
    height: 315px;
  }

  img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: contain;
    background: var(--el-bg-color);
    border-radius: var(--el-border-radius-medium);
    box-shadow: var(--el-box-shadow-light);
  }
}

.template-card {
  position: relative;
  height: 66px;
  padding: var(--el-space-xs);
  cursor: pointer;
  background: var(--el-bg-color);
  border: 2px solid transparent;
  border-radius: var(--el-border-radius-medium);

  > svg {
    position: absolute;
    right: 0;
    bottom: 0;
    width: 15px;
    height: 15px;
    color: var(--el-color-white);
    background: var(--el-color-primary);
    border-radius: var(--el-border-radius-half) 0 0;
  }

  &--active,
  &:hover {
    border-color: var(--el-color-primary);
  }

  &__mini {
    position: relative;
    display: block;
    width: 100%;
    height: 100%;
    overflow: hidden;
    background: var(--el-bg-color);

    &::before {
      position: absolute;
      top: 5px;
      left: 6px;
      width: 50%;
      height: 5px;
      content: '';
      background: var(--el-color-primary);
      border-radius: var(--el-border-radius-small);
    }
  }

  &__text,
  &__qr {
    position: absolute;
    display: block;
  }

  &__text {
    top: 20px;
    left: 7px;
    width: 44%;
    height: 24px;
    background: repeating-linear-gradient(
      to bottom,
      var(--el-text-color-secondary) 0 2px,
      transparent 2px 7px
    );
  }

  &__qr {
    right: 7px;
    bottom: 6px;
    width: 28px;
    height: 28px;
    background: repeating-conic-gradient(
        var(--el-text-color-primary) 0 25%,
        var(--el-bg-color) 0 50%
      )
      50% / 6px 6px;
  }

  &__mini--split-left,
  &__mini--plain-left,
  &__mini--compact-left {
    .template-card__text {
      right: 6px;
      left: auto;
    }

    .template-card__qr {
      right: auto;
      left: 7px;
    }
  }

  &__mini--wide-left,
  &__mini--wide-right,
  &__mini--split-left,
  &__mini--portrait {
    border-top: 9px solid var(--el-color-primary);
  }

  &__mini--portrait,
  &__mini--qr-only {
    width: 42px;
    margin: 0 auto;

    .template-card__qr {
      right: 6px;
    }

    .template-card__text {
      display: none;
    }
  }

  &__mini--table .template-card__text {
    width: 54%;
    background: repeating-linear-gradient(
      to bottom,
      var(--el-border-color) 0 1px,
      transparent 1px 7px
    );
    border: 1px solid var(--el-border-color);
  }
}

.content-row {
  box-sizing: border-box;
  display: grid;
  grid-template-columns: 16px 42px minmax(140px, 1.1fr) 112px minmax(210px, 1.8fr) 64px 24px;
  gap: var(--el-space-md);
  align-items: center;
  min-height: 48px;
  padding: var(--el-space-md);
  margin-bottom: var(--el-space-md);
  background: var(--el-fill-color-light);
  border-radius: var(--el-border-radius-medium);

  > svg {
    width: 16px;
    height: 16px;
    color: var(--el-text-color-secondary);
  }

  > span {
    text-align: center;
    white-space: nowrap;
  }

  > button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    padding: 0;
    color: var(--el-text-color-primary);
    cursor: pointer;
    background: transparent;
    border: 0;

    svg {
      width: 16px;
      height: 16px;
    }
  }

  &--heading {
    grid-template-columns: 16px 48px minmax(140px, 1.1fr) 112px minmax(210px, 1.8fr) 64px;
  }

  &__drag {
    cursor: grab;
  }
}

@media (width <= 1380px) {
  .qr-settings {
    &__workspace {
      grid-template-columns: 420px minmax(620px, 1fr);
    }
  }

  .label-preview {
    transform: scale(0.9);
  }
}
</style>
