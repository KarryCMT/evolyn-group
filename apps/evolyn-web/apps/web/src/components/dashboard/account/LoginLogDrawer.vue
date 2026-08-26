<script setup lang="ts">
import { computed, shallowRef, useTemplateRef, watch } from 'vue';
import {
  EvolynTable,
  type EvolynTableColumn,
  type EvolynTableCustomRenderElement,
} from '@evolyn.do/ui';
import { listMyLoginLogs } from '~/api/account';
import type { LoginLogClient, LoginLogItem } from '~/types';

defineOptions({ name: 'LoginLogDrawer' });

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    /** 登录人显示名：成员昵称优先（ADR-006 账号×成员拆分），用于首列展示 */
    nickname?: string;
    /** 登录人头像地址；缺省渲染姓名首字的文字头像 */
    avatar?: string;
  }>(),
  { nickname: '', avatar: '' },
);

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
}>();

/** 登录平台展示文案：后端 client 枚举（UA 解析）在前端映射 */
const CLIENT_LABELS: Record<LoginLogClient, string> = {
  web: '电脑网页版',
  wap: '手机网页版',
  unknown: '未知',
};

const PAGE_SIZE = 6;

const startDate = shallowRef('');
const endDate = shallowRef('');
const currentPage = shallowRef(1);
const records = shallowRef<LoginLogItem[]>([]);
const total = shallowRef(0);
const loading = shallowRef(false);

/** 表格展示行：后端记录追加派生的「登录平台」文案列，其余字段直接透传 */
const displayRecords = computed(() =>
  records.value.map((item) => ({
    ...item,
    platform: CLIENT_LABELS[item.client] ?? '未知',
  })),
);

/** 拉取当前筛选/页码下的登录日志；失败由 HTTP 层统一提示，这里只复位加载态 */
async function loadLogs() {
  if (loading.value) return;
  loading.value = true;
  try {
    const page = await listMyLoginLogs({
      page: currentPage.value,
      pageSize: PAGE_SIZE,
      startDate: startDate.value || undefined,
      endDate: endDate.value || undefined,
    });
    records.value = page.items;
    total.value = page.total;
  } finally {
    loading.value = false;
  }
}

// 抽屉打开即拉当前视图：用户可能刚完成一次新登录，需反映最新流水
watch(
  () => props.modelValue,
  (open) => {
    if (open) void loadLogs();
  },
);

// 筛选变化回到第一页：已在第一页时直接拉取，否则由页码 watcher 触发，避免双请求
watch([startDate, endDate], () => {
  if (currentPage.value === 1) {
    void loadLogs();
  } else {
    currentPage.value = 1;
  }
});

watch(currentPage, () => {
  void loadLogs();
});

const drawerBody = useTemplateRef<HTMLElement>('drawerBody');

/**
 * 首列画布渲染用的具体色值。VTable 是 canvas 渲染无法消费 CSS 变量，
 * 从抽屉内容根元素按级联实际值读取（个人设置页局部覆盖了品牌主色）。
 */
const canvasTokens = computed(() => {
  const el = drawerBody.value;
  const style = el ? getComputedStyle(el) : null;
  const read = (name: string, fallback: string) => style?.getPropertyValue(name).trim() || fallback;
  return {
    primary: read('--el-color-primary', '#1677ff'),
    textRegular: read('--el-text-color-regular', '#606266'),
  };
});

/** 文字头像取姓名首字，与登录日志列表的头像形态一致 */
const avatarFallback = computed(() => props.nickname.trim().charAt(0) || '我');
const displayName = computed(() => props.nickname || '当前用户');

const ROW_HEIGHT = 40;
// 富单元格从表格边框开始定位，统一预留与 VTable 普通文本单元格一致的左右留白。
const CELL_HORIZONTAL_PADDING = 12;

