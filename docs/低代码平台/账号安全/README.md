# 账号安全与会话体系（专题）

M2 前置安全能力专题：「登录二次验证（TOTP MFA）」与「禁止同时登录」的
完整落地方案——服务端会话实体（account_sessions）、挑战式 MFA 登录流、
单会话挤出与恢复码体系。

## 子文档

- [账号安全与会话体系设计方案.md](./账号安全与会话体系设计方案.md)：方案定版（数据模型/登录流/接口/落地顺序）
- [账号安全与会话体系-任务清单.md](./账号安全与会话体系-任务清单.md)：分批任务跟踪（ADR-009）

## 关联

- 架构设计文档第 27 章 ADR-009（会话与 MFA 定版）
- 前端入口：`apps/evolyn-web/apps/web/src/components/dashboard/account/AccountSecurityPanel.vue`
