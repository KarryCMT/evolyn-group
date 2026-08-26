/**
 * 成员信息管理页面的本地展示物料。
 * 字段目录（key/label/type/锁定规则/配置值）以服务端字段配置快照为唯一
 * 来源（api/memberField.ts 的 MemberFieldSettingDto），本文件不再维护
 * 硬编码的可变配置；卡片预览页保留样例成员数据用于视觉预览，生产成员
 * 卡片必须消费服务端按 cardVisible 裁剪后的数据。
 */

/** 卡片预览的样例成员取值（按字段 key 索引；姓名为卡片固定信息不参与勾选）。 */
export const memberPreviewValues: Record<string, string> = {
  code: 'Cloud001',
  mobile: '+86-13800138000',
  email: 'xiaofan@example.com',
  department: '商业化/大客户部/销售组',
  role: '管理员',
  alias: '帆小云',
  employeeId: 'LYY-0001',
  gender: '女',
  position: '销售经理',
  employment: '全职',
  hireDate: '2024-02-01',
  workplace: '上海',
  birthday: '1994-07-21',
  education: '本科',
};

/** 卡片预览的样例成员固定信息（与字段配置无关）。 */
export const memberPreviewIdentity = {
  avatarText: '帆',
  name: '帆小云',
  tag: '内部成员',
};
