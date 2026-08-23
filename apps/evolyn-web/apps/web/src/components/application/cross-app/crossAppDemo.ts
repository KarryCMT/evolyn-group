import type { CrossAppSourceApplication } from './crossApp.types';

/**
 * 跨应用设置的临时展示数据。它刻意与后端解耦，方便在表单资产接口到位后
 * 仅将数据源替换为 API 响应，而不改变页面的选择、搜索和暂存交互。
 */
export const crossAppDemoApplications: CrossAppSourceApplication[] = [
  {
    id: 'project',
    name: 'IT项目管理',
    tone: 'primary',
    groups: [
      {
        id: 'plan',
        name: '计划管理',
        forms: [
          { id: 'project-archive', name: '1.1 项目档案', kind: 'form' },
          { id: 'project-plan', name: '1.2 计划分解', kind: 'form' },
        ],
      },
      {
        id: 'progress',
        name: '进度管理',
        forms: [
          { id: 'task-follow-up', name: '2.1 任务跟进', kind: 'workflow-form' },
          { id: 'project-daily', name: '2.2 项目日报', kind: 'form' },
          { id: 'task-progress', name: '2.3 关键任务进度汇报', kind: 'workflow-form' },
          { id: 'milestone-progress', name: '2.4 里程碑进度汇报', kind: 'workflow-form' },
        ],
      },
      {
        id: 'change',
        name: '变更管理',
        forms: [{ id: 'project-change', name: '变更申请', kind: 'workflow-form' }],
      },
      {
        id: 'issue',
        name: '问题管理',
        forms: [{ id: 'project-issue', name: '问题跟踪', kind: 'form' }],
      },
      {
        id: 'meeting',
        name: '会议管理',
        forms: [{ id: 'project-meeting', name: '项目会议纪要', kind: 'form' }],
      },
      {
        id: 'template',
        name: '模板管理',
        forms: [{ id: 'project-template', name: '项目模板', kind: 'form' }],
      },
      {
        id: 'base',
        name: '基础数据',
        forms: [{ id: 'project-member', name: '项目成员', kind: 'form' }],
      },
    ],
  },
  {
    id: 'contract',
    name: '合同管理',
    tone: 'danger',
    groups: [
      {
        id: 'contract-main',
        name: '合同台账',
        forms: [
          { id: 'contract-register', name: '合同登记', kind: 'form' },
          { id: 'contract-payment', name: '付款计划', kind: 'workflow-form' },
        ],
      },
    ],
  },
  {
    id: 'task',
    name: '任务管理',
    tone: 'success',
    groups: [
      {
        id: 'task-main',
        name: '任务中心',
        forms: [
          { id: 'task-register', name: '任务清单', kind: 'form' },
          { id: 'task-review', name: '任务复盘', kind: 'form' },
        ],
      },
    ],
  },
  {
    id: 'guide',
    name: '简道云高级功能介绍',
    tone: 'info',
    groups: [
      {
        id: 'guide-main',
        name: '使用指南',
        forms: [{ id: 'guide-catalog', name: '功能目录', kind: 'form' }],
      },
    ],
  },
  {
    id: 'crm-cloud',
    name: 'CRM_云星辰',
    tone: 'danger',
    groups: [
      {
        id: 'crm-main',
        name: '客户管理',
        forms: [{ id: 'crm-customer', name: '客户档案', kind: 'form' }],
      },
    ],
  },
  {
    id: 'test',
    name: '测试应用',
    tone: 'warning',
    groups: [
      {
        id: 'test-main',
        name: '测试数据',
        forms: [{ id: 'test-form', name: '测试表单', kind: 'form' }],
      },
    ],
  },
];

/** 截图中的初始选择，用于前端效果还原；不代表任何已持久化配置。 */
export const initialCrossAppFormIDs = [
  'project-archive',
  'project-plan',
  'task-follow-up',
  'project-daily',
  'task-progress',
  'milestone-progress',
];
