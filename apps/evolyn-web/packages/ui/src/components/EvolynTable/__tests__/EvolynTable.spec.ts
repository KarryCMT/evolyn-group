import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ListTable } from '@visactor/vtable';

const setRecords = vi.fn();
const updateOption = vi.fn();
const release = vi.fn();
const on = vi.fn();
/** 捕获最近一次实例化时传入的 options，便于断言归一化结果 */
let lastOptions: Record<string, unknown> | undefined;

// happy-dom 没有 canvas 实现，mock 掉核心构造器只验证装配与更新策略
// 注意：实现必须用 function 而非箭头函数，Vitest 4 的箭头实现不可被 new 调用
vi.mock('@visactor/vtable', () => ({
  ListTable: vi.fn().mockImplementation(function MockListTable(
    _container: HTMLElement,
    options: Record<string, unknown>,
  ) {
    lastOptions = options;
    return { setRecords, updateOption, release, on };
  }),
}));

// vi.mock 存在提升，被测组件需在 mock 声明之后导入
import EvolynTable from '../EvolynTable.vue';
import { createElementTheme } from '../theme';

const columns = [{ field: 'name', title: '名称' }];

describe('EvolynTable', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    lastOptions = undefined;
  });

  it('mounts a ListTable instance with normalized options', () => {
    const wrapper = mount(EvolynTable, {
      props: {
        columns: [{ field: 'name', title: '名称', sortable: true, align: 'center' }],
        records: [{ name: '张三' }],
      },
    });

    expect(ListTable).toHaveBeenCalledTimes(1);
    expect(wrapper.classes()).toContain('evolyn-table');

    const options = lastOptions as Record<string, any>;
    // 数据与空态文案进入初始 options
    expect(options.records).toEqual([{ name: '张三' }]);
    expect(options.emptyTip).toEqual({ text: '暂无数据' });
    // 列定义归一化：sort/样式对齐映射到 VTable 列字段
    const [column] = options.columns as Array<Record<string, any>>;
    expect(column.field).toBe('name');
    expect(column.title).toBe('名称');
    expect(column.sort).toBe(true);
    expect(column.style?.textAlign).toBe('center');
    // 默认接入 EP 主题
    expect(options.theme).toBeDefined();
  });

  it('passes empty records as-is without falling back to options.records', () => {
    mount(EvolynTable, { props: { columns, records: [] } });
    expect(lastOptions?.records).toEqual([]);
  });

  it('refreshes records via setRecords instead of full updateOption', async () => {
    const wrapper = mount(EvolynTable, { props: { columns, records: [{ name: 'a' }] } });

    await wrapper.setProps({ records: [{ name: 'b' }] });

    expect(setRecords).toHaveBeenCalledTimes(1);
    expect(setRecords).toHaveBeenCalledWith([{ name: 'b' }]);
    expect(updateOption).not.toHaveBeenCalled();
  });

  it('rebuilds options and carries current records on columns change', async () => {
    const wrapper = mount(EvolynTable, { props: { columns, records: [{ name: 'a' }] } });

    await wrapper.setProps({ columns: [{ field: 'age', title: '年龄' }] });

    expect(updateOption).toHaveBeenCalledTimes(1);
    const options = updateOption.mock.calls[0]?.[0] as Record<string, any>;
    expect((options.columns as Array<Record<string, any>>)[0]?.field).toBe('age');
    // 全量更新必须携带当前数据，避免结构变更后表格被清空
    expect(options.records).toEqual([{ name: 'a' }]);
  });

  it('releases the ListTable instance on unmount', () => {
    const wrapper = mount(EvolynTable, { props: { columns } });
    wrapper.unmount();
    expect(release).toHaveBeenCalledTimes(1);
  });

  it('forwards vtable snake_case events as kebab-case emits', () => {
    const wrapper = mount(EvolynTable, { props: { columns } });

    // 从实例 on 的调用中取出 click_cell 的转发回调并触发
    const binding = on.mock.calls.find(([name]) => name === 'click_cell');
    const handler = binding?.[1] as (args: unknown) => void;
    handler({ col: 1, row: 2 });

    expect(wrapper.emitted('click-cell')).toHaveLength(1);
    expect(wrapper.emitted('click-cell')?.[0]).toEqual([{ col: 1, row: 2 }]);
  });

  it('builds selection fills with alpha so overlay cannot cover cell content', () => {
    // happy-dom 读不到 --el-* 变量，主题走 #409eff 兜底值
    const theme = createElementTheme('light');
    const selection = theme.selectionStyle ?? {};

    // 选中填充画在内容之上的覆盖层，不透明色会把单元格文字整块盖住（历史缺陷），
    // 锁定填充必须为 rgba 半透明
    expect(selection.cellBgColor).toBe('rgba(64, 158, 255, 0.1)');
    expect(selection.inlineRowBgColor).toBe('rgba(64, 158, 255, 0.1)');
    // 描边为不透明主色，保证选中框可见
    expect(selection.cellBorderColor).toBe('#409eff');
  });
});
