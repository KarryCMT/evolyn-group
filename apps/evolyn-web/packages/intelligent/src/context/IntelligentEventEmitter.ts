type EventHandler<T = unknown> = (payload: T) => void;

/** 轻量类型化事件中心，用于解耦画布、菜单和属性面板。 */
export class IntelligentEventEmitter<EventMap extends object> {
  private readonly events = new Map<keyof EventMap, Set<EventHandler>>();

  on<Key extends keyof EventMap>(event: Key, handler: EventHandler<EventMap[Key]>): () => void {
    const handlers = this.events.get(event) ?? new Set<EventHandler>();
    handlers.add(handler as EventHandler);
    this.events.set(event, handlers);
    return () => this.off(event, handler);
  }

  off<Key extends keyof EventMap>(event: Key, handler: EventHandler<EventMap[Key]>): void {
    const handlers = this.events.get(event);
    handlers?.delete(handler as EventHandler);
    if (handlers?.size === 0) this.events.delete(event);
  }

  emit<Key extends keyof EventMap>(event: Key, payload: EventMap[Key]): void {
    this.events.get(event)?.forEach((handler) => handler(payload));
  }

  clear(): void {
    this.events.clear();
  }
}
