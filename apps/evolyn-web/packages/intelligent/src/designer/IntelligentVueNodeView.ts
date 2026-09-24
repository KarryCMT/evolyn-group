import { HtmlNode, HtmlNodeModel } from '@logicflow/core';
import { createElement as preactH } from 'preact';
import { type App, createApp, h as vueH } from 'vue';
import IntelligentNodeCard from './IntelligentNodeCard.vue';

/** 固定节点尺寸由文档投影决定，避免缩放后 DOM 测量反向污染画布模型。 */
export class IntelligentNodeModel extends HtmlNodeModel {
  override setAttributes(): void {
    this.width = Number(this.properties.width) || 286;
    this.height = Number(this.properties.height) || 88;
    this.text.editable = false;
  }
}

/**
 * 使用 Vue 3 createApp 挂载节点组件，并显式管理每个节点应用实例的生命周期。
 * 每个节点视图持有独立应用实例，并在刷新或销毁时显式卸载。
 */
export class IntelligentVueNodeView extends HtmlNode {
  private vueApp: App<Element> | null = null;

  override setHtml(rootElement: SVGForeignObjectElement): void {
    this.unmountVueApp();
    rootElement.replaceChildren();
    const container = document.createElement('div');
    container.className = 'intelligent-vue-node-content';
    rootElement.appendChild(container);
    const node = this.props.model as unknown as {
      properties?: {
        assistantType?: 'trigger' | 'action' | 'end';
        label?: string;
        description?: string;
        selected?: boolean;
      };
    };
    this.vueApp = createApp({
      render: () => vueH(IntelligentNodeCard, { node }),
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
