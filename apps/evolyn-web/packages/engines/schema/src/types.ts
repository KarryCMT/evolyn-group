/** 可安全持久化和复制的 JSON 值；领域 Schema 可在其上定义更窄的协议。 */
export type JsonValue =
  | string
  | number
  | boolean
  | null
  | JsonValue[]
  | { [key: string]: JsonValue };
