/** 富文本图片上传函数；组件不感知表单、流程等业务文件归属。 */
export type EvolynRichTextImageUploader = (file: File) => Promise<string>;

/** 工具栏按钮的激活状态。 */
export interface EvolynRichTextToolbarState {
  bold: boolean;
  italic: boolean;
  underline: boolean;
  alignLeft: boolean;
  link: boolean;
  color?: string;
}

/** 工具栏发出的格式操作。 */
export type EvolynRichTextFormatCommand =
  | 'bold'
  | 'italic'
  | 'underline'
  | 'alignLeft'
  | 'clear'
  | 'link'
  | 'unlink';

export interface EvolynRichTextEditorProps {
  /** 是否允许编辑；只读时仍展示已格式化内容。 */
  editable?: boolean;
  /** 编辑区域最小高度，数字按 px 处理。 */
  minHeight?: number | string;
  /** 图片上传函数，应返回已完成权限校验的图片地址。未提供时禁用图片按钮。 */
  uploadImage?: EvolynRichTextImageUploader;
  /** 可选择的图片 MIME 类型。 */
  imageAccept?: string[];
  /** 允许上传的单张图片最大字节数，默认 10MB。 */
  maxImageSize?: number;
  /** 编辑区域的无障碍名称。 */
  ariaLabel?: string;
}

export interface EvolynRichTextEditorEmits {
  /** 用户编辑后输出 HTML；服务端保存和展示前仍须执行白名单净化。 */
  change: [html: string];
  /** 图片已上传并插入编辑器。 */
  imageUpload: [file: File, url: string];
  /** 图片选择或上传失败。 */
  imageUploadError: [message: string];
}
