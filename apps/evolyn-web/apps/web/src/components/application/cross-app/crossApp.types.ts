/** 跨应用页面的本地演示数据结构；后续接入接口时以同名领域 DTO 替换即可。 */
export type CrossAppFormKind = 'form' | 'workflow-form';

export interface CrossAppForm {
  id: string;
  name: string;
  kind: CrossAppFormKind;
}

export interface CrossAppFormGroup {
  id: string;
  name: string;
  forms: CrossAppForm[];
}

export interface CrossAppSourceApplication {
  id: string;
  name: string;
  tone: 'primary' | 'success' | 'warning' | 'danger' | 'info';
  groups: CrossAppFormGroup[];
}
