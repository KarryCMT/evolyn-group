import type { IntelligentActionType } from '../schema';

export interface IntelligentNodeTemplate {
  type: IntelligentActionType;
  name: string;
  description: string;
  group: 'data' | 'message' | 'logic';
}

/** 可插入节点目录；后续接服务端能力目录时保持该投影协议不变。 */
export const intelligentNodeTemplates: readonly IntelligentNodeTemplate[] = [
  {
    type: 'create-record',
    name: '新增数据',
    description: '向目标表单新增一条数据',
    group: 'data',
  },
  {
    type: 'update-record',
    name: '修改数据',
    description: '修改目标表单中的匹配数据',
    group: 'data',
  },
  {
    type: 'delete-record',
    name: '删除数据',
    description: '删除目标表单中的匹配数据',
    group: 'data',
  },
  {
    type: 'send-notification',
    name: '发送通知',
    description: '向指定成员发送消息通知',
    group: 'message',
  },
  {
    type: 'http-request',
    name: '发送 HTTP 请求',
    description: '调用外部 HTTP 服务',
    group: 'message',
  },
  {
    type: 'condition',
    name: '条件分支',
    description: '根据条件决定后续执行路径',
    group: 'logic',
  },
  {
    type: 'data-transform',
    name: '数据转换',
    description: '将前序节点数据转换为目标结构',
    group: 'logic',
  },
] as const;

export type IntelligentFormFieldKind =
  | 'member'
  | 'text'
  | 'department'
  | 'choice'
  | 'date'
  | 'address';

export interface IntelligentFormFieldOption {
  value: string;
  label: string;
  kind: IntelligentFormFieldKind;
  choices?: readonly string[];
}

export const mockFormFields: readonly IntelligentFormFieldOption[] = [
  { value: 'employee_name', label: '员工姓名', kind: 'member' },
  { value: 'contact_phone', label: '联系电话', kind: 'text' },
  { value: 'department', label: '所属部门', kind: 'department' },
  { value: 'position', label: '岗位', kind: 'text' },
  { value: 'id_card', label: '身份证号码', kind: 'text' },
  { value: 'gender', label: '性别', kind: 'choice', choices: ['男', '女'] },
  { value: 'birthday', label: '出生日期', kind: 'date' },
  {
    value: 'ethnicity',
    label: '民族',
    kind: 'choice',
    choices: ['汉族', '蒙古族', '回族', '藏族', '维吾尔族'],
  },
  { value: 'household_address', label: '户籍地址', kind: 'address' },
  {
    value: 'marital_status',
    label: '婚姻状态',
    kind: 'choice',
    choices: ['未婚', '已婚', '离异'],
  },
] as const;
