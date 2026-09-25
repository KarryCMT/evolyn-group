import { PolylineEdge, PolylineEdgeModel } from '@logicflow/core';
import { createElement as preactH } from 'preact';
import { type App, createApp, h as vueH } from 'vue';
import IntelligentEdgeInsertControl from './edgeInsertControl.vue';

export class IntelligentEdgeModel extends PolylineEdgeModel {
  override getEdgeStyle() {
    return {
      ...super.getEdgeStyle(),
      stroke: this.isHovered || this.isSelected ? '#2f7cf6' : '#cfd6e0',
      strokeWidth: this.isHovered || this.isSelected ? 2 : 1.6,
    };
  }
}

/** 在线段中点挂载 Vue 3 控件，点击事件继续冒泡给 LogicFlow 的 edge:click。 */
export class IntelligentEdgeView extends PolylineEdge {
  private vueApp: App<Element> | null = null;
  private mountFrame = 0;

  override getText() {
    const { id, textPosition } = this.props.model;
    const containerId = `intelligent-edge-insert-${id}`;
    cancelAnimationFrame(this.mountFrame);
    this.mountFrame = requestAnimationFrame(() => {
      const container = document.getElementById(containerId);
      if (!container) return;
      this.unmountVueApp();
      this.vueApp = createApp({
        render: () => vueH(IntelligentEdgeInsertControl),
      });
      this.vueApp.mount(container);
    });

    return preactH(
      'foreignObject',
      {
        x: textPosition.x - 16,
        y: textPosition.y - 16,
        width: 32,
        height: 32,
        className: 'intelligent-edge-insert',
      },
      preactH('div', {
        id: containerId,
        className: 'intelligent-edge-insert__mount',
      }),
    );
  }

  override componentWillUnmount(): void {
    cancelAnimationFrame(this.mountFrame);
    this.unmountVueApp();
  }

  private unmountVueApp(): void {
    this.vueApp?.unmount();
    this.vueApp = null;
  }
}