/** 首列「头像 + 昵称」富单元格：以 40px 行高为基准绝对定位（24px 头像垂直居中） */
const columns = computed<EvolynTableColumn[]>(() => {
  const textY = ROW_HEIGHT / 2;
  // 元素字面量需显式标注，否则 type 字段被拓宽为 string 与 VTable 元素联合类型不兼容
  const avatar: EvolynTableCustomRenderElement = props.avatar
    ? {
        type: 'image',
        x: CELL_HORIZONTAL_PADDING,
        y: (ROW_HEIGHT - 24) / 2,
        width: 24,
        height: 24,
        src: props.avatar,
        shape: 'circle',
      }
    : {
        type: 'circle',
        x: CELL_HORIZONTAL_PADDING + 12,
        y: textY,
        radius: 12,
        fill: canvasTokens.value.primary,
      };

  const fallbackInitial: EvolynTableCustomRenderElement[] = props.avatar
    ? []
    : [
        {
          type: 'text',
          x: CELL_HORIZONTAL_PADDING + 12,
          y: textY,
          text: avatarFallback.value,
          fill: '#fff',
          fontSize: 12,
          textAlign: 'center',
          textBaseline: 'middle',
        },
      ];

  return [
    {
      field: 'loggedAt',
      title: '登录人',
      width: 150,
      customRender: () => ({
        expectedWidth: 150,
        expectedHeight: ROW_HEIGHT,
        elements: [
          avatar,
          ...fallbackInitial,
          {
            type: 'text',
            x: CELL_HORIZONTAL_PADDING + 32,
            y: textY,
            text: displayName.value,
            fill: canvasTokens.value.textRegular,
            fontSize: 14,
            textBaseline: 'middle',
          },
        ],
      }),
    },
    { field: 'loggedAt', title: '登录时间', minWidth: 150 },
    { field: 'location', title: '登录地', minWidth: 100 },
    { field: 'platform', title: '登录平台', minWidth: 90 },
    { field: 'ip', title: 'IP', minWidth: 120 },
  ];
});

// 行高显式固定为 40，保证 customRender 的绝对定位基准稳定
const tableOptions = { defaultRowHeight: ROW_HEIGHT };
</script>

<template>
  <el-drawer
    :model-value="modelValue"
    class="login-log-drawer-panel"
    title="登录日志"
    direction="btt"
    size="90vh"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div ref="drawerBody" class="login-log-drawer">
      <div class="login-log-drawer__filter">
        <span class="login-log-drawer__filter-label">登录时间</span>
        <el-date-picker
          v-model="startDate"
          type="date"
          placeholder="开始日期"
          value-format="YYYY-MM-DD"
          class="login-log-drawer__date-input"
        />
        <el-date-picker
          v-model="endDate"
          type="date"
          placeholder="结束日期"
          value-format="YYYY-MM-DD"
          class="login-log-drawer__date-input"
        />
      </div>

      <EvolynTable
        class="login-log-drawer__table"
        :columns="columns"
        :records="displayRecords"
        :options="tableOptions"
      />

      <div class="login-log-drawer__pagination">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="PAGE_SIZE"
          :total="total"
          layout="prev, pager, next"
        />
      </div>
    </div>
  </el-drawer>
</template>

<style scoped lang="scss">
.login-log-drawer {
  display: flex;
  height: 100%;
  flex-direction: column;

  &__filter {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 14px;
  }

  &__filter-label {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__date-input {
    width: 136px;
  }

  &__table {
    // 纵向占满剩余空间；min-height 防 flex 子项溢出父容器
    flex: 1;
    min-height: 0;
  }

  &__pagination {
    display: flex;
    justify-content: flex-end;
    padding-top: 14px;
  }
}
</style>

<!-- 默认头部在 body 外层，抽屉传送时以独立类统一标题栏尺寸。 -->
<style lang="scss">
.login-log-drawer-panel .el-drawer__header {
  height: 56px;
  box-sizing: border-box;
  align-items: center;
  margin: 0;
  padding: 0 16px 0 20px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.login-log-drawer-panel .el-drawer__title {
  color: var(--el-text-color-primary);
  font-size: 18px;
  font-weight: 650;
  line-height: 26px;
}

.login-log-drawer-panel .el-drawer__close-btn {
  display: inline-flex;
  width: 32px;
  height: 32px;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  cursor: pointer;
}

.login-log-drawer-panel .el-drawer__close-btn:hover {
  color: var(--el-color-primary);
  background: var(--el-fill-color-light);
}

.login-log-drawer-panel .el-drawer__close-btn .el-icon {
  font-size: 22px;
}
</style>
