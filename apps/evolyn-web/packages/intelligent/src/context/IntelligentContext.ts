import type { IntelligentActionType, IntelligentDocument } from '../schema';
import { IntelligentEventEmitter } from './IntelligentEventEmitter';
import { IntelligentHistory } from './IntelligentHistory';

export interface IntelligentContextEvents {
  'document:change': IntelligentDocument;
  'node:add': IntelligentActionType;
  'node:select': string | null;
  'panel:close': undefined;
}

/** 智能助手编辑上下文：集中承载跨组件事件与操作历史。 */
export class IntelligentContext {
  readonly events = new IntelligentEventEmitter<IntelligentContextEvents>();
  readonly history = new IntelligentHistory<IntelligentDocument>();

  constructor(document: IntelligentDocument) {
    this.history.reset(document);
  }

  destroy(): void {
    this.events.clear();
  }
}
