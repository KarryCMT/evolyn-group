---
'@evolyn.do/ui': minor
---

新增 EvolynTable 数据表格组件：基于 VisActor VTable ListTable 的二次封装。数据更新分两级（records 走 setRecords 增量刷新、结构变化才全量 updateOption），主题运行时读取 --el-\* CSS 变量以对齐 Element Plus 明暗主题，VTable 事件以烤串命名转发，并经 options 逃生舱透传完整 ListTable 配置。
