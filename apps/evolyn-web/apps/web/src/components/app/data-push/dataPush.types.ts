/** 数据推送列表在接口落地前的前端展示模型。 */
export interface DataPushItem {
  id: string;
  name: string;
  serverAddress: string;
  formName: string;
  events: string;
  remark: string;
  enabled: boolean;
}
