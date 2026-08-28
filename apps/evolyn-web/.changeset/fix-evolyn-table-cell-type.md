---
'@evolyn.do/ui': patch
---

修正 EvolynTable 文本单元格类型契约，移除 VTable 不支持的 `multilinetext` 类型，避免表格挂载时创建未注册单元格而报错。
