<script setup lang="ts">
import { computed, ref, useTemplateRef, watch } from 'vue';
import {
  EvolynTable,
  type EvolynTableColumn,
  type EvolynTableCustomRenderElement,
} from '@evolyn.do/ui';

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

/** 单条登录日志；时间与后端 JSONTime 出网格式一致（秒级 yyyy-MM-dd HH:mm:ss） */
interface LoginLogRecord {
  loggedAt: string;
  location: string;
  platform: string;
  ip: string;
}

// 登录日志查询接口后端尚未提供，先以「当前时间往前偏移」生成演示数据，
// 保证筛选/分页交互真实可验；接口落地后将 records 替换为 API 拉取即可。
function formatDateTime(date: Date): string {
  const pad = (value: number) => String(value).padStart(2, '0');
  return (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ` +
    `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
  );
}

/** 以「距现在 hoursAgo 小时」生成一条记录，地点/IP 取固定样本值 */
function createRecord(hoursAgo: number, ip: string, platform = '电脑网页版'): LoginLogRecord {
  return {
    loggedAt: formatDateTime(new Date(Date.now() - hoursAgo * 3_600_000)),
    location: '广东省 深圳市',
    platform,
    ip,
  };
}

const records: LoginLogRecord[] = [
  createRecord(2, '210.21.226.222'),
  createRecord(9, '210.21.226.222'),
  createRecord(26, '27.46.93.3'),
  createRecord(41, '210.21.226.222'),
  createRecord(60, '27.46.93.4'),
  createRecord(84, '27.46.93.3', '手机 App'),
  createRecord(108, '183.238.228.138'),
  createRecord(130, '210.21.226.222'),
  createRecord(156, '27.46.93.3'),
  createRecord(180, '183.238.228.138', '手机 App'),
  createRecord(204, '210.21.226.222'),
  createRecord(252, '27.46.93.4'),
  createRecord(300, '210.21.226.222'),
];

const PAGE_SIZE = 6;

const startDate = ref('');
const endDate = ref('');
const currentPage = ref(1);

// 起止日期均为闭区间；value-format 与日志时间同为字典序可比的 yyyy-mm-dd 前缀，直接比较字符串
const filteredRecords = computed(() =>
  records.filter((item) => {
    if (startDate.value && item.loggedAt < `${startDate.value} 00:00:00`) return false;
    if (endDate.value && item.loggedAt > `${endDate.value} 23:59:59`) return false;
    return true;
  }),
);

const pagedRecords = computed(() =>
  filteredRecords.value.slice((currentPage.value - 1) * PAGE_SIZE, currentPage.value * PAGE_SIZE),
);

// 筛选条件变化后回到第一页，避免停留在已超出总页数的页码
watch([startDate, endDate], () => {
  currentPage.value = 1;
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

/** 首列「头像 + 昵称」富单元格：以 40px 行高为基准绝对定位（24px 头像垂直居中） */
const columns = computed<EvolynTableColumn[]>(() => {
  const textY = ROW_HEIGHT / 2;
  // 元素字面量需显式标注，否则 type 字段被拓宽为 string 与 VTable 元素联合类型不兼容
  const avatar: EvolynTableCustomRenderElement = props.avatar
    ? {
        type: 'image',
        x: 0,
        y: (ROW_HEIGHT - 24) / 2,
        width: 24,
        height: 24,
        src: props.avatar,
        shape: 'circle',
      }
    : {
        type: 'circle',
        x: 12,
        y: textY,
        radius: 12,
        fill: canvasTokens.value.primary,
      };

  const fallbackInitial: EvolynTableCustomRenderElement[] = props.avatar
    ? []
    : [
        {
          type: 'text',
          x: 12,
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
            x: 32,
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
        :records="pagedRecords"
        :options="tableOptions"
      />

      <div class="login-log-drawer__pagination">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="PAGE_SIZE"
          :total="filteredRecords.length"
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
