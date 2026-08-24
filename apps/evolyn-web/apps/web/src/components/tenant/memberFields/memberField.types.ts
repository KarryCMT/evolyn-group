/** 成员档案字段在设置页与卡片预览页共用的展示模型。 */
export interface MemberProfileField {
  key: string;
  label: string;
  type: string;
  /** 字段设置中是否可由管理员调整个人页可见性。 */
  visibilityLocked?: boolean;
  /** 字段设置中是否可由管理员调整个人页编辑权限。 */
  editableLocked?: boolean;
  visible: boolean;
  editable: boolean;
  /** 成员名为资料卡固定信息，不参与选择。 */
  cardLocked?: boolean;
  cardVisible: boolean;
}

export const memberProfileFields: MemberProfileField[] = [
  {
    key: 'name',
    label: '姓名',
    type: '单行文本',
    visibilityLocked: true,
    editableLocked: true,
    visible: true,
    editable: false,
    cardLocked: true,
    cardVisible: true,
  },
  {
    key: 'code',
    label: '编号',
    type: '单行文本',
    visibilityLocked: true,
    editableLocked: true,
    visible: false,
    editable: false,
    cardVisible: true,
  },
  {
    key: 'mobile',
    label: '手机',
    type: '单行文本',
    visibilityLocked: true,
    editableLocked: true,
    visible: true,
    editable: true,
    cardVisible: true,
  },
  {
    key: 'email',
    label: '邮箱',
    type: '单行文本',
    visibilityLocked: true,
    editableLocked: true,
    visible: true,
    editable: true,
    cardVisible: true,
  },
  {
    key: 'department',
    label: '部门',
    type: '部门多选',
    visibilityLocked: true,
    editableLocked: true,
    visible: false,
    editable: false,
    cardVisible: true,
  },
  {
    key: 'role',
    label: '角色',
    type: '角色多选',
    visibilityLocked: true,
    editableLocked: true,
    visible: false,
    editable: false,
    cardVisible: false,
  },
  {
    key: 'alias',
    label: '别名',
    type: '单行文本',
    visible: false,
    editable: false,
    cardVisible: false,
  },
  {
    key: 'employeeId',
    label: '工号',
    type: '单行文本',
    visible: false,
    editable: false,
    cardVisible: false,
  },
  {
    key: 'gender',
    label: '性别',
    type: '单行文本',
    visible: false,
    editable: false,
    cardVisible: false,
  },
  {
    key: 'position',
    label: '职务',
    type: '单行文本',
    visible: false,
    editable: false,
    cardVisible: false,
  },
  {
    key: 'employment',
    label: '聘用形式',
    type: '单行文本',
    visible: false,
    editable: false,
    cardVisible: false,
  },
  {
    key: 'hireDate',
    label: '入职日期',
    type: '日期时间',
    visible: false,
    editable: false,
    cardVisible: false,
  },
  {
    key: 'workplace',
    label: '工作地点',
    type: '单行文本',
    visible: false,
    editable: false,
    cardVisible: false,
  },
  {
    key: 'birthday',
    label: '出生日期',
    type: '日期时间',
    visible: false,
    editable: false,
    cardVisible: false,
  },
  {
    key: 'education',
    label: '学历',
    type: '单行文本',
    visible: false,
    editable: false,
    cardVisible: false,
  },
];

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
