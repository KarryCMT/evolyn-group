export interface IntelligentHistorySnapshot<T> {
  current?: T;
  canUndo: boolean;
  canRedo: boolean;
}

/** 有界不可变历史栈，替代旧版可变数组与外部回调。 */
export class IntelligentHistory<T> {
  private readonly limit: number;
  private stacks: T[] = [];
  private pointer = -1;

  constructor(limit = 50) {
    this.limit = Math.max(2, limit);
  }

  get snapshot(): IntelligentHistorySnapshot<T> {
    return {
      current: this.stacks[this.pointer],
      canUndo: this.pointer > 0,
      canRedo: this.pointer >= 0 && this.pointer < this.stacks.length - 1,
    };
  }

  reset(initial: T): void {
    this.stacks = [initial];
    this.pointer = 0;
  }

  add(value: T): IntelligentHistorySnapshot<T> {
    this.stacks = this.stacks.slice(0, this.pointer + 1);
    this.stacks.push(value);
    if (this.stacks.length > this.limit) this.stacks.shift();
    this.pointer = this.stacks.length - 1;
    return this.snapshot;
  }

  undo(): IntelligentHistorySnapshot<T> {
    if (this.snapshot.canUndo) this.pointer -= 1;
    return this.snapshot;
  }

  redo(): IntelligentHistorySnapshot<T> {
    if (this.snapshot.canRedo) this.pointer += 1;
    return this.snapshot;
  }
}
