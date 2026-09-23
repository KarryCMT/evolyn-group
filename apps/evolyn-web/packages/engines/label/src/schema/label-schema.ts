export type LabelUnit = 'mm' | 'px';
export type LabelDpi = 96 | 203 | 300 | 600;

export interface LabelSchema {
  schemaVersion: '1.0';
  id?: string;
  name: string;
  page: {
    width: number;
    height: number;
    unit: LabelUnit;
    dpi: LabelDpi;
    background: string;
  };
  source?: {
    type: 'form';
    appId?: string;
    formId?: string;
  };
  elements: LabelElement[];
  settings: {
    snapToGrid: boolean;
    gridSize: number;
    showGrid: boolean;
  };
}

export interface BaseElement {
  id: string;
  type: string;
  name?: string;
  x: number;
  y: number;
  width: number;
  height: number;
  rotation: number;
  zIndex: number;
  visible: boolean;
  locked: boolean;
  opacity?: number;
}

export type ValueSource =
  | { type: 'static'; value: string }
  | { type: 'field'; fieldId: string }
  | {
      type: 'system';
      key: 'recordId' | 'createdAt' | 'updatedAt' | 'createdBy' | 'currentUser' | 'currentDate';
    }
  | { type: 'expression'; expression: string };

export interface TextStyle {
  fontFamily: string;
  fontSize: number;
  fontWeight: number;
  color: string;
  lineHeight: number;
  textAlign: 'left' | 'center' | 'right';
  verticalAlign: 'top' | 'middle' | 'bottom';
  overflow: 'clip' | 'ellipsis' | 'wrap';
}

export interface TextElement extends BaseElement {
  type: 'text';
  value: ValueSource;
  style: TextStyle;
}

export interface FieldElement extends BaseElement {
  type: 'field';
  label?: string;
  value: ValueSource;
  separator?: string;
  formatter?: { type?: 'none' | 'date' | 'number'; pattern?: string };
  style: TextStyle;
}

export interface QRCodeElement extends BaseElement {
  type: 'qrcode';
  value: ValueSource;
  options: {
    errorCorrection: 'L' | 'M' | 'Q' | 'H';
    quietZone: number;
    foreground: string;
    background: string;
  };
}

export interface ImageElement extends BaseElement {
  type: 'image';
  value: ValueSource;
  objectFit: 'contain' | 'cover';
}

export interface RectElement extends BaseElement {
  type: 'rect';
  fill: string;
  stroke?: string;
  strokeWidth?: number;
  radius?: number;
}

export interface LineElement extends BaseElement {
  type: 'line';
  color: string;
  strokeWidth: number;
}

export type LabelElement =
  | TextElement
  | FieldElement
  | QRCodeElement
  | ImageElement
  | RectElement
  | LineElement;

export interface LabelRenderData {
  fields: Record<string, unknown>;
  system: Partial<Record<Extract<ValueSource, { type: 'system' }>['key'], unknown>>;
}
