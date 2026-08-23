import { VueNodeView } from '@logicflow/vue-node-registry';
import { createElement as h } from 'preact';

/**
 * VueNodeRegistry 默认 View 基于 HtmlNode，会额外绘制一层 SVG rect。
 * 工作流节点的视觉完全由 Vue 卡片承担，因此在自定义 View 中移除该底板，
 * 同时沿用 Registry 的 Vue 挂载和尺寸同步能力。
 */
export class WorkflowVueNodeView extends VueNodeView {
  override confirmUpdate(rootElement: SVGForeignObjectElement) {
    // Registry 默认仅在标题模式下刷新；工作流卡片需要响应名称和选中态变化。
    this.setHtml(rootElement);
  }

  override getShape() {
    const { model } = this.props;
    const { x, y, width, height } = model;

    // HtmlNode 依赖属性快照判断是否挂载或更新 Vue 内容。
    this.currentProperties = JSON.stringify(model.properties);

    return h('foreignObject', {
      x: x - width / 2,
      y: y - height / 2,
      width,
      height,
      ref: this.ref,
    });
  }
}
