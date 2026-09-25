import { HtmlNode, HtmlNodeModel } from '@logicflow/core';
import { createElement as preactH } from 'preact';
import { type App, createApp, h as vueH } from 'vue';
import type { IntelligentActionType, IntelligentNodeType } from '../schema';
import IntelligentBaseNode from './baseNode.vue';

export class IntelligentNodeModel extends HtmlNodeModel {
  override setAttributes(): void {
    this.width = Number(this.properties.width) || 286;
    this.height = Number(this.properties.height) || 88;
    this.text.editable = false;
  }
}

/** LogicFlow HTML 节点使用 Vue 3 应用实例渲染，生命周期与节点视图同步。 */
export class IntelligentNodeView extends HtmlNode {
  private vueApp: App<Element> | null = null;

  override setHtml(rootElement: SVGForeignObjectElement): void {
    this.unmountVueApp();
    rootElement.replaceChildren();
    const container = document.createElement('div');
    container.className = 'intelligent-node-content';
    rootElement.appendChild(container);
    const node = this.props.model as unknown as {
      properties?: {
        actionType?: IntelligentActionType;
        assistantType?: IntelligentNodeType;
        configured?: boolean;
        description?: string;
        label?: string;
        selected?: boolean;
      };
    };
    this.vueApp = createApp({
      render: () => vueH(IntelligentBaseNode, { node }),
    });
    this.vueApp.mount(container);
  }

  override confirmUpdate(rootElement: SVGForeignObjectElement): void {
    this.setHtml(rootElement);
  }

  override componentWillUnmount(): void {
    this.unmountVueApp();
    super.componentWillUnmount();
  }

  override getShape() {
    const { model } = this.props;
    const { x, y, width, height } = model;
    return preactH('foreignObject', {
      x: x - width / 2,
      y: y - height / 2,
      width,
      height,
      ref: this.ref,
    });
  }

  private unmountVueApp(): void {
    this.vueApp?.unmount();
    this.vueApp = null;
  }
}
