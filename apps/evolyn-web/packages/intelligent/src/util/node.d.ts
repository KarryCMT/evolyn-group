/** 旧画布节点工具的最小类型边界，避免组件继续退回隐式 any。 */
export interface LegacyNodeNameSource {
  componentName?: string;
  getModelName?: () => string;
  name?: string;
}

export function getNodeName(model: LegacyNodeNameSource): string;

export function focusNode(
  graphModel: object,
  id: string,
  deltaX?: number,
): void;
